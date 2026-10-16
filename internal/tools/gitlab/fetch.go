package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const gitlabDefaultTimeout = time.Minute

// GitLabTools provides GitLab merge request fetching as an LLM tool.
type GitLabTools struct {
	cfg *config.Config
}

// New creates a new GitLabTools instance.
func New(cfg *config.Config) *GitLabTools {
	return &GitLabTools{cfg: cfg}
}

// FetchMergeRequest fetches a GitLab merge request by project path and ID.
func (g *GitLabTools) FetchMergeRequest(projectPath string, mrID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitlabDefaultTimeout)
	defer cancel()

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   gitlabDefaultTimeout,
		Transport: tr,
	}

	// URL-encode the project path

	encodedProjectPath := url.PathEscape(projectPath)

	// Fetch MR details
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s", g.cfg.String("gitlab.base_url"), encodedProjectPath, mrID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to create request for MR %q/%q: %w", projectPath, mrID, err)
	}

	req.Header.Set("Authorization", "Bearer "+g.cfg.String("gitlab.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to fetch MR %q/%q: %w", projectPath, mrID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab: MR %q/%q returned status %d", projectPath, mrID, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var result struct {
		ID             int    `json:"id"`
		IID            int    `json:"iid"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		State          string `json:"state"`
		Created        string `json:"created_at"`
		Updated        string `json:"updated_at"`
		WebURL         string `json:"web_url"`
		SourceBranch   string `json:"source_branch"`
		TargetBranch   string `json:"target_branch"`
		WorkInProgress bool   `json:"work_in_progress"`
		Author         struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"author"`
		Labels    []string `json:"labels"`
		Milestone struct {
			Title string `json:"title"`
		} `json:"milestone"`
		Draft             bool   `json:"draft"`
		ChangesCount      string `json:"changes_count"`
		FirstCommitAuthor string `json:"first_commit_author"`
		FirstCommitSHA    string `json:"first_commit_sha"`
		LastCommit        struct {
			AuthorName  string `json:"author_name"`
			AuthorEmail string `json:"author_email"`
			CreatedAt   string `json:"created_at"`
		} `json:"last_commit"`
		Project struct {
			Name              string `json:"name"`
			WebURL            string `json:"web_url"`
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "gitlab: MR fetched", "project_path", projectPath, "mr_id", mrID)

	// Fetch MR changes (diffs)
	changesURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/changes", g.cfg.String("gitlab.base_url"), encodedProjectPath, mrID)
	reqChanges, err := http.NewRequestWithContext(ctx, http.MethodGet, changesURL, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to create changes request for MR %q/%q: %w", projectPath, mrID, err)
	}

	reqChanges.Header.Set("Authorization", "Bearer "+g.cfg.String("gitlab.api_key"))
	reqChanges.Header.Set("Content-Type", "application/json")

	respChanges, err := client.Do(reqChanges)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to fetch MR changes %q/%q: %w", projectPath, mrID, err)
	}
	defer respChanges.Body.Close()

	if respChanges.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab: MR changes %q/%q returned status %d", projectPath, mrID, respChanges.StatusCode)
	}

	changesBody, err := io.ReadAll(respChanges.Body)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to read changes response body: %w", err)
	}

	var changesResult struct {
		Changes []struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
			AMode   string `json:"a_mode"`
			BMode   string `json:"b_mode"`
			Created bool   `json:"new_file"`
			Deleted bool   `json:"deleted_file"`
			Renamed bool   `json:"renamed_file"`
			Diff    string `json:"diff"`
		} `json:"changes"`
	}

	if err := json.Unmarshal(changesBody, &changesResult); err != nil {
		return "", fmt.Errorf("gitlab: failed to parse changes response: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "gitlab: MR changes fetched", "project_path",
		projectPath, "mr_id", mrID, "changes_count", len(changesResult.Changes))

	var sb strings.Builder
	sb.WriteString("Project: ")
	sb.WriteString(projectPath)
	sb.WriteString("\nTitle: ")
	sb.WriteString(result.Title)
	sb.WriteString("\nStatus: ")
	sb.WriteString(result.State)
	sb.WriteString("\nAuthor: ")
	sb.WriteString(result.Author.Username)
	sb.WriteString("\nSource Branch: ")
	sb.WriteString(result.SourceBranch)
	sb.WriteString("\nTarget Branch: ")
	sb.WriteString(result.TargetBranch)
	if result.Milestone.Title != "" {
		sb.WriteString("\nMilestone: ")
		sb.WriteString(result.Milestone.Title)
	}
	if len(result.Labels) > 0 {
		sb.WriteString("\nLabels: ")
		for i, label := range result.Labels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(label)
		}
	}
	if result.Draft {
		sb.WriteString("\nDraft: true")
	}
	if result.WorkInProgress {
		sb.WriteString("\nWIP: true")
	}
	if result.ChangesCount != "" {
		sb.WriteString("\nChanges: ")
		sb.WriteString(result.ChangesCount)
	}
	sb.WriteString("\nCreated: ")
	sb.WriteString(result.Created)
	sb.WriteString("\nUpdated: ")
	sb.WriteString(result.Updated)
	sb.WriteString("\nWeb URL: ")
	sb.WriteString(result.WebURL)
	sb.WriteString("\n\nDescription:\n")
	sb.WriteString(result.Description)

	if len(changesResult.Changes) > 0 {
		sb.WriteString("\n\nFile Changes:\n")
		for i, diff := range changesResult.Changes {
			fmt.Fprintf(&sb, "%d. ", i+1)
			if diff.Deleted {
				sb.WriteString("Deleted: ")
				sb.WriteString(diff.OldPath)
			} else if diff.Created {
				sb.WriteString("Created: ")
				sb.WriteString(diff.NewPath)
			} else if diff.Renamed {
				sb.WriteString("Renamed: ")
				sb.WriteString(diff.OldPath)
				sb.WriteString(" -> ")
				sb.WriteString(diff.NewPath)
			} else {
				sb.WriteString("Modified: ")
				sb.WriteString(diff.NewPath)
			}
			sb.WriteString("\n")
			sb.WriteString("Diff:\n")
			sb.WriteString(diff.Diff)
			sb.WriteString("\n\n")
		}
	}

	return sb.String(), nil
}

