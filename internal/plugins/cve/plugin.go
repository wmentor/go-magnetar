package cveplugin

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/cve"
)

func init() {
	plugin.Register("cve", &Plugin{})
}

type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	tools *cve.CVETools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	hub.RegisterTool(plugin.LLMTool{
		Definition: cve.StaticDefinition,
		Execute: func(_ context.Context, args string) (string, error) {
			t, err := p.get()
			if err != nil {
				return "", err
			}
			return t.Dispatch("cve", args), nil
		},
	})

	return nil
}

func (p *Plugin) get() (*cve.CVETools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tools != nil {
		return p.tools, nil
	}
	t := cve.New(p.state.Config)
	p.tools = t
	return p.tools, nil
}
