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
	fmt.Fprintf(&sb, "# %s\n\n", rd.FullName)
	fmt.Fprintf(&sb, "**Default Branch:** %s\n\n", rd.DefaultBranch)
	if rd.License != nil {
		fmt.Fprintf(&sb, "**License:** %s\n\n", rd.License.Name)
	}
	if rd.Language != "" {
		fmt.Fprintf(&sb, "**Language:** %s\n\n", rd.Language)
	}
	fmt.Fprintf(&sb, "**Forks:** %d | **Stars:** %d | **Open Issues:** %d\n\n",
		rd.ForksCount, rd.StargazersCount, rd.OpenIssuesCount)
	if rd.Archived {
		sb.WriteString("**⚠️ This repository is archived**\n\n")
	}
	if len(rd.Topics) > 0 {
		fmt.Fprintf(&sb, "**Topics:** %s\n\n", strings.Join(rd.Topics, ", "))
	}
	if rd.Description != "" {
		fmt.Fprintf(&sb, "**Description:**\n%s\n\n", rd.Description)
	}
	if readme != "" {
		fmt.Fprintf(&sb, "**README:**\n%s\n", readme)
	}
	return sb.String()
}

type SecurityAdvisory struct {
	GHSAID                string `json:"ghsa_id"`
	CveID                 string `json:"cve_id"`
	URL                   string `json:"url"`
	HTMLURL               string `json:"html_url"`
	Summary               string `json:"summary"`
	Description           string `json:"description"`
	Type                  string `json:"type"`
	Severity              string `json:"severity"`
	RepositoryAdvisoryURL string `json:"repository_advisory_url"`
	SourceCodeLocation    string `json:"source_code_location"`
	Identifiers           []struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"identifiers"`
	References       []string `json:"references"`
	PublishedAt      string   `json:"published_at"`
	UpdatedAt        string   `json:"updated_at"`
	GitHubReviewedAt string   `json:"github_reviewed_at"`
	NvdPublishedAt   *string  `json:"nvd_published_at"`
	WithdrawnAt      *string  `json:"withdrawn_at"`
	Vulnerabilities  []struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		VulnerableVersionRange string   `json:"vulnerable_version_range"`
		FirstPatchedVersion    string   `json:"first_patched_version"`
		VulnerableFunctions    []string `json:"vulnerable_functions"`
	} `json:"vulnerabilities"`
	CvssSeverities struct {
		CvssV3 struct {
			VectorString *string `json:"vector_string"`
			Score        float64 `json:"score"`
		} `json:"cvss_v3"`
		CvssV4 struct {
			VectorString *string `json:"vector_string"`
			Score        float64 `json:"score"`
		} `json:"cvss_v4"`
	} `json:"cvss_severities"`
	Cwes []struct {
		CweID string `json:"cwe_id"`
		Name  string `json:"name"`
	} `json:"cwes"`
	Credits []struct {
		User struct {
			Login     string `json:"login"`
			ID        int    `json:"id"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		Type string `json:"type"`
	} `json:"credits"`
	Cvss struct {
		VectorString *string  `json:"vector_string"`
		Score        *float64 `json:"score"`
	} `json:"cvss"`
}

func (g *GitHubTools) FetchAdvisory(advisoryID string) (string, error) {
	printer.ToolCall(printer.IconSearch, "github_advisory", "advisory", advisoryID)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/advisories/%s", advisoryID)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_advisory: failed to fetch advisory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_advisory: advisory %s returned status %d: %s", advisoryID, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_advisory: failed to read response body: %w", err)
	}

	var advisoryData SecurityAdvisory
	if err := json.Unmarshal(body, &advisoryData); err != nil {
		return "", fmt.Errorf("github_advisory: failed to parse response: %w", err)
	}

	return g.formatAdvisoryMarkdown(&advisoryData), nil
}

