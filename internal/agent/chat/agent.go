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
	"sync/atomic"
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
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const systemPromptTemplate = `You are a helpful assistant that answers questions strictly based on the knowledge base.

Parameters:
- language: %s

Rules:
- For questions about facts, documents, or knowledge: use *web_search* or *rag_search* first. Base your answer exclusively on what search returns.
- For file operations (read, edit, create, run commands): use the file/exec tools directly. Do not call search tools first.
- You may call *web_search* and *rag_search* multiple times with different queries to gather all necessary information.
- Use *web_search* for time-sensitive or general knowledge queries (e.g., current events, news, popular topics).
- Use *rag_search* for domain-specific questions that should be answered from the knowledge base.
- Do not invent, assume, or extrapolate facts beyond what the tools return.
- If *web_search* and *rag_search* and *rag_search* provide no relevant results, tell the user honestly that you don't have information on this topic.
- Be concise and precise.
- You may execute multiple tools in parallel when the response contains multiple tool calls.
- NEVER combine file_write, file_edit, or file_edit_multiple with any other tool in the same response.
  Always call file_read first in a separate step, then call the edit tool alone.

FILE EDITING RULES:
- ALWAYS call file_read FIRST before any file_edit or file_edit_multiple call. Never guess file contents.
- Use file_edit for a single targeted replacement; old_text must appear EXACTLY ONCE in the file.
- Use file_edit_multiple when making several replacements in one file or when old_text appears more than once.
- Use file_write only when creating a new file or completely rewriting an existing one.
- file_edit and file_edit_multiple work ONLY with plain-text files (.go, .py, .js, .ts, .md, .yaml, .json, .txt, etc.).
- Formats .docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm have dedicated codecs and CANNOT be edited
  with file_edit — use file_write or a specialized tool for those formats.
- The search is CASE-SENSITIVE and requires an exact character-for-character match including whitespace and newlines.
- Copy old_text verbatim from the file_read output — do not paraphrase or reconstruct it from memory.
- Include enough surrounding context in old_text to make it unique: add the preceding or following line if needed.
- If file_edit returns "old_text appears N times", add more context to old_text or switch to file_edit_multiple.
- If file_edit returns "old_text not found", re-read the file and copy the text again exactly as it appears.
- Do not add or remove leading/trailing whitespace or newlines in old_text unless the file actually has them.

SKILL USAGE:
- When a skill is applied (e.g., via /skill command), the skill's output is provided to you as the user's input.
- You MUST treat this skill output as the direct user request and respond to it naturally.
- When responding to skill output, do NOT use tools or generate tool calls UNLESS the skill explicitly instructs you to use tools.
- Skill results without explicit tool instructions MUST be answered directly without calling any tools.

SECURITY WARNING: You are FORBIDDEN from executing destructive commands such as:
- rm -rf /, rm -rf *, sudo, chmod 777, dd, mkfs, fdisk, or any command that could delete/modify/system files.
- curl ... | bash, wget ... | sh, zsh, or any pipe to shell execution.

You are FORBIDDEN from using heredoc syntax to write files:
- Never execute commands like cat > /tmp/params.txt << 'EOF', cat << EOF, or similar heredoc patterns.
- This includes any command that writes content to files using heredoc redirection.

You are FORBIDDEN from downloading and executing untrusted scripts:
- Never execute curl ... | bash, wget ... -O - | sh, or similar patterns.
- Always verify script sources and contents before execution.

IMPORTANT: The exec tool will BLOCK any command matching security patterns (rm -rf, sudo, mkfs, dd with /dev/, pipe to bash/sh/zsh,
git commit/push/rebase/pull/cherry-pick/reset/stash/clean/reflog). DO NOT attempt to bypass these protections.

SECURITY CHECK: When the guard tool analyzes a command for safety:
- If the guard is uncertain about whether a command is safe, ALWAYS reject it (allow=false).
- Err on the side of caution - better to block a safe command than to allow a dangerous one.
`

const (
	maxSearchToolCalls  = 20
	noProgressCountMax  = 4
	maxAskLoopIteration = 50

	// charsPerToken is a rough approximation used for context-window budget estimation.
	// OpenAI models average ~4 UTF-8 characters per token.
	charsPerToken = 4
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
}

