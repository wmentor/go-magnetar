package rag

import (
	"context"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// RagSearchParams represents parameters for the rag_search tool.
type RagSearchParams struct {
	Query string `json:"query" jsonschema_description:"Search query for the knowledge base"`
}

// Tool provides rag_search operation as LLM tool.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
	state   *plugin.State
	store   Store // Store interface instance passed from outside
}

// New creates a new Tool instance with the given store.
func New(cfg *config.Config, p printer.Printer, store Store, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
		store:   store,
		state:   state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "rag_search"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Search the knowledge base for relevant information"
}

// Exec executes the rag_search tool and returns the search results.
func (t *Tool) Exec(ctx context.Context, params *RagSearchParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "rag_search", "query", params.Query)

	// Check if RAG is enabled
	if !t.cfg.Bool("rag.enable") {
		t.printer.Error("rag_search: RAG is disabled", "enable", t.cfg.Bool("rag.enable"))
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: RAG is disabled",
		}, nil
	}

	// Check if LLM base URL is configured
	if t.cfg.String("rag.llm.base_url") == "" {
		t.printer.Error("rag_search: rag.llm.base_url is not configured")
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: RAG LLM base_url is not configured",
		}, nil
	}

	// Perform search using the injected store
	results := t.store.RagSearch(params.Query)

	if results == "" {
		t.printer.ToolCall(printer.IconSearch, "rag_search: no results", "query", params.Query)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "no relevant information found",
		}, nil
	}

	t.printer.ToolCall(printer.IconDone, "rag_search: search completed", "query", params.Query)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: results,
	}, nil
}