// Definition returns the OpenAI tool schema for gitlab_fetch_mr.
func (g *GitLabTools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "gitlab_fetch_mr",
			Description: "Fetch a GitLab merge request and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')",
					},
					"mr_id": map[string]any{
						"type":        "string",
						"description": "Merge request ID (numeric ID or 'iid')",
					},
				},
				"required": []string{"project_path", "mr_id"},
			},
		},
	}
}

// StaticDefinition returns the OpenAI tool schema for gitlab_fetch_mr without
// requiring an initialised GitLabTools instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "gitlab_fetch_mr",
			Description: "Fetch a GitLab merge request and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')",
					},
					"mr_id": map[string]any{
						"type":        "string",
						"description": "Merge request ID (numeric ID or 'iid')",
					},
				},
				"required": []string{"project_path", "mr_id"},
			},
		},
	}
}

// StaticDefinitionRepository returns the OpenAI tool schema for gitlab_repo without
// requiring an initialised GitLabTools instance. Used by the plugin for lazy init.
func StaticDefinitionRepository() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "gitlab_repo",
			Description: "Fetch GitLab repository information and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')",
					},
				},
				"required": []string{"project_path"},
			},
		},
	}
}

// StaticDefinitionFile returns the OpenAI tool schema for gitlab_file without
// requiring an initialised GitLabTools instance. Used by the plugin for lazy init.
func StaticDefinitionFile() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "gitlab_file",
			Description: "Fetch a file from GitLab repository and return its content",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'main')",
					},
					"file": map[string]any{
						"type":        "string",
						"description": "Path to the file in the repository",
					},
				},
				"required": []string{"project_path", "file"},
			},
		},
	}
}

// StaticDefinitionTree returns the OpenAI tool schema for gitlab_tree without
// requiring an initialised GitLabTools instance. Used by the plugin for lazy init.
func StaticDefinitionTree() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "gitlab_tree",
			Description: "List repository contents at root or specified path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'main')",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Path to directory (default: '' for root)",
					},
				},
				"required": []string{"project_path"},
			},
		},
	}
}

// GitLabRepository represents GitLab repository information.
type GitLabRepository struct {
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

// GitLabFile represents a file in GitLab repository.
type GitLabFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	Encoding    string `json:"encoding"`
	Content     string `json:"content,omitempty"`
	Checkout    string `json:"checkout"`
	Reference   string `json:"ref"`
	ExternalURL string `json:"external_url"`
}

// GitLabTreeItem represents an item in GitLab directory tree.
type GitLabTreeItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	Target string `json:"target"`
}

