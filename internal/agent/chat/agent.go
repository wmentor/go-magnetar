package chat

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/term"

	"github.com/wmentor/go-magnetar/internal/agent"
	"github.com/wmentor/go-magnetar/internal/agent/summarizer"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/gmutex"
	hstore "github.com/wmentor/go-magnetar/internal/history"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

//go:embed prompt.txt
var systemPrompt []byte

const (
	maxSearchToolCalls  = 20
	noProgressCountMax  = 4
	maxAskLoopIteration = 50

	// charsPerToken is a rough approximation used for context-window budget estimation.
	// OpenAI models average ~4 UTF-8 characters per token.
	charsPerToken = 4

	// doubleTabInterval is the maximum time between two Tab presses to trigger
	// showing completion suggestions (like bash/zsh double-tab behavior).
	doubleTabInterval = 500 * time.Millisecond
)

// projectContextFiles lists project context files in priority order; the first one found is loaded.
var projectContextFiles = []string{"AGENTS.md", "CLAUDE.md"}

// loadProjectContext returns the name and content of the first existing project context file in root.
func loadProjectContext(root *os.Root) (string, []byte, bool) {
	if root == nil {
		return "", nil, false
	}
	for _, name := range projectContextFiles {
		if content, err := root.ReadFile(name); err == nil {
			return name, content, true
		}
	}
	return "", nil, false
}

// ChatAgent is an interactive REPL-based chat agent backed by a RAG knowledge base.
type ChatAgent struct {
	cfg        *config.Config
	llm        *openai.Client
	summarizer *summarizer.Summarizer
	renderer   *glamour.TermRenderer
	messages   []openai.ChatCompletionMessage
	root       *os.Root
	toolStore  *tools.Store
}

// New creates a new ChatAgent instance.
func New(cfg *config.Config, root *os.Root, toolStore *tools.Store, maskService plugin.MaskService) (*ChatAgent, error) {
	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	llmCfg.HTTPClient = requester.Get()
	llmClient := openai.NewClientWithConfig(llmCfg)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise markdown renderer: %w", err)
	}

	language := cfg.String("language")

	systemContent := strings.ReplaceAll(string(systemPrompt), "{{LANGUAGE}}", language)

	if name, content, ok := loadProjectContext(root); ok {
		systemContent = systemContent + "\n\n# Project context (from " + name + ")\n\n" + string(content)
		printer.ToolCall(printer.IconDone, name+" has been loaded")
		printer.ToolEmptyLine()
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: maskService.Process(context.Background(), systemContent),
		},
	}

	return &ChatAgent{
		cfg:        cfg,
		llm:        llmClient,
		summarizer: summarizer.New(cfg),
		renderer:   renderer,
		messages:   messages,
		root:       root,
		toolStore:  toolStore,
	}, nil
}

