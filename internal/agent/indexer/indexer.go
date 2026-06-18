package indexer

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

const systemPrompt = `You are a document indexing agent. Your task is to process text files and store their content in a knowledge base.

When given a filename:
1. Read the file using the file_read tool.
2. Split the content into logical blocks, each no more than 500 tokens. Preserve semantic boundaries — split by paragraphs, sections, or logical units, not in the middle of a sentence or idea.
3. Save each block to the RAG using the rag_save tool.
4. Report how many blocks were saved.

When given a URL:
1. Fetch the web page using the web_fetch tool.
2. Split the content into logical blocks, each no more than 500 tokens. Preserve semantic boundaries — split by paragraphs, sections, or logical units, not in the middle of a sentence or idea.
3. Save each block to the RAG using the rag_save tool.
4. Report how many blocks were saved.

Do not summarize, paraphrase, or alter the content. Store the original text as-is.`

var (
	extRegExp = regexp.MustCompile(`^\.(md|txt)$`)
)

// Indexer is an AI agent that indexes files into the RAG knowledge base.
type Indexer struct {
	cfg     *config.Config
	llm     *openai.Client
	generic *generic.GenericTools
	rag     *rag.RAGTools
	web     *web.WebTools
}

// New creates a new Indexer instance.
func New(cfg *config.Config) (*Indexer, error) {
	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	ragTools, err := rag.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise RAG tools: %w", err)
	}

	webTools, err := web.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise web tools: %w", err)
	}

	return &Indexer{
		cfg:     cfg,
		llm:     llmClient,
		generic: generic.New(cfg),
		rag:     ragTools,
		web:     webTools,
	}, nil
}

// runAgentLoop executes the tool-use agentic loop until the model stops calling tools.
func (idx *Indexer) runAgentLoop(messages []openai.ChatCompletionMessage) error {
	tools := []openai.Tool{
		idx.generic.Definition(),
		idx.rag.DefinitionSave(),
		idx.web.Definition(),
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := idx.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:     idx.cfg.LLM.Model,
			Messages:  messages,
			Tools:     tools,
			MaxTokens: idx.cfg.LLM.Context,
		})
		cancel()

		if err != nil {
			return fmt.Errorf("indexer: LLM request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("indexer: empty response from LLM")
		}

		choice := resp.Choices[0]
		// Append assistant message to history.
		messages = append(messages, choice.Message)

		if choice.FinishReason == openai.FinishReasonToolCalls {
			// Process each tool call.
			for _, toolCall := range choice.Message.ToolCalls {
				name := toolCall.Function.Name
				args := toolCall.Function.Arguments

				slog.Debug("indexer: tool call", "tool", name, "args", args)

				var result string
				switch name {
				case "file_read":
					result = idx.generic.Dispatch(name, args)
				case "rag_save":
					result = idx.rag.Dispatch(name, args)
				case "web_fetch":
					result = idx.web.Dispatch(name, args)
				default:
					result = "error: unknown tool " + name
				}

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}
			// Continue the loop with updated messages.
			continue
		}

		// FinishReason == "stop" or other terminal reason.
		slog.Info("indexer: done", "response", choice.Message.Content)
		return nil
	}
}

// IndexFile indexes a single file into the RAG knowledge base.
func (idx *Indexer) IndexFile(filename string) error {
	slog.Info("indexer: indexing file", "file", filename)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Please index the file: %s", filename),
		},
	}

	return idx.runAgentLoop(messages)
}

// IndexURL indexes content from a URL into the RAG knowledge base.
func (idx *Indexer) IndexURL(url string) error {
	slog.Info("indexer: indexing URL", "url", url)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Please index the web page: %s", url),
		},
	}

	return idx.runAgentLoop(messages)
}

// IndexDirectory indexes all .md and .txt files in the given directory tree.
func (idx *Indexer) IndexDirectory(dir string) error {
	slog.Info("indexer: indexing directory", "dir", dir)

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("indexer: walk error", "path", path, "err", err)
			return nil // continue walking
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !extRegExp.MatchString(ext) {
			return nil
		}

		if err := idx.IndexFile(path); err != nil {
			slog.Error("indexer: failed to index file", "file", path, "err", err)
		}

		return nil
	})
}
