package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/term"

	"github.com/wmentor/go-magnetar/internal/agent/summarizer"
	"github.com/wmentor/go-magnetar/internal/config"
	hstore "github.com/wmentor/go-magnetar/internal/history"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const systemPromptTemplate = `You are a helpful assistant that answers questions strictly based on the knowledge base.

Parameters:
- language: %s

Rules:
- Always try rag_search first for every question, even if you think you already know the answer.
- If rag_search returns relevant results, base your answer exclusively on those results. Do not use web_fetch in this case.
- You may call rag_search multiple times with different queries to gather all necessary information.
- Only use web_fetch if rag_search returned no relevant results and the user explicitly needs up-to-date or external information.
- Do not invent, assume, or extrapolate facts beyond what the tools return.
- If neither rag_search nor web_fetch provides relevant information, tell the user honestly that you don't have information on this topic.
- Be concise and precise.
- You may execute multiple tools in parallel when the response contains multiple tool calls.

SECURITY WARNING: You are FORBIDDEN from executing destructive commands such as:
- rm -rf /, rm -rf *, sudo, chmod 777, dd, mkfs, fdisk, or any command that could delete/modify/system files.
- curl ... | bash, wget ... | sh, zsh, or any pipe to shell execution.

You are FORBIDDEN from using heredoc syntax to write files:
- Never execute commands like cat > /tmp/params.txt << 'EOF', cat << EOF, or similar heredoc patterns.
- This includes any command that writes content to files using heredoc redirection.

You are FORBIDDEN from downloading and executing untrusted scripts:
- Never execute curl ... | bash, wget ... -O - | sh, or similar patterns.
- Always verify script sources and contents before execution.

IMPORTANT: The exec tool will BLOCK any command matching security patterns (rm -rf, sudo, mkfs, dd with /dev/, pipe to bash/sh/zsh, git commit/push/rebase/pull/cherry-pick/reset/stash/clean/reflog). DO NOT attempt to bypass these protections.

SECURITY CHECK: When the guard tool analyzes a command for safety:
- If the guard is uncertain about whether a command is safe, ALWAYS reject it (allow=false).
- Err on the side of caution - better to block a safe command than to allow a dangerous one.
`

// charsPerToken is a rough approximation used for context-window budget estimation.
// OpenAI models average ~4 UTF-8 characters per token.
const charsPerToken = 4

const agentsFile = "AGENTS.md"

// ChatAgent is an interactive REPL-based chat agent backed by a RAG knowledge base.
type ChatAgent struct {
	cfg        *config.Config
	llm        *openai.Client
	summarizer *summarizer.Summarizer
	renderer   *glamour.TermRenderer
	messages   []openai.ChatCompletionMessage
	root       *os.Root
}

// New creates a new ChatAgent instance.
func New(cfg *config.Config, root *os.Root) (*ChatAgent, error) {
	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	llmClient := openai.NewClientWithConfig(llmCfg)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise markdown renderer: %w", err)
	}

	language := cfg.String("language")
	systemContent := fmt.Sprintf(systemPromptTemplate, language)
	if root != nil {
		if _, statErr := root.Stat(agentsFile); statErr == nil {
			if content, err := root.ReadFile(agentsFile); err == nil {
				systemContent = systemContent + "\n\n# Project context (from AGENTS.md)\n\n" + string(content)
				printer.ToolCall(printer.IconDone, "AGENTS.md has been loaded")
				printer.ToolEmptyLine()
			}
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemContent,
		},
	}

	return &ChatAgent{
		cfg:        cfg,
		llm:        llmClient,
		summarizer: summarizer.New(cfg),
		renderer:   renderer,
		messages:   messages,
		root:       root,
	}, nil
}

// --- AgentHandle implementation ---

// agentHandle is a private adapter that exposes ChatAgent internals to chat
// command plugins through the plugin.AgentHandle interface.
type agentHandle struct {
	a *ChatAgent
}

func (h *agentHandle) Messages() []openai.ChatCompletionMessage {
	out := make([]openai.ChatCompletionMessage, len(h.a.messages))
	copy(out, h.a.messages)
	return out
}

func (h *agentHandle) SetMessages(msgs []openai.ChatCompletionMessage) {
	h.a.messages = msgs
}