func (a *ChatAgent) Reconfigure() {
	llmCfg := openai.DefaultConfig(a.cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = a.cfg.ProfileParamString("llm.base_url")
	llmCfg.HTTPClient = requester.Get()
	a.llm = openai.NewClientWithConfig(llmCfg)
}

func (a *ChatAgent) SetMessages(msgs []openai.ChatCompletionMessage) {
	a.messages = msgs
}

// --- AgentHandle implementation ---

// agentHandle is a private adapter that exposes ChatAgent internals to chat
// command plugins through the plugin.AgentHandle interface.
type agentHandle struct {
	a *ChatAgent
}

func (h *agentHandle) Reconfigure() {
	h.a.Reconfigure()
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

func (h *agentHandle) Compact(mode ...string) error {
	compacted, err := h.a.summarizer.Compact(h.a.messages, mode...)
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

type toolRecord struct {
	Name string
	Args string
}

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *ChatAgent) Ask(state *plugin.State, userInput string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	a.compactIfNeeded()

	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: state.MaskService.Process(ctx, userInput),
	})

	beginIndex := len(a.messages)

	// Build tool list and dispatch map from registered plugins.
	tools := make([]openai.Tool, 0, 32)

	for _, t := range state.ToolStore.Tools() {
		def := t.Definition()
		tools = append(tools, def)
	}

	noProgressCount := 0

	lastAnswerContent := ""

	alreadyDoneToolCalls := map[toolRecord]struct{}{}
	allToolCallStat := map[string]int{}

	var agentCounter atomic.Int64

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
			return "", fmt.Errorf("chat: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("chat: empty response from LLM")
		}

		choice := resp.Choices[0]
		a.messages = append(a.messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonLength {
			printer.Error("chat: LLM response was truncated (finish_reason=length); the answer may be incomplete")
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
				if tool, err := state.ToolStore.GetTool(name); err == nil {
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
			// Sequential tools (e.g. edit, file_write) must not run
			// concurrently because they mutate shared state (the filesystem).
			var tools []openai.ToolCall

			for _, toolCall := range choice.Message.ToolCalls {
				toolKey := toolRecord{Name: toolCall.Function.Name, Args: toolCall.Function.Arguments}

				tool, err := state.ToolStore.GetTool(toolCall.Function.Name)
				if err != nil {
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "error: unknown tool " + toolCall.Function.Name,
						ToolCallID: toolCall.ID,
					})
					printer.Error("unknown tool call", "tool", toolKey.Name, "args", toolKey.Args)
					continue
				}

				if tool.RateLimit == 0 {
					if _, has := alreadyDoneToolCalls[toolKey]; has {
						a.messages = append(a.messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    "error: this tool was called earlier with the same parameters",
							ToolCallID: toolCall.ID,
						})
						printer.Warn("llm require double tool call", "tool", toolKey.Name, "args", toolKey.Args)
						continue
					}
					alreadyDoneToolCalls[toolKey] = struct{}{}
				}

				allToolCallStat[tool.Name]++
				if tool.RateLimit > 0 && tool.RateLimit < allToolCallStat[tool.Name] {
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "error: tool call limit reached",
						ToolCallID: toolCall.ID,
					})
					printer.Warn("tool call limit reached", "tool", toolKey.Name, "args", toolKey.Args)
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
					printer.Error("chat: tool execution failed", "tool", r.toolCall.Function.Name, "err", r.err)
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
				printer.Warn("broke potential infinite loop")
				if beginIndex < len(a.messages) {
					a.messages = a.messages[:beginIndex]
				}
				return lastAnswerContent, nil
			}

			continue
		}

		return choice.Message.Content, nil
	}

	printer.Warn("broke potential infinite loop")
	if beginIndex < len(a.messages) {
		a.messages = a.messages[:beginIndex]
	}
	return lastAnswerContent, nil
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
	input       textinput.Model
	done        bool
	quit        bool
	suggestions []string
	lastTabTime time.Time
	commands    []plugin.ChatCommand
}

var (
	_ = promptStyle
	_ = promptSymbol
	_ = questionStyle
)

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

