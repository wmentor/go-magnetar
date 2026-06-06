package chat

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/sashabaranov/go-openai"
	"github.com/wmentor/go-magnetar/internal/agent/summarizer"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

const systemPrompt = `You are a helpful assistant that answers questions strictly based on the knowledge base.

Rules:
- If you need information to answer a question, or need to verify or clarify what you already know, use the rag_search tool before responding.
- Base your answers only on the information retrieved from rag_search. Do not invent, assume, or extrapolate facts.
- If rag_search returns no relevant results, tell the user honestly that you don't have information on this topic.
- You may call rag_search multiple times with different queries if needed.
- Be concise and precise.`

// charsPerToken is a rough approximation used for context-window budget estimation.
// OpenAI models average ~4 UTF-8 characters per token.
const charsPerToken = 4

// ChatAgent is an interactive REPL-based chat agent backed by a RAG knowledge base.
type ChatAgent struct {
	cfg        *config.Config
	llm        *openai.Client
	rag        *rag.RAGTools
	summarizer *summarizer.Summarizer
	renderer   *glamour.TermRenderer
	messages   []openai.ChatCompletionMessage
}

// New creates a new ChatAgent instance.
func New(cfg *config.Config) (*ChatAgent, error) {
	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	ragTools, err := rag.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise RAG tools: %w", err)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise markdown renderer: %w", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	return &ChatAgent{
		cfg:        cfg,
		llm:        llmClient,
		rag:        ragTools,
		summarizer: summarizer.New(cfg),
		renderer:   renderer,
		messages:   messages,
	}, nil
}

// estimateTokens returns a rough token count for a single message.
// It sums the lengths of Role and Content and divides by charsPerToken.
func estimateTokens(m openai.ChatCompletionMessage) int {
	chars := len(m.Role) + len(m.Content)
	for _, tc := range m.ToolCalls {
		chars += len(tc.Function.Name) + len(tc.Function.Arguments)
	}
	t := chars / charsPerToken
	if t < 1 {
		t = 1
	}
	return t
}

// trimMessages returns a copy of a.messages that fits within the context window.
// It always keeps the system message (index 0) and then fills from the most recent
// messages backwards, leaving reservedOutputTokens tokens for the model reply.
func (a *ChatAgent) trimMessages(reservedOutputTokens int) []openai.ChatCompletionMessage {
	if a.cfg.LLM.Context <= 0 {
		// No limit configured — return as-is.
		return a.messages
	}

	budget := a.cfg.LLM.Context - reservedOutputTokens
	if budget <= 0 {
		budget = a.cfg.LLM.Context
	}

	system := a.messages[0]
	budget -= estimateTokens(system)

	// Walk from the newest message backwards and collect what fits.
	tail := a.messages[1:]
	keep := make([]openai.ChatCompletionMessage, 0, len(tail))
	for i := len(tail) - 1; i >= 0; i-- {
		cost := estimateTokens(tail[i])
		if budget-cost < 0 {
			break
		}
		budget -= cost
		keep = append(keep, tail[i])
	}

	// Reverse keep so messages are in chronological order.
	for l, r := 0, len(keep)-1; l < r; l, r = l+1, r-1 {
		keep[l], keep[r] = keep[r], keep[l]
	}

	trimmed := make([]openai.ChatCompletionMessage, 0, 1+len(keep))
	trimmed = append(trimmed, system)
	trimmed = append(trimmed, keep...)

	if dropped := len(tail) - len(keep); dropped > 0 {
		slog.Debug("chat: trimmed message history to fit context window",
			"dropped", dropped, "kept", len(keep))
	}

	return trimmed
}

// reservedOutputTokens is the share of the context budget reserved for the model's reply.
// Using 20 % of the configured context limit as a reasonable default.
const reservedOutputFraction = 5 // 1/5 = 20 %

// compactIfNeeded checks whether the message history has reached the compaction
// threshold and, if so, replaces a.messages with a summarized version.
func (a *ChatAgent) compactIfNeeded() {
	if !a.summarizer.NeedsCompaction(a.messages) {
		return
	}
	compacted, err := a.summarizer.Compact(a.messages)
	if err != nil {
		slog.Error("chat: history compaction failed, continuing with full history", "err", err)
		return
	}
	a.messages = compacted
}

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *ChatAgent) ask(userInput string) (string, error) {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	// Compact history before sending to the LLM if the threshold is reached.
	a.compactIfNeeded()

	tools := []openai.Tool{
		a.rag.DefinitionSearch(),
	}

	for {
		reserved := 0
		if a.cfg.LLM.Context > 0 {
			reserved = a.cfg.LLM.Context / reservedOutputFraction
		}
		trimmed := a.trimMessages(reserved)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    a.cfg.LLM.Model,
			Messages: trimmed,
			Tools:    tools,
		})
		cancel()

		if err != nil {
			return "", fmt.Errorf("chat: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("chat: empty response from LLM")
		}

		choice := resp.Choices[0]
		// Append assistant message to history.
		a.messages = append(a.messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonToolCalls {
			for _, toolCall := range choice.Message.ToolCalls {
				name := toolCall.Function.Name
				args := toolCall.Function.Arguments

				slog.Debug("chat: tool call", "tool", name, "args", args)

				result := a.rag.Dispatch(name, args)

				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}
			// Continue the loop with tool results.
			continue
		}

		// Final answer.
		return choice.Message.Content, nil
	}
}

// Run starts the interactive REPL loop, reading from stdin and writing to stdout.
func (a *ChatAgent) Run() error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			// EOF or error.
			fmt.Println()
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		answer, err := a.ask(line)
		if err != nil {
			slog.Error("chat: failed to get answer", "err", err)
			fmt.Println("Error: failed to get response. Please try again.")
			continue
		}

		rendered, err := a.renderer.Render(answer)
		if err != nil {
			// Fallback to plain text if rendering fails.
			fmt.Println(answer)
			fmt.Println()
		} else {
			fmt.Print(rendered)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("chat: stdin read error: %w", err)
	}

	return nil
}