func (h *agentHandle) Config() *config.Config {
	return h.a.cfg
}

func (h *agentHandle) Compact() error {
	compacted, err := h.a.summarizer.Compact(h.a.messages)
	if err != nil {
		return err
	}
	h.a.messages = compacted
	return nil
}

func (h *agentHandle) Reset() {
	printer.Info("chat: starting new session, clearing context")
	h.a.messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: h.a.messages[0].Content, // preserve the current system prompt
		},
	}
}

// --- Token / context helpers ---

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
func (a *ChatAgent) trimMessages(reservedOutputTokens int) []openai.ChatCompletionMessage {
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
		printer.Debug("chat: trimmed message history to fit context window",
			"dropped", dropped, "kept", len(keep))
	}

	return trimmed
}

// reservedOutputFraction is the share of context budget reserved for the model reply (20%).
const reservedOutputFraction = 5

// compactIfNeeded checks whether the message history has reached the compaction
// threshold and, if so, replaces a.messages with a summarized version.
func (a *ChatAgent) compactIfNeeded() {
	if !a.summarizer.NeedsCompaction(a.messages) {
		return
	}
	compacted, err := a.summarizer.Compact(a.messages)
	if err != nil {
		printer.Error("chat: history compaction failed, continuing with full history", "err", err)
		return
	}
	a.messages = compacted
}

// maxSearchToolCalls is the maximum number of search-related tool calls per user request.
const maxSearchToolCalls = 10

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *ChatAgent) Ask(userInput string) (string, error) {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	a.compactIfNeeded()

	// Build tool list and dispatch map from registered plugins.
	llmTools := plugin.LLMTools()
	tools := make([]openai.Tool, 0, len(llmTools))
	toolMap := make(map[string]plugin.LLMTool, len(llmTools))
	for _, t := range llmTools {
		def := t.Definition()
		tools = append(tools, def)
		toolMap[def.Function.Name] = t
	}

	toolCallCount := 0

	for {
		reserved := 0
		if a.cfg.ProfileParamInt("llm.context") > 0 {
			reserved = a.cfg.ProfileParamInt("llm.context") / reservedOutputFraction
		}
		trimmed := a.trimMessages(reserved)

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
		resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:           a.cfg.ProfileParamString("llm.model"),
			Messages:        trimmed,
			Tools:           tools,
			Temperature:     float32(a.cfg.ProfileParamFloat64("llm.temperature")),
			TopP:            float32(a.cfg.ProfileParamFloat64("llm.top_p")),
			ReasoningEffort: a.cfg.ProfileParamString("llm.reasoning_effort"),
		})
		cancel()

		if err != nil {
			return "", fmt.Errorf("chat: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("chat: empty response from LLM")
		}

		choice := resp.Choices[0]
		a.messages = append(a.messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonToolCalls {
			type toolResult struct {
				toolCall openai.ToolCall
				result   string
				err      error
			}

			results := make(chan toolResult, len(choice.Message.ToolCalls))

			var wg sync.WaitGroup

			for _, toolCall := range choice.Message.ToolCalls {
				wg.Add(1)
				go func(tc openai.ToolCall) {
					defer wg.Done()

					name := tc.Function.Name
					args := tc.Function.Arguments

					var result string
					t, ok := toolMap[name]
					if !ok {
						result = "error: unknown tool " + name
					} else {
						if t.IsSearchTool {
							toolCallCount++
							if toolCallCount > maxSearchToolCalls {
								result = fmt.Sprintf("error: reached maximum number of search tool calls (%d)", maxSearchToolCalls)
							} else {
								r, err := t.Execute(context.Background(), args)
								if err != nil {
									result = "error: " + err.Error()
								} else {
									result = r
								}
							}
						} else {
							r, err := t.Execute(context.Background(), args)
							if err != nil {
								result = "error: " + err.Error()
							} else {
								result = r
							}
						}
					}
					results <- toolResult{toolCall: tc, result: result}
				}(toolCall)
			}

			wg.Wait()
			close(results)

			for r := range results {
				if r.err != nil {
					printer.Error("chat: tool execution failed", "tool", r.toolCall.Function.Name, "err", r.err)
				}
				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    r.result,
					ToolCallID: r.toolCall.ID,
				})
			}
			continue
		}

		return choice.Message.Content, nil
	}
}