func newInputModel(commands []plugin.ChatCommand) inputModel {
	ti := textinput.New()
	ti.Placeholder = "Ask anything... (Enter to submit)"
	ti.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	ti.Prompt = promptSymbol + " "
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = getTerminalWidth() - len(promptSymbol) - 2
	return inputModel{input: ti, commands: commands}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type { //nolint:exhaustive // ok.
		case tea.KeyEnter:
			m.done = true
			m.suggestions = nil
			text := m.input.Value()
			if text != "" {
				history.Add(text)
			}
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.quit = true
			m.suggestions = nil
			return m, tea.Quit
		case tea.KeyUp:
			m.suggestions = nil
			if record := history.Prev(); record != "" {
				m.input.SetValue(record)
				m.input.CursorEnd()
			}
			return m, nil
		case tea.KeyDown:
			m.suggestions = nil
			if record := history.Next(); record != "" {
				m.input.SetValue(record)
				m.input.CursorEnd()
			} else {
				m.input.SetValue("")
			}
			return m, nil
		case tea.KeyTab:
			m.handleTabCompletion()
			return m, nil
		default:
			// Clear suggestions on any other key press
			m.suggestions = nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *inputModel) View() string {
	var view strings.Builder
	view.WriteString(m.input.View())

	// Render suggestions below the input if there are any
	if len(m.suggestions) > 0 {
		view.WriteString("\n")
		for _, suggestion := range m.suggestions {
			view.WriteString("  " + suggestion + "\n")
		}
	}

	return view.String()
}

// handleTabCompletion provides shell command completion when Tab is pressed.
// For chat commands (starting with /), it completes command names and file paths for arguments.
// For regular commands, it completes executable names from PATH for the first word.
// For subsequent words, it completes file paths.
// Double Tab (within 500ms) shows all available completion suggestions.
func (m *inputModel) handleTabCompletion() {
	now := time.Now()

	// Check if this is a double-tab (second tab within interval)
	isDoubleTab := !m.lastTabTime.IsZero() && now.Sub(m.lastTabTime) < doubleTabInterval && len(m.suggestions) > 0

	// If double-tab and we have suggestions, just show them (they're already set)
	if isDoubleTab {
		m.lastTabTime = time.Time{} // Reset to prevent third tab from showing again
		return
	}

	text := m.input.Value()
	if text == "" {
		m.suggestions = nil
		return
	}

	// Check if we're completing a chat command (starts with /)
	if strings.HasPrefix(text, "/") {
		m.handleChatCommandCompletion(text)
		m.lastTabTime = now
		return
	}

	// Check if we're completing the first word (executable) or subsequent words (file paths)
	endsWithSpace := len(text) > 0 && text[len(text)-1] == ' '
	parts := strings.Fields(text)

	// Determine if we're completing the first word (command name)
	completingFirstWord := len(parts) == 0 || (len(parts) == 1 && !endsWithSpace)

	var prefix string
	var completions []string

	if completingFirstWord {
		// Complete executable name from PATH
		if len(parts) == 1 {
			prefix = parts[0]
		}
		completions = completeExecutable(prefix)
	} else {
		// Complete file path for the current word
		if endsWithSpace {
			prefix = ""
		} else if len(parts) > 0 {
			prefix = parts[len(parts)-1]
		}
		completions = completeFilePath(prefix)
	}

	// Save completions as suggestions for potential double-tab
	m.suggestions = completions
	m.lastTabTime = now

	// Apply the completion
	if len(completions) == 1 { //nolint:nestif // later.
		completion := completions[0]

		// For file paths, append "/" if it's a directory, " " if it's a file
		if !completingFirstWord {
			// Expand ~ for stat check
			statPath := completion
			if strings.HasPrefix(completion, "~") {
				if expanded, err := expandTilde(completion); err == nil {
					statPath = expanded
				}
			}
			if info, err := os.Stat(statPath); err == nil {
				if info.IsDir() && !strings.HasSuffix(completion, string(filepath.Separator)) {
					completion += string(filepath.Separator)
				} else if !info.IsDir() {
					completion += " "
				}
			}
		} else {
			// For executables, just add a space
			completion += " "
		}

		if completingFirstWord {
			if endsWithSpace || len(parts) == 0 {
				m.input.SetValue(text + completion)
			} else {
				m.input.SetValue(completion)
			}
		} else {
			if endsWithSpace {
				m.input.SetValue(text + completion)
			} else {
				// Replace the last word
				parts[len(parts)-1] = completion
				m.input.SetValue(strings.Join(parts, " "))
			}
		}
		m.input.CursorEnd()
		m.suggestions = nil // Single completion, no need to show suggestions
	} else if len(completions) > 1 {
		// Multiple completions - find and apply common prefix
		commonPrefix := findCommonPrefix(completions)
		if commonPrefix != "" && commonPrefix != prefix {
			if completingFirstWord {
				if endsWithSpace || len(parts) == 0 {
					m.input.SetValue(text + commonPrefix)
				} else {
					m.input.SetValue(commonPrefix)
				}
			} else {
				if endsWithSpace {
					m.input.SetValue(text + commonPrefix)
				} else {
					parts[len(parts)-1] = commonPrefix
					m.input.SetValue(strings.Join(parts, " "))
				}
			}
			m.input.CursorEnd()
		}
	}
}

// handleChatCommandCompletion handles completion for chat commands (starting with /).
// It completes command names when completing the first word, and file paths for arguments.
func (m *inputModel) handleChatCommandCompletion(text string) {
	endsWithSpace := len(text) > 0 && text[len(text)-1] == ' '
	parts := strings.Fields(text)

	// Determine if we're completing the command name or an argument
	completingCommandName := len(parts) <= 1 && !endsWithSpace

	var prefix string
	var completions []string

	if completingCommandName {
		// Complete chat command name
		if len(parts) == 1 {
			prefix = strings.TrimPrefix(parts[0], "/")
		}
		completions = m.completeChatCommand(prefix)
	} else {
		// Complete file path for command argument
		if endsWithSpace {
			prefix = ""
		} else if len(parts) > 1 {
			prefix = parts[len(parts)-1]
		}
		completions = completeFilePath(prefix)
	}

	// Save completions as suggestions for potential double-tab
	m.suggestions = completions

	// Apply the completion
	if len(completions) == 1 { //nolint:nestif // later.
		completion := completions[0]

		if completingCommandName {
			// For command names, add space after completion
			// Always replace the entire input since text starts with /
			m.input.SetValue("/" + completion + " ")
		} else {
			// For file paths, append "/" if it's a directory, " " if it's a file
			statPath := completion
			if strings.HasPrefix(completion, "~") {
				if expanded, err := expandTilde(completion); err == nil {
					statPath = expanded
				}
			}
			if info, err := os.Stat(statPath); err == nil {
				if info.IsDir() && !strings.HasSuffix(completion, string(filepath.Separator)) {
					completion += string(filepath.Separator)
				} else if !info.IsDir() {
					completion += " "
				}
			}

			if endsWithSpace {
				m.input.SetValue(text + completion)
			} else {
				// Replace the last word
				parts[len(parts)-1] = completion
				// Reconstruct with the command name (parts[0] already contains /)
				m.input.SetValue(strings.Join(parts, " "))
			}
		}
		m.input.CursorEnd()
		m.suggestions = nil // Single completion, no need to show suggestions
	} else if len(completions) > 1 {
		// Multiple completions - find and apply common prefix
		commonPrefix := findCommonPrefix(completions)
		if commonPrefix != "" && commonPrefix != prefix {
			if completingCommandName {
				// Always replace the entire input since text starts with /
				m.input.SetValue("/" + commonPrefix)
			} else {
				if endsWithSpace {
					m.input.SetValue(text + commonPrefix)
				} else {
					parts[len(parts)-1] = commonPrefix
					// Reconstruct with the command name (parts[0] already contains /)
					m.input.SetValue(strings.Join(parts, " "))
				}
			}
			m.input.CursorEnd()
		}
	}
}

// completeChatCommand returns chat command names that match the prefix.
func (m *inputModel) completeChatCommand(prefix string) []string {
	var completions []string

	for _, cmd := range m.commands {
		// Check main name
		if strings.HasPrefix(strings.ToLower(cmd.Name), strings.ToLower(prefix)) {
			completions = append(completions, cmd.Name)
		} else {
			// Check aliases
			for _, alias := range cmd.Aliases {
				if strings.HasPrefix(strings.ToLower(alias), strings.ToLower(prefix)) {
					completions = append(completions, alias)
					break
				}
			}
		}
	}

	slices.Sort(completions)
	return completions
}

// completeExecutable returns executable names from PATH that match the prefix.
func completeExecutable(prefix string) []string {
	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	seen := make(map[string]bool)
	var completions []string

	for _, dir := range pathDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}

			// Check if executable
			fullPath := filepath.Join(dir, name)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			if info.Mode()&0o111 != 0 {
				if !seen[name] {
					seen[name] = true
					completions = append(completions, name)
				}
			}
		}
	}

	slices.Sort(completions)
	return completions
}