// New creates a new ChatAgent instance.
func New(cfg *config.Config, root *os.Root) (*ChatAgent, error) {
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
	systemContent := fmt.Sprintf(systemPromptTemplate, language)
	if name, content, ok := loadProjectContext(root); ok {
		systemContent = systemContent + "\n\n# Project context (from " + name + ")\n\n" + string(content)
		printer.ToolCall(printer.IconDone, name+" has been loaded")
		printer.ToolEmptyLine()
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

type toolRecord struct {
	Name string
	Args string
}

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

	toolCallCount := new(int64)
	noProgressCount := 0
	allInterationRealToolCall := new(int64)

	lastAnswerContent := ""

	alreadyDoneToolCalls := map[toolRecord]struct{}{}

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

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
		resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:           a.cfg.ProfileParamString("llm.model"),
			Messages:        trimmed,
			Tools:           activeTools,
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

		if choice.FinishReason == openai.FinishReasonLength {
			printer.Error("chat: LLM response was truncated (finish_reason=length); the answer may be incomplete")
		}

		if choice.FinishReason == openai.FinishReasonToolCalls { //nolint:nestif // later.
			type toolResult struct {
				toolCall openai.ToolCall
				result   string
				err      error
			}

			results := make(chan toolResult, len(choice.Message.ToolCalls))

			var wg sync.WaitGroup

			atomic.StoreInt64(allInterationRealToolCall, 0)

			// executeTool runs a single tool call and sends the result to results.
			executeTool := func(tc openai.ToolCall) {
				name := tc.Function.Name
				args := tc.Function.Arguments
				toolKey := toolRecord{Name: name, Args: args}

				var result string
				t, ok := toolMap[name]
				if !ok {
					result = "error: unknown tool " + name
				} else {
					if t.IsSearchTool {
						if atomic.LoadInt64(toolCallCount) >= maxSearchToolCalls {
							result = "error: reached maximum number of search tool calls, no further searches allowed"
							txt := fmt.Sprintf("LLM requested tool call %s but maximum search tool calls (%d) has been reached",
								toolKey.Name, maxSearchToolCalls)
							printer.Warn(txt, toolKey.Name, toolKey.Args)
						} else {
							atomic.AddInt64(toolCallCount, 1)
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
			}

			// Split tool calls into parallel and sequential buckets.
			// Sequential tools (e.g. file_edit, file_write) must not run
			// concurrently because they mutate shared state (the filesystem).
			var parallelCalls []openai.ToolCall
			var sequentialCalls []openai.ToolCall

			for _, toolCall := range choice.Message.ToolCalls {
				toolKey := toolRecord{Name: toolCall.Function.Name, Args: toolCall.Function.Arguments}
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
				if atomic.LoadInt64(toolCallCount) < maxSearchToolCalls {
					atomic.AddInt64(allInterationRealToolCall, 1)
				}

				t, ok := toolMap[toolCall.Function.Name]
				if ok && t.Sequential {
					sequentialCalls = append(sequentialCalls, toolCall)
				} else {
					parallelCalls = append(parallelCalls, toolCall)
				}
			}

			// Run parallel tools concurrently.
			for _, tc := range parallelCalls {
				wg.Add(1)
				go func(tc openai.ToolCall) {
					defer wg.Done()
					executeTool(tc)
				}(tc)
			}
			wg.Wait()

			// Run sequential tools one at a time, in the order the LLM listed them.
			for _, tc := range sequentialCalls {
				executeTool(tc)
			}

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

			if atomic.LoadInt64(allInterationRealToolCall) == 0 {
				noProgressCount++
			} else {
				noProgressCount = 0
			}

			lastAnswerContent = choice.Message.Content

			if noProgressCount >= noProgressCountMax {
				printer.Warn("broke potential infinite loop")
				return lastAnswerContent, nil
			}

			continue
		}

		return choice.Message.Content, nil
	}

	printer.Warn("broke potential infinite loop")
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
	input textinput.Model
	done  bool
	quit  bool
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
		switch msg.Type { //nolint:exhaustive // ok.
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
	// Strip leading "/" and split into name + args.
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "/")
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
			line, quit, err = readInput()
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
				if cmdResponse.Exit {
					break
				}
				// Print Stdout to stdout if present
				if cmdResponse.Stdout != "" {
					a.printAnswer(state, cmdResponse.Stdout, cmdResponse.NoLog)
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
		answer, err := a.Ask(line)
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
