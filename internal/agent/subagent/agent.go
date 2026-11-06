package subagent

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/gmutex"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const (
	maxSearchToolCalls  = 20
	noProgressCountMax  = 4
	maxAskLoopIteration = 50

	// charsPerToken is a rough approximation used for context-window budget estimation.
	// OpenAI models average ~4 UTF-8 characters per token.
	charsPerToken = 4

	// reservedOutputFraction is the share of context budget reserved for the model reply (20%).
	reservedOutputFraction = 5
)

// Agent is a subagent backed.
type Agent struct {
	name      string
	cfg       *config.Config
	llm       *openai.Client
	messages  []openai.ChatCompletionMessage
	root      *os.Root
	toolStore *tools.Store
	printer   printer.Printer
}

// New creates a new ChatAgent instance.
func New(agentName string, state *plugin.State) (*Agent, error) {
	a, err := state.AgentStore.Get(agentName)
	if err != nil {
		return nil, fmt.Errorf("get agent %q error: %w", agentName, err)
	}

	availableTools := slices.Clone(a.AvailableTools)
	for _, subAgent := range a.Subagents {
		_, err := state.AgentStore.Get(subAgent)
		if err != nil {
			return nil, fmt.Errorf("get subagent %q error: %w", subAgent, err)
		}
		availableTools = append(availableTools, "agent_"+agent.NormalizeAgentName(subAgent))
	}

	toolStore, err := state.ToolStore.CloneWith(a.AvailableTools...)
	if err != nil {
		return nil, errors.Wrap(err, "init tool store")
	}

	cfg := state.Config.Copy()

	if a.Profile != "" {
		if err = cfg.SetProfile(a.Profile); err != nil {
			return nil, fmt.Errorf("set profile %q error: %w", a.Profile, err)
		}
	}

	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	llmCfg.HTTPClient = state.RequesterFactory.Get()
	llmClient := openai.NewClientWithConfig(llmCfg)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: a.Prompt,
		},
	}

	return &Agent{
		name:      agentName,
		cfg:       cfg,
		llm:       llmClient,
		messages:  messages,
		root:      state.Root,
		toolStore: toolStore,
		printer:   state.Printer,
	}, nil
}

// estimateTokens returns a rough token count for a single message.
func estimateTokens(m openai.ChatCompletionMessage) int {
	chars := len(m.Role) + len(m.Content)
	for _, tc := range m.ToolCalls {
		chars += len(tc.Function.Name) + len(tc.Function.Arguments)
	}
	t := max(chars/charsPerToken, 1)
	return t
}

// trimMessages returns a copy of a.messages that fits within the context window.
func (a *Agent) trimMessages(reservedOutputTokens int) []openai.ChatCompletionMessage {
	if a.cfg.ProfileParamInt("llm.context") <= 0 {
		return a.messages
	}

	budget := a.cfg.ProfileParamInt("llm.context") - reservedOutputTokens
	if budget <= 0 {
		budget = a.cfg.ProfileParamInt("llm.context")
	}

	system := a.messages[0]
	budget -= estimateTokens(system)

	tail := a.messages[1:]
	keep := make([]openai.ChatCompletionMessage, 0, len(tail))
	for _, t := range slices.Backward(tail) {
		cost := estimateTokens(t)
		if budget-cost < 0 {
			break
		}
		budget -= cost
		keep = append(keep, t)
	}

	for l, r := 0, len(keep)-1; l < r; l, r = l+1, r-1 {
		keep[l], keep[r] = keep[r], keep[l]
	}

	trimmed := make([]openai.ChatCompletionMessage, 0, 1+len(keep))
	trimmed = append(trimmed, system)
	trimmed = append(trimmed, keep...)

	if dropped := len(tail) - len(keep); dropped > 0 {
		a.printer.Debug("agent: trimmed message history to fit context window",
			"dropped", dropped, "kept", len(keep))
	}

	return trimmed
}