// expandTilde replaces ~ with the user's home directory.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

// completeFilePath returns file paths that match the prefix.
func completeFilePath(prefix string) []string {
	if prefix == "" {
		// List all files in current directory
		entries, err := os.ReadDir(".")
		if err != nil {
			return nil
		}

		var completions []string
		for _, entry := range entries {
			completions = append(completions, entry.Name())
		}

		slices.Sort(completions)
		return completions
	}

	// Handle ~ expansion
	hasTilde := strings.HasPrefix(prefix, "~")
	expandedPrefix := prefix
	if hasTilde {
		var err error
		expandedPrefix, err = expandTilde(prefix)
		if err != nil {
			return nil
		}
	}

	// Check if prefix ends with separator (completing inside a directory)
	if strings.HasSuffix(expandedPrefix, string(filepath.Separator)) {
		entries, err := os.ReadDir(expandedPrefix)
		if err != nil {
			return nil
		}

		var completions []string
		for _, entry := range entries {
			completion := expandedPrefix + entry.Name()
			// Restore ~ in the completion if it was in the prefix
			if hasTilde {
				completion = restoreTilde(completion, prefix)
			}
			completions = append(completions, completion)
		}

		slices.Sort(completions)
		return completions
	}

	// Complete the last component of the path
	dir := filepath.Dir(expandedPrefix)
	base := filepath.Base(expandedPrefix)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var completions []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			fullPath := filepath.Join(dir, name)
			// Restore ~ in the completion if it was in the prefix
			if hasTilde {
				fullPath = restoreTilde(fullPath, prefix)
			}
			completions = append(completions, fullPath)
		}
	}

	slices.Sort(completions)
	return completions
}

