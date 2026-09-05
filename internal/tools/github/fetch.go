package github

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const githubDefaultTimeout = time.Minute

// GitHubTools provides GitHub repository fetching as an LLM tool.
type GitHubTools struct {
	cfg *config.Config
}

// New creates a new GitHubTools instance.
func New(cfg *config.Config) *GitHubTools {
	return &GitHubTools{cfg: cfg}
}

// parseGitHubRepoURL extracts owner and repo from GitHub repository URL.
func parseGitHubRepoURL(repoURL string) (string, string, error) {
	// Remove protocol prefix if present
	repoURL = strings.TrimPrefix(repoURL, "https://github.com/")
	repoURL = strings.TrimPrefix(repoURL, "http://github.com/")
	repoURL = strings.TrimPrefix(repoURL, "github.com/")

	// Split into owner and repo
	parts := strings.Split(strings.Trim(repoURL, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repository URL format")
	}

	owner := parts[0]
	repo := parts[1]

	// Remove .git suffix if present
	repo = strings.TrimSuffix(repo, ".git")

	return owner, repo, nil
}

// FetchRepository fetches GitHub repository information and returns Markdown content.
func (g *GitHubTools) FetchRepository(repo string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_repo: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_repo", "repo", repo)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	// Fetch repository details
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repoName)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_repo: failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_repo: repository %s/%s returned status %d: %s", owner, repoName, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_repo: failed to read response body: %w", err)
	}

	var repoData struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
		License       *struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"license"`
		Language        string   `json:"language"`
		ForksCount      int      `json:"forks_count"`
		Archived        bool     `json:"archived"`
		StargazersCount int      `json:"stargazers_count"`
		OpenIssuesCount int      `json:"open_issues_count"`
		Topics          []string `json:"topics"`
		Description     string   `json:"description"`
		HTMLURL         string   `json:"html_url"`
	}

	if err := json.Unmarshal(body, &repoData); err != nil {
		return "", fmt.Errorf("github_repo: failed to parse response: %w", err)
	}

	// Fetch README
	readmeContent, err := g.fetchReadme(owner, repoName)
	if err != nil {
		printer.ToolCall(printer.IconError, "github_repo: failed to fetch README", "repo", repo, "err", err)
		// Continue without README
	}

	return g.formatRepoMarkdown(repoData, readmeContent), nil
}

func (g *GitHubTools) formatRepoMarkdown(repoData any, readme string) string {
	// Type assertion for repoData
	rd := repoData.(struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
		License       *struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"license"`
		Language        string   `json:"language"`
		ForksCount      int      `json:"forks_count"`
		Archived        bool     `json:"archived"`
		StargazersCount int      `json:"stargazers_count"`
		OpenIssuesCount int      `json:"open_issues_count"`
		Topics          []string `json:"topics"`
		Description     string   `json:"description"`
		HTMLURL         string   `json:"html_url"`
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", rd.FullName))
	sb.WriteString(fmt.Sprintf("**Default Branch:** %s\n\n", rd.DefaultBranch))
	if rd.License != nil {
		sb.WriteString(fmt.Sprintf("**License:** %s\n\n", rd.License.Name))
	}
	if rd.Language != "" {
		sb.WriteString(fmt.Sprintf("**Language:** %s\n\n", rd.Language))
	}
	sb.WriteString(fmt.Sprintf("**Forks:** %d | **Stars:** %d | **Open Issues:** %d\n\n",
		rd.ForksCount, rd.StargazersCount, rd.OpenIssuesCount))
	if rd.Archived {
		sb.WriteString("**⚠️ This repository is archived**\n\n")
	}
	if len(rd.Topics) > 0 {
		sb.WriteString(fmt.Sprintf("**Topics:** %s\n\n", strings.Join(rd.Topics, ", ")))
	}
	if rd.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", rd.Description))
	}
	if readme != "" {
		sb.WriteString(fmt.Sprintf("**README:**\n%s\n", readme))
	}
	return sb.String()
}

func (g *GitHubTools) fetchReadme(owner string, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github: failed to fetch README: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// README might not exist
		if resp.StatusCode == http.StatusNotFound {
			return "", nil
		}
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github: README fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github: failed to read README response: %w", err)
	}

	var readmeData struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Sha         string `json:"sha"`
		Size        int    `json:"size"`
		Encoding    string `json:"encoding"`
		Content     string `json:"content"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		GitURL      string `json:"git_url"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
		Links       struct {
			Self string `json:"self"`
			Git  string `json:"git"`
			HTML string `json:"html"`
		} `json:"_links"`
	}

	if err := json.Unmarshal(body, &readmeData); err != nil {
		return "", fmt.Errorf("github: failed to parse README response: %w", err)
	}

	if readmeData.Content == "" {
		return "", nil
	}

	var content string
	if readmeData.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(readmeData.Content)
		if err != nil {
			return "", fmt.Errorf("github: failed to decode README: %w", err)
		}
		content = string(decoded)
	} else {
		content = readmeData.Content
	}

	// Clean up the content (remove metadata lines if any)
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		// Skip GitHub's generated header
		if strings.HasPrefix(line, "<!--") && strings.HasSuffix(line, "-->") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n"), nil
}

