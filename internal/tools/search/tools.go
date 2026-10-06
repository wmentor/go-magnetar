package search

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/store"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

type SearchTools struct {
	cfg   *config.Config
	state *plugin.State
	web   *web.WebTools
	store rag.Store
}

func New(cfg *config.Config, state *plugin.State) (*SearchTools, error) {
	webTools, err := web.New(cfg, plugin.GetRoot(), state)
	if err != nil {
		return nil, err
	}

	var vecStore rag.Store
	if cfg.Bool("rag.enable") && cfg.String("rag.llm.base_url") != "" {
		vecStore, err = store.NewStore(cfg)
		if err != nil {
			return nil, fmt.Errorf("init RAG store error: %w", err)
		}
	}

	return &SearchTools{
		cfg:   cfg,
		state: state,
		web:   webTools,
		store: vecStore,
	}, nil
}

func (s *SearchTools) Search(query string) (string, error) {
	printer.ToolCall(printer.IconSearch, "search", "query", query)

	var webResult, ragResult string
	var webErr, ragErr error

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		webResult, webErr = s.web.WebSearch(query)
	}()

	go func() {
		defer wg.Done()
		if s.store != nil {
			ragResult = s.store.RagSearch(query)
			if ragResult == "" {
				ragErr = fmt.Errorf("no relevant information found")
			}
		} else {
			ragErr = fmt.Errorf("RAG not enabled")
		}
	}()

	wg.Wait()

	var results []string
	if webErr == nil {
		results = append(results, "WEB SEARCH RESULTS:\n"+webResult)
	}
	if ragErr == nil {
		results = append(results, "RAG SEARCH RESULTS:\n"+ragResult)
	}

	if len(results) == 0 {
		return "no results found", nil
	}

	return strings.Join(results, "\n\n---\n\n"), nil
}

func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "search",
			Description: "Execute web and knowledge base search in parallel and return merged results",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query string",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (s *SearchTools) Dispatch(name string, args string) string {
	switch name {
	case "search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("search: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result, err := s.Search(params.Query)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return result
	default:
		return "error: unknown tool " + name
	}
}