// restoreTilde replaces the home directory prefix with ~ if the original prefix had it.
func restoreTilde(expandedPath, originalPrefix string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return expandedPath
	}

	// Check if the original prefix started with ~/
	if strings.HasPrefix(originalPrefix, "~/") {
		if strings.HasPrefix(expandedPath, home) {
			return "~" + expandedPath[len(home):]
		}
	} else if originalPrefix == "~" {
		if expandedPath == home {
			return "~"
		}
	}

	return expandedPath
}

// findCommonPrefix finds the longest common prefix among the given strings.
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}

	return prefix
}

// readInput launches a bubbletea program to collect one line from the user.
func readInput(commands []plugin.ChatCommand) (string, bool, error) {
	m := newInputModel(commands)
	p := tea.NewProgram(&m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", false, err
	}
	final := result.(*inputModel) //nolint:forcetypeassert // valid case.
	fmt.Fprintln(os.Stdout)
	return strings.TrimSpace(final.input.Value()), final.quit, nil
}

// printAnswer renders and prints the given answer to stdout.
// If rendering fails, it falls back to printing the raw answer.
func (a *ChatAgent) printAnswer(state *plugin.State, answer string, noLog bool) {
	if !noLog {
		state.ResponseStore.Push(answer)
	}
	rendered, err := a.renderer.Render(answer)
	if err != nil {
		fmt.Fprintln(os.Stdout, answer)
		fmt.Fprintln(os.Stdout)
	} else {
		fmt.Fprint(os.Stdout, rendered)
	}
}