// FetchFile fetches a file from GitHub repository.
func (g *GitHubTools) FetchFile(repo string, branch string, file string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_file: failed to parse repository URL: %w", err)
	}

	if branch == "" {
		branch = "master"
	}

	printer.ToolCall(printer.IconSearch, "github_file", "repo", repo, "branch", branch, "file", file)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	// URL-encode the file path
	encodedPath := url.PathEscape(file)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repoName, encodedPath, branch)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_file: failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_file: file %s returned status %d: %s", file, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_file: failed to read response body: %w", err)
	}

	var fileData struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Sha         string `json:"sha"`
		Size        int    `json:"size"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		GitURL      string `json:"git_url"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
		Encoding    string `json:"encoding,omitempty"`
		Content     string `json:"content,omitempty"`
		Links       struct {
			Self string `json:"self"`
			Git  string `json:"git"`
			HTML string `json:"html"`
		} `json:"_links"`
	}

	if err := json.Unmarshal(body, &fileData); err != nil {
		return "", fmt.Errorf("github_file: failed to parse response: %w", err)
	}

	if fileData.Content == "" {
		return "", fmt.Errorf("github_file: file %s has no content", file)
	}

	var content string
	if fileData.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileData.Content)
		if err != nil {
			return "", fmt.Errorf("github_file: failed to decode file %s: %w", file, err)
		}
		content = string(decoded)
	} else {
		content = fileData.Content
	}

	return content, nil
}

// FetchTree lists repository contents at the root or specified path.
func (g *GitHubTools) FetchTree(repo string, branch string, path string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_tree: failed to parse repository URL: %w", err)
	}

	if branch == "" {
		branch = "master"
	}

	printer.ToolCall(printer.IconSearch, "github_tree", "repo", repo, "branch", branch, "path", path)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	// Build API URL
	var apiURL string
	if path == "" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repoName, branch)
	} else {
		encodedPath := url.PathEscape(path)
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?path=%s&recursive=1", owner, repoName, branch, encodedPath)
	}

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_tree: failed to fetch directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_tree: directory fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_tree: failed to read response body: %w", err)
	}

	var treeResponse struct {
		Sha  string `json:"sha"`
		URL  string `json:"url"`
		Size int    `json:"size"`
		Tree []struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			Sha         string `json:"sha"`
			Size        int    `json:"size"`
			URL         string `json:"url"`
			HTMLURL     string `json:"html_url"`
			GitURL      string `json:"git_url"`
			DownloadURL string `json:"download_url"`
			Type        string `json:"type"`
			Encoding    string `json:"encoding,omitempty"`
			Links       struct {
				Self string `json:"self"`
				Git  string `json:"git"`
				HTML string `json:"html"`
			} `json:"_links"`
		} `json:"tree"`
	}

	if err := json.Unmarshal(body, &treeResponse); err != nil {
		return "", fmt.Errorf("github_tree: failed to parse response: %w", err)
	}

	files := treeResponse.Tree

	return g.formatTreeMarkdown(files, path), nil
}

func (g *GitHubTools) formatTreeMarkdown(files []struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Encoding    string `json:"encoding,omitempty"`
	Links       struct {
		Self string `json:"self"`
		Git  string `json:"git"`
		HTML string `json:"html"`
	} `json:"_links"`
}, path string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**Contents of %s**\n\n", path))
	sb.WriteString("| Name | Size |\n")
	sb.WriteString("|------|------|\n")

	for _, f := range files {
		sizeStr := ""
		if f.Type != "tree" {
			sizeStr = fmt.Sprintf("%d bytes", f.Size)
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", f.Path, sizeStr))
		}
	}

	return sb.String()
}

// fetchWithHeaders performs an HTTP request with GitHub API headers.
func (g *GitHubTools) fetchWithHeaders(ctx context.Context, method string, apiURL string) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   githubDefaultTimeout,
		Transport: tr,
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("github: failed to create request for %q: %w", apiURL, err)
	}

	req.Header.Set("Authorization", "Bearer "+g.cfg.String("github.api_key"))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	if g.cfg.String("github.base_url") != "" {
		apiURL = strings.Replace(apiURL, "https://api.github.com", g.cfg.String("github.base_url"), 1)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: failed to fetch %q: %w", apiURL, err)
	}

	return resp, nil
}