// FetchRepository fetches GitLab repository information and returns Markdown content.
func (g *GitLabTools) FetchRepository(projectPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(projectPath)

	// Fetch repository details
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", g.cfg.String("gitlab.base_url"), encodedProjectPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to create request for project %q: %w", projectPath, err)
	}

	req.Header.Set("Authorization", "Bearer "+g.cfg.String("gitlab.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to fetch project %q: %w", projectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab: project %q returned status %d", projectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var repoData GitLabRepository
	if err := json.Unmarshal(body, &repoData); err != nil {
		return "", fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "gitlab_repo", "project_path", projectPath)

	return g.formatRepoMarkdown(&repoData), nil
}

// formatRepoMarkdown formats repository information as Markdown.
func (g *GitLabTools) formatRepoMarkdown(repoData *GitLabRepository) string {
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

// FetchFile fetches a file from GitLab repository.
func (g *GitLabTools) FetchFile(projectPath string, branch string, file string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path and file path
	encodedProjectPath := url.PathEscape(projectPath)
	encodedFilePath := url.PathEscape(file)

	// Build API URL
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s?ref=%s",
		g.cfg.String("gitlab.base_url"), encodedProjectPath, encodedFilePath, branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to create request for file %q in project %q: %w",
			file, projectPath, err)
	}

	req.Header.Set("Authorization", "Bearer "+g.cfg.String("gitlab.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to fetch file %q in project %q: %w",
			file, projectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab: file %q in project %q returned status %d",
			file, projectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var fileData GitLabFile
	if err := json.Unmarshal(body, &fileData); err != nil {
		return "", fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "gitlab_file", "project_path", projectPath, "branch", branch, "file", file)

	if fileData.Content == "" {
		return "", fmt.Errorf("gitlab: file %s has no content", file)
	}

	var content string
	if fileData.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileData.Content)
		if err != nil {
			return "", fmt.Errorf("gitlab: failed to decode file %s: %w", file, err)
		}
		content = string(decoded)
	} else {
		content = fileData.Content
	}

	return content, nil
}

// FetchTree lists repository contents at the root or specified path.
func (g *GitLabTools) FetchTree(projectPath string, branch string, path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(projectPath)

	// Build API URL
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s",
		g.cfg.String("gitlab.base_url"), encodedProjectPath, branch)

	// Add path parameter if specified
	if path != "" {
		encodedPath := url.PathEscape(path)
		apiURL += fmt.Sprintf("&path=%s", encodedPath)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to create request for tree %q in project %q: %w",
			path, projectPath, err)
	}

	req.Header.Set("Authorization", "Bearer "+g.cfg.String("gitlab.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to fetch tree %q in project %q: %w",
			path, projectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab: tree %q in project %q returned status %d",
			path, projectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var treeItems []GitLabTreeItem
	if err := json.Unmarshal(body, &treeItems); err != nil {
		return "", fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "gitlab_tree", "project_path", projectPath, "branch", branch, "path", path)

	return g.formatTreeMarkdown(treeItems, path), nil
}

// formatTreeMarkdown formats directory tree as Markdown.
func (g *GitLabTools) formatTreeMarkdown(treeItems []GitLabTreeItem, path string) string {
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
			// For files, we would need to fetch individual file info for size
			// For now, just show "file"
			sizeStr = "file"
		case "tree":
			sizeStr = "dir"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s |\n", item.Name, item.Type, sizeStr)
	}

	sb.WriteString("\n")
	return sb.String()
}

// doRequest performs an HTTP request with GitLab API headers.
func (g *GitLabTools) doRequest(req *http.Request) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   gitlabDefaultTimeout,
		Transport: tr,
	}

	// Add Accept header
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch %q: %w", req.URL.String(), err)
	}

	return resp, nil
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (g *GitLabTools) Dispatch(name string, args string) string {
	switch name {
	case "gitlab_fetch_mr":
		var params struct {
			ProjectPath string `json:"project_path"`
			MRID        string `json:"mr_id"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchMergeRequest(params.ProjectPath, params.MRID)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "gitlab_repo":
		var params struct {
			ProjectPath string `json:"project_path"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchRepository(params.ProjectPath)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "gitlab_file":
		var params struct {
			ProjectPath string `json:"project_path"`
			Branch      string `json:"branch"`
			File        string `json:"file"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchFile(params.ProjectPath, params.Branch, params.File)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "gitlab_tree":
		var params struct {
			ProjectPath string `json:"project_path"`
			Branch      string `json:"branch"`
			Path        string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchTree(params.ProjectPath, params.Branch, params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	default:
		return "error: unknown tool " + name
	}
}
