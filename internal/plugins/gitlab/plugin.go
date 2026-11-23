package gitlabplug

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/file"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/mr"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/repo"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/tree"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

func init() {
	plugin.Register("gitlab", &Plugin{})
}

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("gitlab.enable") || s.Config.String("gitlab.base_url") == "" {
		return nil
	}

	cfg := s.ConcreteConfig()

	// Register gitlab_fetch_mr tool
	mrTool := mr.New(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(mrTool.Name(), mrTool.Description(), mrTool.Exec)

	// Register gitlab_repo tool
	repoTool := repo.New(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(repoTool.Name(), repoTool.Description(), repoTool.Exec)

	// Register gitlab_file tool
	fileTool := file.New(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(fileTool.Name(), fileTool.Description(), fileTool.Exec)

	// Register gitlab_tree tool
	treeTool := tree.New(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(treeTool.Name(), treeTool.Description(), treeTool.Exec)

	s.WebProcStore.AddProcessorFunc(p.execWebFetch)

	return nil
}

func (p *Plugin) execWebFetch(ctx context.Context, urlStr string) (string, error) {
	myCfg := p.state.Config
	if !strings.HasPrefix(urlStr, myCfg.String("gitlab.base_url")) {
		return "", plugin.ErrNoWebProcessor
	}
	// Check for merge request
	if strings.Contains(urlStr, "/-/merge_requests") || strings.Contains(urlStr, "/-/mr") {
		projectPath, mrID, isListURL, err := web.ExtractProjectAndMergeRequestFromGitLabURL(urlStr)
		if err == nil && projectPath != "" {
			if isListURL {
				// URL points to list of MRs - fetch all opened MRs
				return p.fetchGitLabMergeRequestsList(ctx, projectPath)
			}
			if mrID != "" {
				return p.fetchGitLabMergeRequest(ctx, projectPath, mrID)
			}
		}
	}

	// Check for file (blob)
	if strings.Contains(urlStr, "/-/blob/") {
		projectPath, branch, file, err := web.ExtractGitLabFileURL(urlStr)
		if err == nil && projectPath != "" && file != "" {
			if branch == "" {
				branch = "main"
			}
			return p.fetchGitLabFile(ctx, projectPath, branch, file)
		}
	}

	// Check for tree (directory)
	if strings.Contains(urlStr, "/-/tree/") || strings.Contains(urlStr, "/-/commits/") {
		projectPath, branch, path, err := web.ExtractGitLabTreeURL(urlStr)
		if err == nil && projectPath != "" {
			if branch == "" {
				branch = "main"
			}
			return p.fetchGitLabTree(ctx, projectPath, branch, path)
		}
	}

	// Check for repository (root level)
	if !strings.Contains(urlStr, "/-/") {
		projectPath, err := web.ExtractGitLabRepoURL(urlStr)
		if err == nil && projectPath != "" {
			return p.fetchGitLabRepository(ctx, projectPath)
		}
	}

	return "", plugin.ErrNoWebProcessor
}

// fetchGitLabRepository fetches a GitLab repository and returns its content.
func (p *Plugin) fetchGitLabRepository(ctx context.Context, projectPath string) (string, error) {
	p.state.Printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab repository", "project_path", projectPath)
	tool := repo.New(p.state.Config, p.state.Printer, p.state.RequesterFactory)
	result, err := tool.Exec(ctx, &repo.GitLabRepoParams{
		ProjectPath: projectPath,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabTree fetches a GitLab directory tree and returns its content.
func (p *Plugin) fetchGitLabTree(ctx context.Context, projectPath string, branch string, path string) (string, error) {
	p.state.Printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab tree", "project_path", projectPath, "branch", branch, "path", path)
	tool := tree.New(p.state.Config, p.state.Printer, p.state.RequesterFactory)
	result, err := tool.Exec(ctx, &tree.GitLabTreeParams{
		ProjectPath: projectPath,
		Branch:      branch,
		Path:        path,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabFile fetches a GitLab file and returns its content.
func (p *Plugin) fetchGitLabFile(ctx context.Context, projectPath string, branch string, fileName string) (string, error) {
	p.state.Printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab file", "project_path", projectPath, "branch", branch, "file", fileName)
	tool := file.New(p.state.Config, p.state.Printer, p.state.RequesterFactory)
	result, err := tool.Exec(ctx, &file.GitLabFileParams{
		ProjectPath: projectPath,
		Branch:      branch,
		File:        fileName,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabMergeRequest fetches a GitLab merge request and returns its content.
func (p *Plugin) fetchGitLabMergeRequest(ctx context.Context, projectPath string, mrID string) (string, error) {
	p.state.Printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab merge request", "project_path", projectPath, "mr_id", mrID)
	tool := mr.New(p.state.Config, p.state.Printer, p.state.RequesterFactory)
	result, err := tool.Exec(ctx, &mr.GitLabMRParams{
		ProjectPath: projectPath,
		MRID:        mrID,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabMergeRequestsList fetches a list of GitLab merge requests and returns its content.
func (p *Plugin) fetchGitLabMergeRequestsList(ctx context.Context, projectPath string) (string, error) {
	p.state.Printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab merge requests list", "project_path", projectPath)
	tool := mr.New(p.state.Config, p.state.Printer, p.state.RequesterFactory)
	result, err := tool.FetchMergeRequests(ctx, &mr.FetchMergeRequestsParams{
		ProjectPath: projectPath,
		State:       "opened",
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}
