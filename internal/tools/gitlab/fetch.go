package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
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
	slog.Debug("gitlab: detected merge request", "project_path", projectPath, "mr_id", mrID)

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

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", fmt.Errorf("gitlab: decode MR %q/%q error: %w", projectPath, mrID, err1)
	}

	body, err := io.ReadAll(utf8)
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

	slog.Debug("gitlab: MR fetched", "project_path", projectPath, "mr_id", mrID)

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

	fmt.Println(sb.String())

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
