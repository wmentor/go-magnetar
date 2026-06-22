package ragplugin

import (
	"context"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

func init() {
	plugin.Register("rag", &Plugin{})
}

// Plugin wraps the RAG tools and exposes rag_search as an LLM tool.
type Plugin struct{}

func (p *Plugin) Init(s plugin.State, hub plugin.Hub) error {
	tools, err := rag.New(s.Config)
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
