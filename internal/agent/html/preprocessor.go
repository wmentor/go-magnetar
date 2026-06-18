package html

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
)

const systemPrompt = `You are an HTML content preprocessor. Your task is to clean HTML pages by removing noise elements and convert the result to clean Markdown.

Noise elements to remove:
- Advertising blocks (ads, banners, pop-ups)
- Navigation headers/footers
- Related articles suggestions
- Social media widgets
- Cookie notices
- Newsletter prompts
- Duplicate content (repeated sections, too-short paragraphs)
- Unnecessary whitespace and formatting
- User comments

Keep:
- Main content (articles, blog posts, documentation)
- Headings, paragraphs, lists, links, code blocks
- Essential metadata (author, date, title if important)

Output format:
1. Extract the core content and structure
2. Convert to clean Markdown
3. Remove all inline styles
4. Preserve important formatting (code, links, emphasis)
5. Return only the cleaned Markdown text (no explanations, no prefix)

Do not add any commentary, only the cleaned Markdown content.

IMPORTANT: Return the text as-is without truncation. Preserve all main content and do not cut off sections.`

// Preprocessor is an AI agent that cleans HTML pages and converts to Markdown.
type Preprocessor struct {
	cfg     *config.Config
	llm     *openai.Client
	generic *generic.GenericTools
}

// New creates a new Preprocessor instance.
func New(cfg *config.Config) (*Preprocessor, error) {
	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	return &Preprocessor{
		cfg:     cfg,
		llm:     llmClient,
		generic: generic.New(cfg),
	}, nil
}

// runAgentLoop executes the tool-use agentic loop until the model stops calling tools.
// Returns the final answer content.
func (p *Preprocessor) runAgentLoop(messages []openai.ChatCompletionMessage) (string, error) {
	tools := []openai.Tool{
		p.generic.Definition(),
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := p.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:     p.cfg.LLM.Model,
			Messages:  messages,
			Tools:     tools,
			MaxTokens: p.cfg.LLM.Context,
		})
		cancel()

		if err != nil {
			return "", fmt.Errorf("preprocessor: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("preprocessor: empty response from LLM")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonToolCalls {
			for _, toolCall := range choice.Message.ToolCalls {
				name := toolCall.Function.Name
				args := toolCall.Function.Arguments

				slog.Debug("preprocessor: tool call", "tool", name, "args", args)

				var result string
				switch name {
				case "file_read":
					result = p.generic.Dispatch(name, args)
				default:
					result = "error: unknown tool " + name
				}

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}
			continue
		}

		slog.Debug("preprocessor: done", "response", choice.Message.Content)
		return choice.Message.Content, nil
	}
}

// ProcessHTML cleans an HTML file and returns Markdown
func (p *Preprocessor) ProcessHTML(filename string) (string, error) {
	slog.Debug("preprocessor: processing HTML file", "file", filename)

	htmlContent, ok := p.generic.FileRead(filename)
	if !ok {
		return "", fmt.Errorf("preprocessor: failed to read file %s", filename)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Please clean this HTML and convert to Markdown:\n\n%s", htmlContent),
		},
	}

	return p.runAgentLoop(messages)
}

// ProcessHTMLString takes HTML as string and returns cleaned Markdown
func (p *Preprocessor) ProcessHTMLString(htmlStr string) (string, error) {
	slog.Debug("preprocessor: processing HTML string")

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Please clean this HTML and convert to Markdown:\n\n%s", htmlStr),
		},
	}

	return p.runAgentLoop(messages)
}