// Definition returns the OpenAI tool schema for github_repo.
func (g *GitHubTools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_repo",
			Description: "Fetch GitHub repository information and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
				},
				"required": []string{"repo"},
			},
		},
	}
}

// DefinitionFile returns the OpenAI tool schema for github_file.
func (g *GitHubTools) DefinitionFile() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_file",
			Description: "Fetch a file from GitHub repository and return its content",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'master')",
					},
					"file": map[string]any{
						"type":        "string",
						"description": "Path to the file in the repository",
					},
				},
				"required": []string{"repo", "file"},
			},
		},
	}
}

// DefinitionTree returns the OpenAI tool schema for github_tree.
func (g *GitHubTools) DefinitionTree() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_tree",
			Description: "List repository contents at root or specified path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'master')",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Path to directory (default: '' for root)",
					},
				},
				"required": []string{"repo"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (g *GitHubTools) Dispatch(name string, args string) string {
	switch name {
	case "github_repo":
		var params struct {
			Repo string `json:"repo"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchRepository(params.Repo)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	case "github_file":
		var params struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
			File   string `json:"file"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchFile(params.Repo, params.Branch, params.File)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	case "github_tree":
		var params struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchTree(params.Repo, params.Branch, params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	case "github_issue":
		var params struct {
			Repo  string `json:"repo"`
			Issue string `json:"issue"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchIssue(params.Repo, params.Issue)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	case "github_milestone":
		var params struct {
			Repo      string `json:"repo"`
			Milestone string `json:"milestone"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchMilestone(params.Repo, params.Milestone)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	default:
		return "error: unknown tool " + name
	}
}

func extractGitHubIssueURL(url string) (string, string, string, error) {
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", fmt.Errorf("not a GitHub issue URL")
}

func extractGitHubMilestoneURL(url string) (string, string, string, error) {
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/milestone/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", fmt.Errorf("not a GitHub milestone URL")
}

func (g *GitHubTools) FetchIssue(repo string, issueNum string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_issue: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_issue", "repo", repo, "issue", issueNum)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s", owner, repoName, issueNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_issue: failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_issue: issue #%s returned status %d: %s", issueNum, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_issue: failed to read response body: %w", err)
	}

	var issueData struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Locked bool   `json:"locked"`
		Body   string `json:"body"`
		User   struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"user"`
		Labels []struct {
			Name        string `json:"name"`
			Color       string `json:"color"`
			Description string `json:"description"`
		} `json:"labels"`
		Assignees []struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"assignees"`
		CommentCount int    `json:"comments"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		ClosedAt     string `json:"closed_at"`
		HTMLURL      string `json:"html_url"`
	}

	if err := json.Unmarshal(body, &issueData); err != nil {
		return "", fmt.Errorf("github_issue: failed to parse response: %w", err)
	}

	var comments []string
	if issueData.CommentCount > 0 {
		comments, err = g.fetchIssueComments(owner, repoName, issueNum)
		if err != nil {
			printer.ToolCall(printer.IconError, "github_issue: failed to fetch comments", "repo", repo, "issue", issueNum, "err", err)
		}
	}

	return g.formatIssueMarkdown(&issueData, comments), nil
}

func (g *GitHubTools) fetchIssueComments(owner string, repoName string, issueNum string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s/comments", owner, repoName, issueNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("github_issue: failed to fetch comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_issue: comments endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github_issue: failed to read comments response: %w", err)
	}

	var commentsData []struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
		User struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"user"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	if err := json.Unmarshal(body, &commentsData); err != nil {
		return nil, fmt.Errorf("github_issue: failed to parse comments response: %w", err)
	}

	var comments []string
	for _, comment := range commentsData {
		commentStr := fmt.Sprintf("### Comment by @%s\n\n%s\n\n**Created:** %s\n**Updated:** %s\n", comment.User.Login, comment.Body, comment.CreatedAt, comment.UpdatedAt)
		comments = append(comments, commentStr)
	}

	return comments, nil
}

func (g *GitHubTools) formatIssueMarkdown(issueData *struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Locked bool   `json:"locked"`
	Body   string `json:"body"`
	User   struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"user"`
	Labels []struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"labels"`
	Assignees []struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"assignees"`
	CommentCount int    `json:"comments"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	ClosedAt     string `json:"closed_at"`
	HTMLURL      string `json:"html_url"`
}, comments []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Issue #%d: %s\n\n", issueData.Number, issueData.Title))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", issueData.State))
	if issueData.Locked {
		sb.WriteString("**⚠️ This issue is locked**\n\n")
	}
	sb.WriteString(fmt.Sprintf("**Author:** @%s\n\n", issueData.User.Login))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", issueData.CreatedAt))
	sb.WriteString(fmt.Sprintf("**Updated:** %s\n\n", issueData.UpdatedAt))
	if issueData.ClosedAt != "" {
		sb.WriteString(fmt.Sprintf("**Closed:** %s\n\n", issueData.ClosedAt))
	}
	sb.WriteString(fmt.Sprintf("**URL:** %s\n\n", issueData.HTMLURL))

	if len(issueData.Labels) > 0 {
		var labelNames []string
		for _, label := range issueData.Labels {
			labelNames = append(labelNames, label.Name)
		}
		sb.WriteString(fmt.Sprintf("**Labels:** %s\n\n", strings.Join(labelNames, ", ")))
	}

	if len(issueData.Assignees) > 0 {
		var assigneeLogins []string
		for _, assignee := range issueData.Assignees {
			assigneeLogins = append(assigneeLogins, assignee.Login)
		}
		sb.WriteString(fmt.Sprintf("**Assignees:** %s\n\n", strings.Join(assigneeLogins, ", ")))
	}

	if issueData.Body != "" {
		sb.WriteString(fmt.Sprintf("## Description\n\n%s\n\n", issueData.Body))
	}

	if len(comments) > 0 {
		sb.WriteString(fmt.Sprintf("## Comments (%d)\n\n", len(comments)))
		for _, comment := range comments {
			sb.WriteString(comment)
			sb.WriteString("\n---\n\n")
		}
	}

	return sb.String()
}

func (g *GitHubTools) DefinitionIssue() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_issue",
			Description: "Fetch a GitHub issue and its comments, returns issue details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"issue": map[string]any{
						"type":        "string",
						"description": "Issue number (e.g., '7')",
					},
				},
				"required": []string{"repo", "issue"},
			},
		},
	}
}

func (g *GitHubTools) StaticDefinitionIssue() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_issue",
			Description: "Fetch a GitHub issue and its comments, returns issue details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"issue": map[string]any{
						"type":        "string",
						"description": "Issue number (e.g., '7')",
					},
				},
				"required": []string{"repo", "issue"},
			},
		},
	}
}

// StaticDefinitionIssue returns the OpenAI tool schema for github_issue without
// requiring an initialised GitHubTools instance.
func StaticDefinitionIssue() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_issue",
			Description: "Fetch a GitHub issue and its comments, returns issue details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"issue": map[string]any{
						"type":        "string",
						"description": "Issue number (e.g., '7')",
					},
				},
				"required": []string{"repo", "issue"},
			},
		},
	}
}

func (g *GitHubTools) FetchMilestone(repo string, milestoneNum string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_milestone: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_milestone", "repo", repo, "milestone", milestoneNum)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/milestones/%s", owner, repoName, milestoneNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_milestone: failed to fetch milestone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_milestone: milestone #%s returned status %d: %s", milestoneNum, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_milestone: failed to read response body: %w", err)
	}

	var milestoneData struct {
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		LabelsURL   string `json:"labels_url"`
		ID          int    `json:"id"`
		Number      int    `json:"number"`
		State       string `json:"state"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Creator     struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"creator"`
		OpenIssues   int    `json:"open_issues"`
		ClosedIssues int    `json:"closed_issues"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		DueOn        string `json:"due_on"`
	}

	if err := json.Unmarshal(body, &milestoneData); err != nil {
		return "", fmt.Errorf("github_milestone: failed to parse response: %w", err)
	}

	var issues []struct {
		URL    string `json:"url"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		User struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"user"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		ClosedAt  string `json:"closed_at"`
		HTMLURL   string `json:"html_url"`
	}

	issuesURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=all&milestone=%s", owner, repoName, milestoneNum)
	respIssues, err := g.fetchWithHeaders(ctx, http.MethodGet, issuesURL)
	if err != nil {
		printer.ToolCall(printer.IconError, "github_milestone: failed to fetch issues", "err", err)
	} else if respIssues.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respIssues.Body)
		printer.ToolCall(printer.IconError, "github_milestone: issues endpoint returned error", "status", respIssues.StatusCode, "body", string(body))
		respIssues.Body.Close()
	} else {
		defer respIssues.Body.Close()
		issuesBody, err := io.ReadAll(respIssues.Body)
		if err != nil {
			printer.ToolCall(printer.IconError, "github_milestone: failed to read issues response", "err", err)
		} else {
			if err := json.Unmarshal(issuesBody, &issues); err != nil {
				printer.ToolCall(printer.IconError, "github_milestone: failed to parse issues response", "err", err)
			} else {
				printer.ToolCall(printer.IconSearch, "github_milestone: fetched issues", "count", len(issues))
			}
		}
	}

	return g.formatMilestoneMarkdown(&milestoneData, issues), nil
}

func (g *GitHubTools) formatMilestoneMarkdown(milestoneData *struct {
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	LabelsURL   string `json:"labels_url"`
	ID          int    `json:"id"`
	Number      int    `json:"number"`
	State       string `json:"state"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Creator     struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"creator"`
	OpenIssues   int    `json:"open_issues"`
	ClosedIssues int    `json:"closed_issues"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	DueOn        string `json:"due_on"`
}, issues []struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	User struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ClosedAt  string `json:"closed_at"`
	HTMLURL   string `json:"html_url"`
}) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Milestone #%d: %s\n\n", milestoneData.Number, milestoneData.Title))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", milestoneData.State))
	sb.WriteString(fmt.Sprintf("**Creator:** @%s\n\n", milestoneData.Creator.Login))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", milestoneData.CreatedAt))
	sb.WriteString(fmt.Sprintf("**Updated:** %s\n\n", milestoneData.UpdatedAt))
	if milestoneData.DueOn != "" {
		sb.WriteString(fmt.Sprintf("**Due on:** %s\n\n", milestoneData.DueOn))
	}
	sb.WriteString(fmt.Sprintf("**URL:** %s\n\n", milestoneData.HTMLURL))

	if milestoneData.Description != "" {
		sb.WriteString(fmt.Sprintf("## Description\n\n%s\n\n", milestoneData.Description))
	}

	sb.WriteString(fmt.Sprintf("**Issues:** %d open, %d closed\n\n", milestoneData.OpenIssues, milestoneData.ClosedIssues))

	if len(issues) > 0 {
		sb.WriteString(fmt.Sprintf("## Issues (%d)\n\n", len(issues)))
		for _, issue := range issues {
			sb.WriteString(fmt.Sprintf("### [%d] %s\n\n", issue.Number, issue.Title))
			sb.WriteString(fmt.Sprintf("**URL:** %s\n\n", issue.HTMLURL))
			sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", issue.State))
			if issue.Body != "" {
				sb.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", issue.Body))
			}
			if len(issue.Labels) > 0 {
				var labelNames []string
				for _, label := range issue.Labels {
					labelNames = append(labelNames, label.Name)
				}
				sb.WriteString(fmt.Sprintf("**Labels:** %s\n\n", strings.Join(labelNames, ", ")))
			}
			sb.WriteString(fmt.Sprintf("**Created:** %s\n**Updated:** %s\n", issue.CreatedAt, issue.UpdatedAt))
			if issue.ClosedAt != "" {
				sb.WriteString(fmt.Sprintf("**Closed:** %s\n", issue.ClosedAt))
			}
			sb.WriteString("\n---\n\n")
		}
	}

	return sb.String()
}

