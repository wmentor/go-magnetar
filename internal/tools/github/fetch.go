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

	default:
		return "error: unknown tool " + name
	}
}

// StaticDefinition returns the OpenAI tool schema for github_repo without
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
