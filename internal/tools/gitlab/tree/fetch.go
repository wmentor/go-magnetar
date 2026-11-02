package tree

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

// GitLabTreeParams represents parameters for the gitlab_tree tool.
type GitLabTreeParams struct {
	ProjectPath string `json:"project_path" jsonschema_description:"GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')"`
	Branch      string `json:"branch" jsonschema_description:"Branch name (default: 'main')"`
	Path        string `json:"path" jsonschema_description:"Path to directory (default: '' for root)"`
}

// Tool provides gitlab_tree operation as LLM tool.
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
	return "gitlab_tree"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "List repository contents at root or specified path"
}

// Exec executes the gitlab_tree tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *GitLabTreeParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	// Default branch to 'main' if not specified
	branch := params.Branch
	if branch == "" {
		branch = "main"
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab_tree", "project_path", params.ProjectPath, "branch", branch, "path", params.Path)

	ctx, cancel := context.WithTimeout(ctx, gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(params.ProjectPath)

	// Build API URL
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s",
		t.cfg.String("gitlab.base_url"), encodedProjectPath, branch)

	// Add path parameter if specified
	if params.Path != "" {
		encodedPath := url.PathEscape(params.Path)
		apiURL += fmt.Sprintf("&path=%s", encodedPath)
	}

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create request for tree %q in project %q: %w",
			params.Path, params.ProjectPath, err)
	}

	request.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch tree %q in project %q: %w",
			params.Path, params.ProjectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: tree %q in project %q returned status %d",
			params.Path, params.ProjectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var treeItems []struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Type   string `json:"type"`
		Mode   string `json:"mode"`
		Target string `json:"target"`
	}

	if err := json.Unmarshal(body, &treeItems); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: formatTreeMarkdown(treeItems, params.Path),
	}, nil
}

// formatTreeMarkdown formats directory tree as Markdown.
func formatTreeMarkdown(treeItems []struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	Target string `json:"target"`
}, path string,
) string {
	var sb strings.Builder

	if path == "" {
		fmt.Fprintf(&sb, "**Repository root contents**\n\n")
	} else {
		fmt.Fprintf(&sb, "**Contents of %s**\n\n", path)
	}

	sb.WriteString("| Name | Type | Size |\n")
	sb.WriteString("|------|------|------|\n")

	for _, item := range treeItems {
		sizeStr := "-"
		switch item.Type {
		case "blob":
			sizeStr = "file"
		case "tree":
			sizeStr = "dir"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s |\n", item.Name, item.Type, sizeStr)
	}

	sb.WriteString("\n")
	return sb.String()
}
