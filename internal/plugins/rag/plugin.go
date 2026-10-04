package ragplugin

import (
	"context"
	"fmt"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/store"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

func init() {
	plugin.Register("rag", &Plugin{})
}

// Plugin wraps the RAG tools and exposes rag_search as an LLM tool.
type Plugin struct{}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	if !s.Config.Bool("rag.enable") || s.Config.String("rag.llm.base_url") == "" {
		return nil
	}

	vecStore, err := store.NewStore(s.Config)
	if err != nil {
		return fmt.Errorf("init RAG store error: %w", err)
	}

	tools, err := rag.New(s.Config, vecStore)
	if err != nil {
		return err
	}
	hub.RegisterTool(plugin.LLMTool{
		Definition:   tools.DefinitionSearch,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("rag_search", args), nil
		},
	})
	return nil
}
