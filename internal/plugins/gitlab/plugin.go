package gitlabplug

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab"
)

func init() {
	plugin.Register("gitlab", &Plugin{})
}

// Plugin wraps the GitLab tools and exposes gitlab_fetch_mr as an LLM tool.
type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	tools *gitlab.GitLabTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("gitlab.enable") || s.Config.String("gitlab.base_url") == "" {
		return nil
	}

	tools, err := p.get()
	if err != nil {
		return err
	}

	// Register gitlab_fetch_mr
	hub.RegisterTool(plugin.LLMTool{
		Definition: gitlab.StaticDefinition,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("gitlab_fetch_mr", args), nil
		},
	})

	// Register gitlab_repo
	hub.RegisterTool(plugin.LLMTool{
		Definition: gitlab.StaticDefinitionRepository,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("gitlab_repo", args), nil
		},
	})

	// Register gitlab_file
	hub.RegisterTool(plugin.LLMTool{
		Definition: gitlab.StaticDefinitionFile,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("gitlab_file", args), nil
		},
	})

	// Register gitlab_tree
	hub.RegisterTool(plugin.LLMTool{
		Definition: gitlab.StaticDefinitionTree,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("gitlab_tree", args), nil
		},
	})

	return nil
}

func (p *Plugin) get() (*gitlab.GitLabTools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tools != nil {
		return p.tools, nil
	}
	p.tools = gitlab.New(p.state.Config)
	return p.tools, nil
}
