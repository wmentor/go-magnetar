package ragplugin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

var (
	ErrToolNotInitialized = errors.New("rag.search: tool is not initialized")
	ErrEmptySearchText    = errors.New("usage: /rag.search <search_text>")
)

func init() {
	plugin.Register("rag", &Plugin{})
}

type Plugin struct {
	tool *rag.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	// Only register rag_search if RAG is enabled
	cfg := s.Config
	if cfg.Bool("rag.enable") && cfg.String("rag.llm.base_url") != "" {
		// Create RAG tool with the store instance
		p.tool = rag.New(cfg, s.Printer, s.VectorStore, s)
		s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec, tools.OptionRateLimit(5))

		hub.RegisterChatCommand(plugin.ChatCommand{
			Name:    "rag.search",
			Aliases: []string{},
			Help:    "Search the RAG knowledge base for relevant information",
			Execute: p.executeSearch,
		})
	}
	return nil
}

func (p *Plugin) executeSearch(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	if p.tool == nil {
		return nil, ErrToolNotInitialized
	}

	searchText := strings.TrimSpace(params.Args)
	if searchText == "" {
		return nil, ErrEmptySearchText
	}

	rawResp, err := p.tool.Exec(ctx, &rag.RagSearchParams{Query: searchText})
	if err != nil {
		return nil, fmt.Errorf("rag.search: %w", err)
	}

	resp, ok := rawResp.(*tools.Response)
	if !ok {
		return nil, fmt.Errorf("rag.search: unexpected response type %T", rawResp)
	}

	return &plugin.ChatCommandResponse{
		Stdout: resp.Result,
	}, nil
}
