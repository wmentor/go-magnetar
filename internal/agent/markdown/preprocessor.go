package markdown

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
)

const (
	agentLoopTimeout = time.Minute * 15
)

const systemPrompt = `You are an editor who cleans Markdown text from clutter left after parsing web pages.

REMOVE:

* Calls to subscribe, share, or leave a comment
* "Read also", "Related articles" blocks
* Navigation breadcrumbs
* Ad inserts, promo blocks
* Broken or meaningless links
* Meta-information like "published N minutes ago"

KEEP:

* Headings, lists, tables, quotes
* Code blocks
* The semantic content of the article
* Markdown formatting

Return ONLY the cleaned Markdown, with no comments or explanations.`

// Preprocessor is an AI agent that cleans HTML pages and converts to Markdown.
type Preprocessor struct {
	cfg     *config.Config
	llm     *openai.Client
	generic *generic.GenericTools
}

// New creates a new Preprocessor instance.
func New(cfg *config.Config, root *os.Root) (*Preprocessor, error) {
	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	return &Preprocessor{
		cfg:     cfg,
		llm:     llmClient,
		generic: generic.New(cfg, root),
	}, nil
}

// runAgentLoop executes the tool-use agentic loop until the model stops calling tools.
// Returns the final answer content.
func (p *Preprocessor) runAgentLoop(messages []openai.ChatCompletionMessage) (string, error) {
	tools := []openai.Tool{
		p.generic.DefinitionFileRead(),
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), agentLoopTimeout)
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

func (p *Preprocessor) ProcessHTML(filename string) (string, error) {
	slog.Debug("preprocessor: processing markdown file", "file", filename)

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

func (p *Preprocessor) ProcessHTMLString(htmlStr string) (string, error) {
	slog.Debug("preprocessor: processing markdown string")

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
