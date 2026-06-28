package jiraplug

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/jira"
)

func init() {
	plugin.Register("jira", &Plugin{})
}

// Plugin wraps the JIRA tools and exposes jira_task_get as an LLM tool.
type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	tools *jira.JiraTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if s.Config.Bool("jira.disable") || s.Config.String("jira.base_url") == "" {
		return nil
	}

	hub.RegisterTool(plugin.LLMTool{
		Definition:   jira.StaticDefinition,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			t, err := p.get()
			if err != nil {
				return "", err
			}
			return t.Dispatch("jira_task_get", args), nil
		},
	})
	return nil
}

func (p *Plugin) get() (*jira.JiraTools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tools != nil {
		return p.tools, nil
	}
	p.tools = jira.New(p.state.Config)
	return p.tools, nil
}
