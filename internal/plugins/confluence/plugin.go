package confluenceplugin

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func init() {
	plugin.Register("confluence", &Plugin{})
}

// Plugin wraps the Confluence tools and exposes confluence_page_edit as an LLM tool.
type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	tools *confluence.ConfluenceTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("confluence.enable") || s.Config.String("confluence.base_url") == "" {
		return nil
	}

	tools, err := p.get()
	if err != nil {
		return err
	}

	hub.RegisterTool(plugin.LLMTool{
		Definition: confluence.StaticDefinitionEdit,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("confluence_page_edit", args), nil
		},
	})

	return nil
}

func (p *Plugin) get() (*confluence.ConfluenceTools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tools != nil {
		return p.tools, nil
	}
	p.tools = confluence.New(p.state.Config, p.state)
	return p.tools, nil
}
