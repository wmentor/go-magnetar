package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const gitlabDefaultTimeout = time.Minute

// GitLabRepoParams represents parameters for the gitlab_repo tool.
type GitLabRepoParams struct {
	ProjectPath string `json:"project_path" jsonschema_description:"GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')"`
}

// Tool provides gitlab_repo operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "gitlab_repo"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Fetch GitLab repository information and return its details in Markdown format"
}

// Exec executes the gitlab_repo tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *GitLabRepoParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab_repo", "project_path", params.ProjectPath)

	ctx, cancel := context.WithTimeout(ctx, gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(params.ProjectPath)

	// Fetch repository details
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", t.cfg.String("gitlab.base_url"), encodedProjectPath)

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create request for project %q: %w", params.ProjectPath, err)
	}

	request.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch project %q: %w", params.ProjectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: project %q returned status %d", params.ProjectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var repoData struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		WebURL            string `json:"web_url"`
		AvatarURL         string `json:"avatar_url"`
		StarCount         int    `json:"star_count"`
		ForksCount        int    `json:"forks_count"`
		OpenIssuesCount   int    `json:"open_issues_count"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
		SSHURLToRepo      string `json:"ssh_url_to_repo"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		LastActivityAt    string `json:"last_activity_at"`
	}

	if err := json.Unmarshal(body, &repoData); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: formatRepoMarkdown(&repoData),
	}, nil
}

// formatRepoMarkdown formats repository information as Markdown.
func formatRepoMarkdown(repoData *struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	AvatarURL         string `json:"avatar_url"`
	StarCount         int    `json:"star_count"`
	ForksCount        int    `json:"forks_count"`
	OpenIssuesCount   int    `json:"open_issues_count"`
	DefaultBranch     string `json:"default_branch"`
	Visibility        string `json:"visibility"`
	SSHURLToRepo      string `json:"ssh_url_to_repo"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	LastActivityAt    string `json:"last_activity_at"`
},
) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", repoData.Name)
	fmt.Fprintf(&sb, "**Project Path:** %s\n\n", repoData.PathWithNamespace)
	fmt.Fprintf(&sb, "**Default Branch:** %s\n\n", repoData.DefaultBranch)
	fmt.Fprintf(&sb, "**Visibility:** %s\n\n", repoData.Visibility)
	fmt.Fprintf(&sb, "**Forks:** %d | **Stars:** %d | **Open Issues:** %d\n\n",
		repoData.ForksCount, repoData.StarCount, repoData.OpenIssuesCount)
	if repoData.Description != "" {
		fmt.Fprintf(&sb, "**Description:**\n%s\n\n", repoData.Description)
	}
	fmt.Fprintf(&sb, "**Web URL:** %s\n\n", repoData.WebURL)
	fmt.Fprintf(&sb, "**SSH URL:** %s\n\n", repoData.SSHURLToRepo)
	fmt.Fprintf(&sb, "**HTTP URL:** %s\n\n", repoData.HTTPURLToRepo)
	if repoData.LastActivityAt != "" {
		fmt.Fprintf(&sb, "**Last Activity:** %s\n", repoData.LastActivityAt)
	}

	return sb.String()
}
