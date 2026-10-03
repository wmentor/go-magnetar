package rag

import (
	"encoding/json"
	"errors"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

var (
	ErrInvalidConfig = errors.New("invalid value of config")
	ErrInvalidStore  = errors.New("invalid value of verctor store")
)

type Store interface {
	RagSearch(query string) string
	RagSave(content string, prepend string, part int) bool
}

// RAGTools provides RAG operations as LLM tools.
type RAGTools struct {
	cfg   *config.Config
	store Store
}

// New creates a new RAGTools instance.
func New(cfg *config.Config, store Store) (*RAGTools, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	if store == nil || store == Store(nil) {
		return nil, ErrInvalidStore
	}

	return &RAGTools{
		cfg:   cfg,
		store: store,
	}, nil
}

// DefinitionSearch returns the OpenAI tool schema for rag_search.
func (r *RAGTools) DefinitionSearch() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "rag_search",
			Description: "Search the knowledge base for relevant information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (r *RAGTools) RagSave(content string, prepend string, part int) bool {
	return r.store.RagSave(content, prepend, part)
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (r *RAGTools) Dispatch(name string, args string) string {
	switch name {
	case "rag_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.ToolCall(printer.IconError, "rag_search: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result := r.store.RagSearch(params.Query)
		if result == "" {
			return "no relevant information found"
		}
		return result

	default:
		return "error: unknown tool " + name
	}
}