func (g *GitHubTools) formatAdvisoryMarkdown(advisory *SecurityAdvisory) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", advisory.Summary)
	fmt.Fprintf(&sb, "**GHSA ID:** %s\n\n", advisory.GHSAID)
	if advisory.CveID != "" {
		fmt.Fprintf(&sb, "**CVE ID:** %s\n\n", advisory.CveID)
	}
	fmt.Fprintf(&sb, "**Severity:** %s\n\n", advisory.Severity)
	fmt.Fprintf(&sb, "**Published:** %s\n\n", advisory.PublishedAt)
	fmt.Fprintf(&sb, "**Updated:** %s\n\n", advisory.UpdatedAt)
	if advisory.WithdrawnAt != nil {
		fmt.Fprintf(&sb, "**Withdrawn:** %s\n\n", *advisory.WithdrawnAt)
	}

	if advisory.Description != "" {
		fmt.Fprintf(&sb, "## Description\n\n%s\n\n", advisory.Description)
	}

	if len(advisory.References) > 0 {
		fmt.Fprintf(&sb, "## References\n\n")
		for _, ref := range advisory.References {
			fmt.Fprintf(&sb, "- %s\n", ref)
		}
		sb.WriteString("\n")
	}

	if len(advisory.Vulnerabilities) > 0 {
		fmt.Fprintf(&sb, "## Vulnerable Package\n\n")
		fmt.Fprintf(&sb, "**Package:** %s (%s)\n\n", advisory.Vulnerabilities[0].Package.Name, advisory.Vulnerabilities[0].Package.Ecosystem)

		var patchedVersions []string
		var vulnerableRanges []string
		for _, vuln := range advisory.Vulnerabilities {
			if vuln.FirstPatchedVersion != "" {
				patchedVersions = append(patchedVersions, vuln.FirstPatchedVersion)
			}
			if vuln.VulnerableVersionRange != "" {
				vulnerableRanges = append(vulnerableRanges, vuln.VulnerableVersionRange)
			}
		}

		if len(patchedVersions) > 0 {
			fmt.Fprintf(&sb, "**Patched Versions:** %s\n\n", strings.Join(patchedVersions, ", "))
		}

		if len(vulnerableRanges) > 0 {
			fmt.Fprintf(&sb, "**Vulnerable Versions:** %s\n\n", strings.Join(vulnerableRanges, ", "))
		}
	}

	return sb.String()
}

func (g *GitHubTools) FetchRepoAdvisories(repo string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_repo_advisories: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_repo_advisories", "repo", repo)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/security-advisories", owner, repoName)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_repo_advisories: failed to fetch advisories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_repo_advisories: advisories endpoint for %s/%s returned status %d: %s", owner, repoName, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_repo_advisories: failed to read response body: %w", err)
	}

	var advisories []SecurityAdvisory
	if err := json.Unmarshal(body, &advisories); err != nil {
		return "", fmt.Errorf("github_repo_advisories: failed to parse response: %w", err)
	}

	return g.formatRepoAdvisoriesMarkdown(advisories), nil
}