// StaticDefinitionMilestone returns the OpenAI tool schema for github_milestone without
// requiring an initialised GitHubTools instance.
func StaticDefinitionMilestone() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_milestone",
			Description: "Fetch a GitHub milestone and all its issues, returns milestone details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"milestone": map[string]any{
						"type":        "string",
						"description": "Milestone number (e.g., '278')",
					},
				},
				"required": []string{"repo", "milestone"},
			},
		},
	}
}

// requiring an initialised GitHubTools instance.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_repo",
			Description: "Fetch GitHub repository information and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
				},
				"required": []string{"repo"},
			},
		},
	}
}

// StaticDefinitionFile returns the OpenAI tool schema for github_file without
// requiring an initialised GitHubTools instance.
func StaticDefinitionFile() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_file",
			Description: "Fetch a file from GitHub repository and return its content",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'master')",
					},
					"file": map[string]any{
						"type":        "string",
						"description": "Path to the file in the repository",
					},
				},
				"required": []string{"repo", "file"},
			},
		},
	}
}

// StaticDefinitionTree returns the OpenAI tool schema for github_tree without
// requiring an initialised GitHubTools instance.
func StaticDefinitionTree() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_tree",
			Description: "List repository contents at root or specified path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch name (default: 'master')",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Path to directory (default: '' for root)",
					},
				},
				"required": []string{"repo"},
			},
		},
	}
}
