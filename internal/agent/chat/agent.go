package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent/summarizer"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
)

const systemPrompt = `You are a helpful assistant that answers questions strictly based on the knowledge base.

Rules:
- Always try rag_search first for every question, even if you think you already know the answer.
- If rag_search returns relevant results, base your answer exclusively on those results. Do not use web_fetch in this case.
- You may call rag_search multiple times with different queries to gather all necessary information.
- Only use web_fetch if rag_search returned no relevant results and the user explicitly needs up-to-date or external information.
- Do not invent, assume, or extrapolate facts beyond what the tools return.
- If neither rag_search nor web_fetch provides relevant information, tell the user honestly that you don't have information on this topic.
- Be concise and precise.`

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
	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise markdown renderer: %w", err)
	}

	systemContent := systemPrompt
	if root != nil {
		genTools := generic.New(cfg, root)
		if _, statErr := root.Stat(agentsFile); statErr == nil {
			if agentsContent, ok := genTools.FileRead(agentsFile); ok {
				systemContent = systemPrompt + "\n\n# Project context (from AGENTS.md)\n\n" + agentsContent
				slog.Info("agents: loaded AGENTS.md into system prompt")
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
	slog.Info("chat: starting new session, clearing context")
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
	t := chars / charsPerToken
	if t < 1 {
		t = 1
	}
	return t
}

// trimMessages returns a copy of a.messages that fits within the context window.
func (a *ChatAgent) trimMessages(reservedOutputTokens int) []openai.ChatCompletionMessage {
	if a.cfg.LLM.Context <= 0 {
		return a.messages
	}

	budget := a.cfg.LLM.Context - reservedOutputTokens
	if budget <= 0 {
		budget = a.cfg.LLM.Context
	}

	system := a.messages[0]
	budget -= estimateTokens(system)

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
		slog.Error("chat: history compaction failed, continuing with full history", "err", err)
		return
	}
	a.messages = compacted
}

// maxSearchToolCalls is the maximum number of search-related tool calls per user request.
const maxSearchToolCalls = 10

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *ChatAgent) ask(userInput string) (string, error) {
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
		a.messages = append(a.messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonToolCalls {
			for _, toolCall := range choice.Message.ToolCalls {
				name := toolCall.Function.Name
				args := toolCall.Function.Arguments

				slog.Debug("chat: tool call", "tool", name, "args", args)

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

				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
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
	promptSymbol = promptStyle.Render("◆")

	questionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("69")).
			PaddingLeft(1).
			Faint(false)

	history      []string
	historyIndex int

	historyFile = func() string {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".go-magnetar-history.json")
	}()
	historySizeLimit = 200
)

func loadHistory() {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			history = nil
			historyIndex = 0
			return
		}
		slog.Debug("chat: failed to load history file", "err", err)
		history = nil
		historyIndex = 0
		return
	}
	if len(data) == 0 {
		history = nil
		historyIndex = 0
		return
	}
	if err := json.Unmarshal(data, &history); err != nil {
		slog.Debug("chat: failed to parse history file", "err", err)
		history = nil
		historyIndex = 0
		return
	}
	if len(history) > historySizeLimit {
		history = history[len(history)-historySizeLimit:]
	}
	historyIndex = len(history)
}

func saveHistory() error {
	if len(history) == 0 {
		return os.Remove(historyFile)
	}
	if len(history) > historySizeLimit {
		history = history[len(history)-historySizeLimit:]
	}
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return os.WriteFile(historyFile, data, 0600)
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
var _ = history
var _ = historyIndex
var _ = historyFile

func newInputModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "Ask anything..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	ti.Prompt = promptSymbol + " "
	ti.Focus()
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
				history = append(history, text)
				historyIndex = len(history)
				if err := saveHistory(); err != nil {
					slog.Debug("chat: failed to save history", "err", err)
				}
			}
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.quit = true
			if err := saveHistory(); err != nil {
				slog.Debug("chat: failed to save history on Ctrl+C", "err", err)
			}
			return m, tea.Quit
		case tea.KeyUp:
			if len(history) > 0 && historyIndex > 0 {
				historyIndex--
				m.input.SetValue(history[historyIndex])
				m.input.CursorEnd()
			}
			return m, nil
		case tea.KeyDown:
			if historyIndex < len(history)-1 {
				historyIndex++
				m.input.SetValue(history[historyIndex])
				m.input.CursorEnd()
			} else if historyIndex == len(history)-1 {
				historyIndex = len(history)
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

var _ = history

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
				slog.Error("chat: command error", "cmd", cmd.Name, "err", err)
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

		if handled, exit := a.handleCommand(line); handled {
			if exit {
				break
			}
			continue
		}

		answer, err := a.ask(line)
		if err != nil {
			slog.Error("chat: failed to get answer", "err", err)
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
