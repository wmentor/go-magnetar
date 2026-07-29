package webplugin

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

func init() {
	plugin.Register("web", &Plugin{})
}

// Plugin wraps the web tools and exposes web_fetch as an LLM tool.
// The WebTools instance is created lazily on first use so that the
// agent's working-directory Root (set via plugin.SetRoot) is available.
type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	root  any // tracks last root used to detect changes
	tools *web.WebTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	hub.RegisterTool(plugin.LLMTool{
		Definition:   web.StaticDefinition,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			t, err := p.get()
			if err != nil {
				return "", err
			}
			return t.Dispatch("web_fetch", args), nil
		},
	})
	return nil
}

func (p *Plugin) get() (*web.WebTools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	root := plugin.GetRoot()
	if p.tools != nil && p.root == root {
		return p.tools, nil
	}
	t, err := web.New(p.state.Config, root)
	if err != nil {
		return nil, err
	}
	p.tools = t
	p.root = root
	return p.tools, nil
}