type toolRecord struct {
	Name string
	Args string
}

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *Agent) Ask(input string) (string, error) {
	a.printer.ToolCall(printer.IconRobot, "run subagent "+a.name)

	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})

	beginIndex := len(a.messages)

	// Build tool list and dispatch map from registered plugins.
	tools := make([]openai.Tool, 0, 32)

	for _, t := range a.toolStore.Tools() {
		def := t.Definition()
		tools = append(tools, def)
	}

	noProgressCount := 0

	lastAnswerContent := ""

	alreadyDoneToolCalls := map[toolRecord]struct{}{}
	allToolCallStat := map[string]int{}

	var agentCounter atomic.Int64

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	for range maxAskLoopIteration {
		reserved := 0
		if a.cfg.ProfileParamInt("llm.context") > 0 {
			reserved = a.cfg.ProfileParamInt("llm.context") / reservedOutputFraction
		}
		trimmed := a.trimMessages(reserved)

		// When the search tool call limit has been reached, send the request
		// without any tools so the LLM is forced to produce a final text answer
		// instead of attempting further tool calls (which would loop forever).
		activeTools := tools

		resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:           a.cfg.ProfileParamString("llm.model"),
			Messages:        trimmed,
			Tools:           activeTools,
			Temperature:     float32(a.cfg.ProfileParamFloat64("llm.temperature")),
			TopP:            float32(a.cfg.ProfileParamFloat64("llm.top_p")),
			ReasoningEffort: a.cfg.ProfileParamString("llm.reasoning_effort"),
		})
		if err != nil {
			return "", fmt.Errorf("agent: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("agent: empty response from LLM")
		}

		choice := resp.Choices[0]
		a.messages = append(a.messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonLength {
			a.printer.Error("agent: LLM response was truncated (finish_reason=length); the answer may be incomplete")
			// Return whatever content we have; continuing would just repeat the same truncated response
			return choice.Message.Content, nil
		}

		if choice.FinishReason == openai.FinishReasonToolCalls { //nolint:nestif // later.
			type toolResult struct {
				toolCall openai.ToolCall
				result   string
				err      error
			}

			results := make(chan toolResult, len(choice.Message.ToolCalls))

			var wg sync.WaitGroup

			// executeTool runs a single tool call and sends the result to results.
			executeTool := func(tc openai.ToolCall) {
				name := tc.Function.Name
				args := tc.Function.Arguments

				var result string
				if tool, err := a.toolStore.GetTool(name); err == nil {
					if !tool.IsAgent {
						if tool.Sequential {
							gmutex.Lock()
							defer gmutex.Unlock()
						} else {
							gmutex.RLock()
							defer gmutex.RUnlock()
						}
					} else {
						if err := agent.WaitCanRun(ctx, &agentCounter, false); err != nil {
							result = fmt.Sprintf("run agent error: %s", err.Error())
							results <- toolResult{toolCall: tc, result: result}
							return
						}
						defer func() {
							agentCounter.Add(-1)
						}()
					}
					result = tool.Execute(context.Background(), []byte(args))
				} else {
					result = "error: unknown tool " + name
				}

				results <- toolResult{toolCall: tc, result: result}
			}

			// Split tool calls into parallel and sequential buckets.
			// Sequential tools (e.g. file_edit, file_write) must not run
			// concurrently because they mutate shared state (the filesystem).
			var tools []openai.ToolCall

			for _, toolCall := range choice.Message.ToolCalls {
				toolKey := toolRecord{Name: toolCall.Function.Name, Args: toolCall.Function.Arguments}
				if _, has := alreadyDoneToolCalls[toolKey]; has {
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "error: this tool was called earlier with the same parameters",
						ToolCallID: toolCall.ID,
					})
					a.printer.Warn("llm require double tool call", "tool", toolKey.Name, "args", toolKey.Args)
					continue
				}
				alreadyDoneToolCalls[toolKey] = struct{}{}

				tool, err := a.toolStore.GetTool(toolCall.Function.Name)
				if err != nil {
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "error: unknown tool " + toolCall.Function.Name,
						ToolCallID: toolCall.ID,
					})
					a.printer.Error("unknown tool call", "tool", toolKey.Name, "args", toolKey.Args)
					continue
				}
				allToolCallStat[tool.Name]++
				if tool.RateLimit > 0 && tool.RateLimit < allToolCallStat[tool.Name] {
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "error: tool call limit reached",
						ToolCallID: toolCall.ID,
					})
					a.printer.Warn("tool call limit reached", "tool", toolKey.Name, "args", toolKey.Args)
					continue
				}

				tools = append(tools, toolCall)
			}

			noProgressCount++

			// Run parallel tools concurrently.
			for _, tc := range tools {
				noProgressCount = 0
				wg.Add(1)
				go func(tc openai.ToolCall) {
					defer wg.Done()
					executeTool(tc)
				}(tc)
			}
			wg.Wait()

			close(results)

			for r := range results {
				message := r.result
				if r.err != nil {
					a.printer.Error("chat: tool execution failed", "tool", r.toolCall.Function.Name, "err", r.err)
					message = r.err.Error()
				}

				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    message,
					ToolCallID: r.toolCall.ID,
				})
			}

			lastAnswerContent = choice.Message.Content

			if noProgressCount >= noProgressCountMax {
				a.printer.Warn("broke potential infinite loop")
				if beginIndex < len(a.messages) {
					a.messages = a.messages[:beginIndex]
				}
				return lastAnswerContent, nil
			}

			continue
		}

		return choice.Message.Content, nil
	}

	a.printer.Warn("broke potential infinite loop")
	if beginIndex < len(a.messages) {
		a.messages = a.messages[:beginIndex]
	}
	return lastAnswerContent, nil
}
