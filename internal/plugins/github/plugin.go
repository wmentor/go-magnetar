package githubplug

import (
	"context"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/github"
	"github.com/wmentor/go-magnetar/internal/tools/web"
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
	p.tool = github.New(s.ConcreteConfig(), s.Printer, s.RequesterFactory, s)

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

	// Register web fetch processor.
	s.WebProcStore.AddProcessorFunc(p.execWebFetch)

	return nil
}

func (p *Plugin) execWebFetch(ctx context.Context, urlStr string) (string, error) {
	if !strings.Contains(urlStr, "github.com/") {
		return "", plugin.ErrNoWebProcessor
	}
	myCfg := p.state.Config
	if strings.Contains(urlStr, "/blob/") {
		owner, repo, branch, file, err := web.ExtractGitHubFileURL(urlStr)
		if err == nil && owner != "" && repo != "" && file != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchFile(ctx, owner+"/"+repo, branch, file)
		}
	}
	// Check for pull request before tree/commits to avoid conflicts with /pull/123/commits
	if web.IsGitHubPullRequestURL(urlStr) {
		owner, repo, prNum, err := web.ExtractGitHubPullRequestURL(urlStr)
		if err == nil && owner != "" && repo != "" && prNum != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchPullRequest(ctx, owner+"/"+repo, prNum)
		}
	}
	// Check for pull requests list
	if web.IsGitHubPullRequestsURL(urlStr) {
		owner, repo, err := web.ExtractGitHubPullRequestsURL(urlStr)
		if err == nil && owner != "" && repo != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchPullRequestsList(ctx, owner+"/"+repo)
		}
	}
	if strings.Contains(urlStr, "/tree/") || strings.Contains(urlStr, "/commits/") || strings.HasSuffix(urlStr, "/") {
		owner, repo, branch, path, err := web.ExtractGitHubTreeURL(urlStr)
		if err == nil && owner != "" && repo != "" {
			if branch == "" {
				branch = "main"
			}
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchTree(ctx, owner+"/"+repo, branch, path)
		}
	}
	if strings.Contains(urlStr, "/issues/") {
		owner, repo, issueNum, err := web.ExtractGitHubIssueURL(urlStr)
		if err == nil && owner != "" && repo != "" && issueNum != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchIssue(ctx, owner+"/"+repo, issueNum)
		}
	}
	if strings.Contains(urlStr, "/milestone/") {
		owner, repo, milestoneNum, err := web.ExtractGitHubMilestoneURL(urlStr)
		if err == nil && owner != "" && repo != "" && milestoneNum != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchMilestone(ctx, owner+"/"+repo, milestoneNum)
		}
	}
	if strings.Contains(urlStr, "/advisories/") || strings.Contains(urlStr, "/security/advisories") {
		if strings.Contains(urlStr, "/security/advisories") && !strings.Contains(urlStr, "/advisories/GHSA-") {
			owner, repo, err := web.ExtractGitHubRepoURL(urlStr)
			if err == nil && owner != "" && repo != "" {
				ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
				return ghTools.FetchRepoAdvisories(ctx, owner+"/"+repo)
			}
		} else {
			advisoryID, err := web.ExtractGitHubAdvisoryURL(urlStr)
			if err == nil && advisoryID != "" {
				ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
				return ghTools.FetchAdvisory(ctx, advisoryID)
			}
		}
	}
	if strings.Contains(urlStr, "/releases/tag/") {
		owner, repo, tag, err := web.ExtractGitHubReleaseURL(urlStr)
		if err == nil && owner != "" && repo != "" && tag != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchRelease(ctx, owner+"/"+repo, tag)
		}
	}
	// Check for issues list URL
	if web.IsGitHubIssuesURL(urlStr) {
		owner, repo, err := web.ExtractGitHubIssuesURL(urlStr)
		if err == nil && owner != "" && repo != "" {
			ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
			return ghTools.FetchOpenIssues(ctx, owner+"/"+repo)
		}
	}
	owner, repo, err := web.ExtractGitHubRepoURL(urlStr)
	if err == nil && owner != "" && repo != "" {
		ghTools := github.New(myCfg, p.state.Printer, p.state.RequesterFactory, nil)
		return ghTools.FetchRepository(ctx, owner+"/"+repo)
	}

	return "", plugin.ErrNoWebProcessor
}
