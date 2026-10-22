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
	// Only register rag_search if RAG is enabled
	cfg := s.Config
	if cfg.Bool("rag.enable") && cfg.String("rag.llm.base_url") != "" {
		// Create store instance
		var vecStore rag.Store
		if cfg.Bool("rag.enable") && cfg.String("rag.llm.base_url") != "" {
			var err error
			vecStore, err = store.NewStore(cfg)
			if err != nil {
				return fmt.Errorf("init RAG store error: %w", err)
			}
		}

		// Create RAGTools instance
		ragTools, err := rag.New(cfg, vecStore)
		if err != nil {
			return fmt.Errorf("init RAG tools error: %w", err)
		}

		// Register rag_search tool
		hub.RegisterTool(plugin.LLMTool{
			Definition:   ragTools.DefinitionSearch,
			IsSearchTool: true,
			Execute: func(_ context.Context, args string) (string, error) {
				return ragTools.Dispatch("rag_search", args), nil
			},
		})
	}
	return nil
}
