package githubplug

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/github"
)

func init() {
	plugin.Register("github", &Plugin{})
}

// Plugin wraps the GitHub tools and exposes github_repo, github_file, and github_tree as LLM tools.
type Plugin struct {
	mu      sync.Mutex
	state   *plugin.State
	tools   *github.GitHubTools
	enabled bool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if s.Config.Bool("github.disable") || s.Config.String("github.base_url") == "" {
		p.enabled = false
		return nil
	}

	p.enabled = true
	tools, err := p.get()
	if err != nil {
		return err
	}

	// Register github_repo tool
	hub.RegisterTool(plugin.LLMTool{
		Definition:   github.StaticDefinition,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("github_repo", args), nil
		},
	})

	// Register github_file tool
	hub.RegisterTool(plugin.LLMTool{
		Definition:   github.StaticDefinitionFile,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("github_file", args), nil
		},
	})

	// Register github_tree tool
	hub.RegisterTool(plugin.LLMTool{
		Definition:   github.StaticDefinitionTree,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("github_tree", args), nil
		},
	})

	// Register github_issue tool
	hub.RegisterTool(plugin.LLMTool{
		Definition:   github.StaticDefinitionIssue,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("github_issue", args), nil
		},
	})

	return nil
}

func (p *Plugin) get() (*github.GitHubTools, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tools != nil {
		return p.tools, nil
	}
	p.tools = github.New(p.state.Config)
	return p.tools, nil
}

func (p *Plugin) IsEnabled() bool {
	return p.enabled
}