// handleCommand processes a slash-command entered by the user.
// Returns (*plugin.ChatCommandResponse, error).
func (a *ChatAgent) handleCommand(line string, knownCommands []plugin.ChatCommand) (*plugin.ChatCommandResponse, error) {
	trimLine := strings.TrimSpace(line)

	if !strings.HasPrefix(line, "/") && trimLine != "exit" && trimLine != "quit" {
		parts := strings.SplitN(trimLine, " ", 2)
		name := parts[0]

		if _, err := exec.LookPath(name); err != nil {
			return nil, plugin.ErrNoCommand
		}

		cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", line)
		buffer := bytes.NewBuffer(nil)

		sout := io.MultiWriter(os.Stdout, buffer)
		serr := io.MultiWriter(os.Stderr, buffer)

		cmd.Stdout = sout
		cmd.Stderr = serr

		err := cmd.Run()
		if err != nil {
			if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
				fmt.Fprintf(sout, "\n[Exit code: %d]\n", exitErr.ExitCode())
			} else {
				fmt.Fprintf(sout, "\n[Error: %v]\n", err)
			}
		}

		fmt.Println("")

		return plugin.NopChatCommandResponse, nil
	}

	// Strip leading "/" and split into name + args.
	trimmed := strings.TrimPrefix(trimLine, "/")
	parts := strings.SplitN(trimmed, " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) == 2 {
		args = strings.TrimSpace(parts[1])
	}

	for _, cmd := range knownCommands {
		if matchCommand(name, cmd) {
			handle := &agentHandle{a: a}
			cmdInput := &plugin.ChatCommandInput{
				AgentHandler: handle,
				Args:         args,
				Printer:      printer.GetDefault(),
			}
			resp, err := cmd.Execute(context.Background(), cmdInput)
			if err != nil {
				printer.Error("chat: command error", "cmd", cmd.Name, "err", err)
				return nil, err
			}
			return resp, nil
		}
	}
	return nil, plugin.ErrNoCommand
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
func (a *ChatAgent) Run(state *plugin.State, skillName string) error {
	loadHistory()
	for {
		var line string
		var quit bool
		var err error

		if skillName != "" {
			line = "/skill " + skillName
			skillName = ""
		} else {
			line, quit, err = readInput(plugin.ChatCommands())
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
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

		start := time.Now()
		cmdResponse, err := a.handleCommand(line, plugin.ChatCommands())
		elapsed := time.Since(start)
		if err == nil {
			if cmdResponse != nil {
				if cmdResponse.Stdout != "" {
					a.printAnswer(state, cmdResponse.Stdout, cmdResponse.NoLog)
				}
				if cmdResponse.Exit {
					break
				}
				if cmdResponse.Content == "" {
					printTiming(state, elapsed)
					continue
				}
				line = cmdResponse.Content
			}
		} else if !errors.Is(err, plugin.ErrNoCommand) {
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Measured from before the chat command, so a command that hands its
		// content to the model is counted in the answer's time.
		answer, err := a.Ask(state, line)
		elapsed = time.Since(start)
		if err != nil {
			printer.Error("chat: failed to get answer", "err", err)
			fmt.Fprintln(os.Stdout, "Error: failed to get response. Please try again.")
			continue
		}

		a.printAnswer(state, answer, false)
		printTiming(state, elapsed)
	}

	return nil
}

// printTiming prints how long an answer took when /timing or --enable-timing turned it on.
func printTiming(state *plugin.State, elapsed time.Duration) {
	if state.TimingEnabled.Load() {
		fmt.Fprintf(os.Stdout, "  Execution time: %s\n\n", printDuration(elapsed))
	}
}

// printDuration renders d rounded to the millisecond, or to the second from 10s up.
func printDuration(d time.Duration) string {
	if d >= time.Second*10 {
		d = d.Round(time.Second)
		return d.String()
	} else {
		d = d.Round(time.Millisecond)
	}

	return d.String()
}
