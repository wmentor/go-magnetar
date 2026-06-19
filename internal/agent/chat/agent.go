package chat

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/go-units"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent/summarizer"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
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

// ChatAgent is an interactive REPL-based chat agent backed by a RAG knowledge base.
type ChatAgent struct {
	cfg        *config.Config
	llm        *openai.Client
	rag        *rag.RAGTools
	web        *web.WebTools
	generic    *generic.GenericTools
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

	genTools := generic.New(cfg, root)

	ragTools, err := rag.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise RAG tools: %w", err)
	}

	webTools, err := web.New(cfg, root)
	if err != nil {
		return nil, fmt.Errorf("chat: failed to initialise web tools: %w", err)
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
		generic:    genTools,
		rag:        ragTools,
		web:        webTools,
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
		a.generic.DefinitionFileRead(),
		a.generic.DefinitionFileList(),
		a.generic.DefinitionFileWrite(),
		a.rag.DefinitionSearch(),
		a.web.Definition(),
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

			var result string
			switch name {
			case "rag_search":
				result = a.rag.Dispatch(name, args)
			case "web_fetch":
				result = a.web.Dispatch(name, args)
			case "file_list", "file_read", "file_write":
				result = a.generic.Dispatch(name, args)
			default:
				result = "error: unknown tool " + name
			}

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

// inputModel is a minimal bubbletea model used solely to collect one line of
// user input with a styled prompt. It quits on Enter (submitting the text) or
// Ctrl+C / Ctrl+D (requesting an exit).
type inputModel struct {
	input textinput.Model
	done  bool
	quit  bool
}

var (
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")). // purple
			Bold(true)
	promptSymbol = promptStyle.Render("◆")

	questionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("99")).
			PaddingLeft(1).
			Faint(true)
)

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

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.quit = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	return m.input.View()
}

// readInput launches a bubbletea program to collect one line from the user.
// Returns the trimmed input, a quit flag, and any error.
func readInput() (string, bool, error) {
	m := newInputModel()
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", false, err
	}
	final := result.(inputModel)
	// Print a newline after the input line so subsequent output starts cleanly.
	fmt.Fprintln(os.Stdout)
	return strings.TrimSpace(final.input.Value()), final.quit, nil
}

// contextStats returns byte and estimated token counts for the current message history.
func (a *ChatAgent) contextStats() (bytes int, tokens int) {
	for _, m := range a.messages {
		bytes += len(m.Role) + len(m.Content)
		for _, tc := range m.ToolCalls {
			bytes += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
		tokens += estimateTokens(m)
	}
	return bytes, tokens
}

// helpText is the reference text shown by the /help command.
const helpText = `Available chat commands:
    /help            show this help message
    /exit  exit      end the session and exit
    /compact         compress conversation history via summarizer
    /new             start a new session and clear context
    /stat            show context statistics (messages, tokens, bytes, models)

The assistant can use the following tools:
    file_read        read the contents of a file by its path
    file_list        list all files in the current directory tree
    file_write       write content to a file
    rag_search       search the knowledge base for information
    web_fetch        fetch and preprocess web pages for up-to-date information
`

// handleCommand processes a slash-command entered by the user.
// It returns true if the input was a command (and was handled), false otherwise.
// The second return value signals that the REPL should exit.
func (a *ChatAgent) handleCommand(line string) (handled bool, exit bool) {
	cmd := strings.ToLower(strings.TrimSpace(line))
	switch cmd {
	case "/help", "help":
		fmt.Fprint(os.Stdout, helpText+"\n")
		return true, false

	case "/exit", "exit":
		return true, true

	case "/compact":
		compacted, err := a.summarizer.Compact(a.messages)
		if err != nil {
			slog.Error("chat: manual compaction failed", "err", err)
			fmt.Fprintln(os.Stdout, "Error: compaction failed — see logs for details.")
		} else {
			before := len(a.messages)
			a.messages = compacted
			after := len(a.messages)
			fmt.Fprintf(os.Stdout, "Context compacted: %d → %d messages.\n\n", before, after)
		}
		return true, false

	case "/new":
		slog.Info("chat: starting new session, clearing context")
		a.messages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		}
		fmt.Fprintln(os.Stdout, "New session started. Context cleared.")
		return true, false

	case "/stat":
		b, tokens := a.contextStats()
		fmt.Fprintf(os.Stdout,
			"Context stats:\n  messages    : %d\n  tokens      : ~%d (estimated)\n  bytes       : %s\n  llm model   : %s\n  rag model   : %s\n  vector size : %d\n\n",
			len(a.messages), tokens, units.HumanSize(float64(b)),
			a.cfg.LLM.Model,
			a.cfg.RAG.LLM.Model, a.cfg.RAG.LLM.VectorSize,
		)
		return true, false
	}
	return false, false
}

// Run starts the interactive REPL loop.
func (a *ChatAgent) Run() error {
	for {
		line, quit, err := readInput()
		if err != nil {
			// Treat closed stdin / broken pipe as normal exit.
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

		// Handle slash-commands before sending input to the LLM.
		if handled, exit := a.handleCommand(line); handled {
			if exit {
				break
			}
			continue
		}

		fmt.Fprintln(os.Stdout, questionStyle.Render(line))
		fmt.Fprintln(os.Stdout)

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
