package chat

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
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

// ChatAgent is an interactive REPL-based chat agent backed by a RAG knowledge base.
type ChatAgent struct {
	cfg      *config.Config
	llm      *openai.Client
	rag      *rag.RAGTools
	messages []openai.ChatCompletionMessage
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

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	return &ChatAgent{
		cfg:      cfg,
		llm:      llmClient,
		rag:      ragTools,
		messages: messages,
	}, nil
}

// ask sends the user input to the LLM, handles tool calls, and returns the final answer.
func (a *ChatAgent) ask(userInput string) (string, error) {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	tools := []openai.Tool{
		a.rag.DefinitionSearch(),
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:     a.cfg.LLM.Model,
			Messages:  a.messages,
			Tools:     tools,
			MaxTokens: a.cfg.LLM.Context,
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

		fmt.Println(answer)
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("chat: stdin read error: %w", err)
	}

	return nil
}