func (g *GitHubTools) formatRepoAdvisoriesMarkdown(advisories []SecurityAdvisory) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Security Advisories\n\n")

	if len(advisories) == 0 {
		fmt.Fprintf(&sb, "No security advisories found.\n\n")
		return sb.String()
	}

	fmt.Fprintf(&sb, "**Total Advisories:** %d\n\n", len(advisories))
	fmt.Fprintf(&sb, "## Advisories\n\n")
	sb.WriteString("| ID | Title | Severity | Published |\n")
	sb.WriteString("|----|-------|----------|----------|\n")

	for _, advisory := range advisories {
		severity := advisory.Severity
		if severity == "" {
			severity = "Unknown"
		}
		fmt.Fprintf(&sb, "| [%s](%s) | %s | %s | %s |\n", advisory.GHSAID, advisory.HTMLURL, advisory.Summary, severity, advisory.PublishedAt)
	}

	sb.WriteString("\n")

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

	fmt.Fprintf(&sb, "**Contents of %s**\n\n", path)
	sb.WriteString("| Name | Size |\n")
	sb.WriteString("|------|------|\n")

	for _, f := range files {
		sizeStr := ""
		if f.Type != "tree" {
			sizeStr = fmt.Sprintf("%d bytes", f.Size)
			fmt.Fprintf(&sb, "| %s | %s |\n", f.Path, sizeStr)
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

func (g *GitHubTools) FetchIssue(repo string, issueNum string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_issue: failed to parse repository URL: %w", err)
	}

	if strings.HasPrefix(issueNum, "GHSA-") {
		return g.FetchAdvisory(issueNum)
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

	fmt.Fprintf(&sb, "# Issue #%d: %s\n\n", issueData.Number, issueData.Title)
	fmt.Fprintf(&sb, "**Status:** %s\n\n", issueData.State)
	if issueData.Locked {
		sb.WriteString("**⚠️ This issue is locked**\n\n")
	}
	fmt.Fprintf(&sb, "**Author:** @%s\n\n", issueData.User.Login)
	fmt.Fprintf(&sb, "**Created:** %s\n\n", issueData.CreatedAt)
	fmt.Fprintf(&sb, "**Updated:** %s\n\n", issueData.UpdatedAt)
	if issueData.ClosedAt != "" {
		fmt.Fprintf(&sb, "**Closed:** %s\n\n", issueData.ClosedAt)
	}
	fmt.Fprintf(&sb, "**URL:** %s\n\n", issueData.HTMLURL)

	if len(issueData.Labels) > 0 {
		var labelNames []string
		for _, label := range issueData.Labels {
			labelNames = append(labelNames, label.Name)
		}
		fmt.Fprintf(&sb, "**Labels:** %s\n\n", strings.Join(labelNames, ", "))
	}

	if len(issueData.Assignees) > 0 {
		var assigneeLogins []string
		for _, assignee := range issueData.Assignees {
			assigneeLogins = append(assigneeLogins, assignee.Login)
		}
		fmt.Fprintf(&sb, "**Assignees:** %s\n\n", strings.Join(assigneeLogins, ", "))
	}

	if issueData.Body != "" {
		fmt.Fprintf(&sb, "## Description\n\n%s\n\n", issueData.Body)
	}

	if len(comments) > 0 {
		fmt.Fprintf(&sb, "## Comments (%d)\n\n", len(comments))
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

	fmt.Fprintf(&sb, "# Milestone #%d: %s\n\n", milestoneData.Number, milestoneData.Title)
	fmt.Fprintf(&sb, "**Status:** %s\n\n", milestoneData.State)
	fmt.Fprintf(&sb, "**Creator:** @%s\n\n", milestoneData.Creator.Login)
	fmt.Fprintf(&sb, "**Created:** %s\n\n", milestoneData.CreatedAt)
	fmt.Fprintf(&sb, "**Updated:** %s\n\n", milestoneData.UpdatedAt)
	if milestoneData.DueOn != "" {
		fmt.Fprintf(&sb, "**Due on:** %s\n\n", milestoneData.DueOn)
	}
	fmt.Fprintf(&sb, "**URL:** %s\n\n", milestoneData.HTMLURL)

	if milestoneData.Description != "" {
		fmt.Fprintf(&sb, "## Description\n\n%s\n\n", milestoneData.Description)
	}

	fmt.Fprintf(&sb, "**Issues:** %d open, %d closed\n\n", milestoneData.OpenIssues, milestoneData.ClosedIssues)

	if len(issues) > 0 {
		fmt.Fprintf(&sb, "## Issues (%d)\n\n", len(issues))
		for _, issue := range issues {
			fmt.Fprintf(&sb, "### [%d] %s\n\n", issue.Number, issue.Title)
			fmt.Fprintf(&sb, "**URL:** %s\n\n", issue.HTMLURL)
			fmt.Fprintf(&sb, "**Status:** %s\n\n", issue.State)
			if issue.Body != "" {
				fmt.Fprintf(&sb, "**Description:**\n%s\n\n", issue.Body)
			}
			if len(issue.Labels) > 0 {
				var labelNames []string
				for _, label := range issue.Labels {
					labelNames = append(labelNames, label.Name)
				}
				fmt.Fprintf(&sb, "**Labels:** %s\n\n", strings.Join(labelNames, ", "))
			}
			fmt.Fprintf(&sb, "**Created:** %s\n**Updated:** %s\n", issue.CreatedAt, issue.UpdatedAt)
			if issue.ClosedAt != "" {
				fmt.Fprintf(&sb, "**Closed:** %s\n", issue.ClosedAt)
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