// --- Input / REPL ---

var (
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Bold(true)
	promptSymbol = promptStyle.Render("👨")

	questionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("69")).
			PaddingLeft(1).
			Faint(false)

	history *hstore.Storage

	historySizeLimit = 200
	historyFile      = func() string {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".go-magnetar-history.json")
	}()
)

func loadHistory() {
	history = hstore.New(historyFile, historySizeLimit)
}

// inputModel is a minimal bubbletea model used solely to collect one line of
// user input with a styled prompt.
type inputModel struct {
	input textinput.Model
	done  bool
	quit  bool
}

var _ = promptStyle
var _ = promptSymbol
var _ = questionStyle

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80
	}
	if width <= 0 {
		return 80
	}
	return width
}

func newInputModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "Ask anything... (Enter to submit)"
	ti.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	ti.Prompt = promptSymbol + " "
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = getTerminalWidth() - len(promptSymbol) - 2
	return inputModel{input: ti}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			text := m.input.Value()
			if text != "" {
				history.Add(text)
			}
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.quit = true
			return m, tea.Quit
		case tea.KeyUp:
			if record := history.Prev(); record != "" {
				m.input.SetValue(record)
				m.input.CursorEnd()
			}
			return m, nil
		case tea.KeyDown:
			if record := history.Next(); record != "" {
				m.input.SetValue(record)
				m.input.CursorEnd()
			} else {
				m.input.SetValue("")
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *inputModel) View() string {
	return m.input.View()
}

// readInput launches a bubbletea program to collect one line from the user.
func readInput() (string, bool, error) {
	m := newInputModel()
	p := tea.NewProgram(&m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", false, err
	}
	final := result.(*inputModel)
	fmt.Fprintln(os.Stdout)
	return strings.TrimSpace(final.input.Value()), final.quit, nil
}

// handleCommand processes a slash-command entered by the user.
// Returns (handled, exit).
func (a *ChatAgent) handleCommand(line string) (handled bool, exit bool) {
	// Strip leading "/" and split into name + args.
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "/")
	parts := strings.SplitN(trimmed, " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) == 2 {
		args = strings.TrimSpace(parts[1])
	}

	handle := &agentHandle{a: a}

	for _, cmd := range plugin.ChatCommands() {
		if matchCommand(name, cmd) {
			err := cmd.Execute(context.Background(), handle, args)
			if err != nil {
				if errors.Is(err, plugin.ErrExit) {
					return true, true
				}
				printer.Error("chat: command error", "cmd", cmd.Name, "err", err)
			}
			return true, false
		}
	}
	return false, false
}

// matchCommand reports whether name matches cmd.Name or any of its aliases
// (case-insensitive).
func matchCommand(name string, cmd plugin.ChatCommand) bool {
	if strings.EqualFold(name, cmd.Name) {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.EqualFold(name, alias) {
			return true
		}
	}
	return false
}

// Run starts the interactive REPL loop.
func (a *ChatAgent) Run() error {
	loadHistory()
	for {
		line, quit, err := readInput()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("chat: input error: %w", err)
		}
		if quit {
			break
		}
		if line == "" {
			continue
		}

		fmt.Fprintln(os.Stdout, questionStyle.Render(line))
		fmt.Fprintln(os.Stdout)

		for _, preprocessor := range plugin.Preprocessors() {
			if str, err := preprocessor(context.Background(), line); err == nil {
				line = str
			} else {
				printer.Error("chat: preprocessor error", "err", err)
			}
		}

		if handled, exit := a.handleCommand(line); handled {
			if exit {
				break
			}
			continue
		}

		answer, err := a.Ask(line)
		if err != nil {
			printer.Error("chat: failed to get answer", "err", err)
			fmt.Fprintln(os.Stdout, "Error: failed to get response. Please try again.")
			continue
		}

		rendered, err := a.renderer.Render(answer)
		if err != nil {
			fmt.Fprintln(os.Stdout, answer)
			fmt.Fprintln(os.Stdout)
		} else {
			fmt.Fprint(os.Stdout, rendered)
		}
	}

	return nil
}
