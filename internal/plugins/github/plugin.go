package githubplug

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/github"
)

func init() {
	plugin.Register("github", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *github.GitHubTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s
	p.tool = github.New(s.Config, s.Printer, s.RequesterFactory, s)

	if !p.tool.Enabled() {
		return nil
	}

	// Register github_repo tool
	s.ToolStore.Register("github_repo", "Fetch GitHub repository information and return its details in Markdown format", p.tool.Exec)

	// Register github_file tool
	s.ToolStore.Register("github_file", "Fetch a file from GitHub repository and return its content", p.tool.ExecFile)

	// Register github_tree tool
	s.ToolStore.Register("github_tree", "List repository contents at root or specified path", p.tool.ExecTree)

	// Register github_issue tool
	s.ToolStore.Register("github_issue", "Fetch a GitHub issue and its comments, returns issue details in Markdown format", p.tool.ExecIssue)

	// Register github_milestone tool
	s.ToolStore.Register("github_milestone", "Fetch a GitHub milestone and all its issues, returns milestone details in Markdown format",
		p.tool.ExecMilestone)

	// Register github_pull_request tool
	s.ToolStore.Register("github_pull_request", "Fetch a GitHub pull request with all details including files changed, comments, and reviews",
		p.tool.ExecPullRequest)

	// Register github_release_info tool
	s.ToolStore.Register("github_release_info", "Fetch a GitHub release by tag name and return its details in Markdown format", p.tool.ExecRelease)

	return nil
}
