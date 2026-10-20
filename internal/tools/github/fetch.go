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
	rd := repoData.(struct { //nolint:forcetypeassert // possible later.
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
		return "", fmt.Errorf("github_repo_advisories: advisories endpoint for %s/%s returned status %d: %s",
			owner, repoName, resp.StatusCode, string(body))
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
}, path string,
) string {
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
func (g *GitHubTools) fetchWithHeaders(ctx context.Context, method string, apiURL string) (*http.Response, error) { //nolint:unparam // ok.
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

	case "github_pull_request":
		var params struct {
			Repo string `json:"repo"`
			PR   string `json:"pr"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchPullRequest(params.Repo, params.PR)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content

	case "github_release_info":
		var params struct {
			Repo string `json:"repo"`
			Tag  string `json:"tag"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := g.FetchRelease(params.Repo, params.Tag)
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
		commentStr := fmt.Sprintf("### Comment by @%s\n\n%s\n\n**Created:** %s\n**Updated:** %s\n",
			comment.User.Login, comment.Body, comment.CreatedAt, comment.UpdatedAt)
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
}, comments []string,
) string {
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
		labelNames := make([]string, 0, len(issueData.Labels))
		for _, label := range issueData.Labels {
			labelNames = append(labelNames, label.Name)
		}
		fmt.Fprintf(&sb, "**Labels:** %s\n\n", strings.Join(labelNames, ", "))
	}

	if len(issueData.Assignees) > 0 {
		assigneeLogins := make([]string, 0, len(issueData.Assignees))
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
},
) string {
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
				labelNames := make([]string, 0, len(issue.Labels))
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

// PullRequestFile represents a file changed in a pull request.
type PullRequestFile struct {
	SHA         string `json:"sha"`
	BlobURL     string `json:"blob_url"`
	RawURL      string `json:"raw_url"`
	ContentsURL string `json:"contents_url"`
	Path        string `json:"filename"`
	Status      string `json:"status"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Changes     int    `json:"changes"`
	Size        int    `json:"size"`
	Diff        string `json:"patch"` // GitHub uses "patch" for diff
}

// PullRequestComment represents a review comment on a PR.
type PullRequestComment struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	User struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Path      string `json:"path"`
	Position  int    `json:"position"`
	Line      int    `json:"line"`
	InReplyTo int    `json:"in_reply_to_id"`
}

// PullRequestReview represents a review on a PR.
type PullRequestReview struct {
	ID   int `json:"id"`
	User struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"user"`
	Body      string `json:"body"`
	State     string `json:"state"`
	Submitted string `json:"submitted_at"`
	CommitID  string `json:"commit_id"`
}

// PullRequest represents a GitHub pull request with all its details.
type PullRequest struct {
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
	Mergeable     *bool                `json:"mergeable"`
	CommentsCount int                  `json:"comments"`        // Number of review comments
	ReviewsCount  int                  `json:"review_comments"` // Number of reviews
	Files         []PullRequestFile    `json:"files"`
	Comments      []PullRequestComment `json:"-"` // populated separately from comments endpoint
	Reviews       []PullRequestReview  `json:"-"` // populated separately from reviews endpoint
	// Statistics from PR details
	Commits        int    `json:"commits"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	ChangedFiles   int    `json:"changed_files"`
	MergeCommitSHA string `json:"merge_commit_sha"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	ClosedAt       string `json:"closed_at"`
	MergedAt       string `json:"merged_at"`
	HTMLURL        string `json:"html_url"`
	Base           struct {
		Label string `json:"label"`
		Ref   string `json:"ref"`
		Repo  struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repo"`
	} `json:"base"`
	Head struct {
		Label string `json:"label"`
		Ref   string `json:"ref"`
		Repo  struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repo"`
	} `json:"head"`
}

// FetchPullRequest fetches a GitHub pull request with all its details including files, comments, and reviews.
func (g *GitHubTools) FetchPullRequest(repo string, prNum string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_pull_request: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_pull_request", "repo", repo, "pr", prNum)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	// Fetch PR details
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s", owner, repoName, prNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_pull_request: failed to fetch PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_pull_request: PR #%s returned status %d: %s", prNum, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_pull_request: failed to read response body: %w", err)
	}

	var prData PullRequest
	if err := json.Unmarshal(body, &prData); err != nil {
		return "", fmt.Errorf("github_pull_request: failed to parse response: %w", err)
	}

	// Fetch files changed in the PR
	files, err := g.fetchPullRequestFiles(owner, repoName, prNum)
	if err != nil {
		printer.ToolCall(printer.IconError, "github_pull_request: failed to fetch files", "repo", repo, "pr", prNum, "err", err)
	} else {
		prData.Files = files
		prData.Additions, prData.Deletions, prData.ChangedFiles = g.calculateChanges(files)
	}

	// Fetch review comments
	comments, err := g.fetchPullRequestComments(owner, repoName, prNum)
	if err != nil {
		printer.ToolCall(printer.IconError, "github_pull_request: failed to fetch comments", "repo", repo, "pr", prNum, "err", err)
	} else {
		prData.Comments = comments
	}

	// Fetch reviews
	reviews, err := g.fetchPullRequestReviews(owner, repoName, prNum)
	if err != nil {
		printer.ToolCall(printer.IconError, "github_pull_request: failed to fetch reviews", "repo", repo, "pr", prNum, "err", err)
	} else {
		prData.Reviews = reviews
	}

	return g.formatPullRequestMarkdown(&prData), nil
}

// fetchPullRequestFiles fetches files changed in a pull request.
//
//nolint:dupl // Similar structure to fetchPullRequestComments and fetchPullRequestReviews.
func (g *GitHubTools) fetchPullRequestFiles(owner string, repoName string, prNum string) ([]PullRequestFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/files", owner, repoName, prNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to fetch files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_pull_request: files endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to read files response: %w", err)
	}

	var files []PullRequestFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to parse files response: %w", err)
	}

	return files, nil
}

// fetchPullRequestComments fetches review comments on a pull request.
//
//nolint:dupl // Similar structure to fetchPullRequestFiles and fetchPullRequestReviews.
func (g *GitHubTools) fetchPullRequestComments(owner string, repoName string, prNum string) ([]PullRequestComment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/comments", owner, repoName, prNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to fetch comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_pull_request: comments endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to read comments response: %w", err)
	}

	var comments []PullRequestComment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to parse comments response: %w", err)
	}

	return comments, nil
}

// fetchPullRequestReviews fetches reviews on a pull request.
//
//nolint:dupl // Similar structure to fetchPullRequestFiles and fetchPullRequestComments.
func (g *GitHubTools) fetchPullRequestReviews(owner string, repoName string, prNum string) ([]PullRequestReview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/reviews", owner, repoName, prNum)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to fetch reviews: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_pull_request: reviews endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to read reviews response: %w", err)
	}

	var reviews []PullRequestReview
	if err := json.Unmarshal(body, &reviews); err != nil {
		return nil, fmt.Errorf("github_pull_request: failed to parse reviews response: %w", err)
	}

	return reviews, nil
}

// calculateChanges calculates total additions, deletions, and changed files from file list.
func (g *GitHubTools) calculateChanges(files []PullRequestFile) (int, int, int) {
	var additions, deletions int
	changedFiles := len(files)
	for _, file := range files {
		additions += file.Additions
		deletions += file.Deletions
	}
	return additions, deletions, changedFiles
}

// formatPullRequestMarkdown formats PR data as Markdown.
func (g *GitHubTools) formatPullRequestMarkdown(pr *PullRequest) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Pull Request #%d: %s\n\n", pr.Number, pr.Title)
	fmt.Fprintf(&sb, "**Status:** %s\n\n", pr.State)
	if pr.Locked {
		sb.WriteString("**⚠️ This PR is locked**\n\n")
	}
	fmt.Fprintf(&sb, "**Author:** @%s\n\n", pr.User.Login)
	fmt.Fprintf(&sb, "**Created:** %s\n\n", pr.CreatedAt)
	fmt.Fprintf(&sb, "**Updated:** %s\n\n", pr.UpdatedAt)
	if pr.ClosedAt != "" {
		fmt.Fprintf(&sb, "**Closed:** %s\n\n", pr.ClosedAt)
	}
	if pr.MergedAt != "" {
		fmt.Fprintf(&sb, "**Merged:** %s\n\n", pr.MergedAt)
	}
	fmt.Fprintf(&sb, "**URL:** %s\n\n", pr.HTMLURL)

	// Base and Head branches
	fmt.Fprintf(&sb, "**Base:** %s:%s\n\n", pr.Base.Repo.Owner.Login, pr.Base.Ref)
	fmt.Fprintf(&sb, "**Head:** %s:%s\n\n", pr.Head.Repo.Owner.Login, pr.Head.Ref)

	// Change statistics
	if pr.ChangedFiles > 0 {
		fmt.Fprintf(&sb, "**Changes:** %d files, %d additions, %d deletions\n\n",
			pr.ChangedFiles, pr.Additions, pr.Deletions)
	}

	// Labels
	if len(pr.Labels) > 0 {
		labelNames := make([]string, 0, len(pr.Labels))
		for _, label := range pr.Labels {
			labelNames = append(labelNames, label.Name)
		}
		fmt.Fprintf(&sb, "**Labels:** %s\n\n", strings.Join(labelNames, ", "))
	}

	// Merge status
	if pr.Mergeable != nil {
		if *pr.Mergeable {
			fmt.Fprintf(&sb, "**Mergeable:** Yes\n\n")
		} else {
			fmt.Fprintf(&sb, "**Mergeable:** No\n\n")
		}
	}

	// Description
	if pr.Body != "" {
		fmt.Fprintf(&sb, "## Description\n\n%s\n\n", pr.Body)
	}

	// Files changed
	if len(pr.Files) > 0 {
		fmt.Fprintf(&sb, "## Files Changed (%d)\n\n", len(pr.Files))
		sb.WriteString("| File | Changes | Status |\n")
		sb.WriteString("|------|---------|--------|\n")
		for _, file := range pr.Files {
			status := "modified"
			if len(file.Diff) > 0 {
				if strings.Contains(file.Diff, "new file") {
					status = "added"
				} else if strings.Contains(file.Diff, "deleted file") {
					status = "deleted"
				}
			}
			fmt.Fprintf(&sb, "| `%s` | +%d -%d | %s |\n", file.Path, file.Additions, file.Deletions, status)
		}
		sb.WriteString("\n")

		// Show detailed diff for each file
		for _, file := range pr.Files {
			if file.Diff != "" {
				fmt.Fprintf(&sb, "### %s\n\n", file.Path)
				fmt.Fprintf(&sb, "```diff\n%s\n```\n\n", file.Diff)
			}
		}
	}

	// Review comments
	if len(pr.Comments) > 0 {
		fmt.Fprintf(&sb, "## Review Comments (%d)\n\n", len(pr.Comments))
		for _, comment := range pr.Comments {
			fmt.Fprintf(&sb, "### Comment by @%s on %s (line %d)\n\n", comment.User.Login, comment.Path, comment.Line)
			fmt.Fprintf(&sb, "%s\n\n", comment.Body)
			fmt.Fprintf(&sb, "**Posted:** %s\n\n", comment.CreatedAt)
			sb.WriteString("---\n\n")
		}
	}

	// Reviews
	if len(pr.Reviews) > 0 {
		fmt.Fprintf(&sb, "## Reviews (%d)\n\n", len(pr.Reviews))
		for _, review := range pr.Reviews {
			statusEmoji := "⚪"
			switch review.State {
			case "approved":
				statusEmoji = "✅"
			case "changes_requested":
				statusEmoji = "🔄"
			case "commented":
				statusEmoji = "💬"
			case "rejected":
				statusEmoji = "❌"
			}
			fmt.Fprintf(&sb, "### %s Review by @%s\n\n", statusEmoji, review.User.Login)
			if review.Body != "" {
				fmt.Fprintf(&sb, "%s\n\n", review.Body)
			}
			fmt.Fprintf(&sb, "**Status:** %s\n", strings.Title(review.State))
			fmt.Fprintf(&sb, "**Submitted:** %s\n\n", review.Submitted)
			sb.WriteString("---\n\n")
		}
	}

	return sb.String()
}

// DefinitionPullRequest returns the OpenAI tool schema for github_pull_request.
func (g *GitHubTools) DefinitionPullRequest() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_pull_request",
			Description: "Fetch a GitHub pull request with all details including files changed, comments, and reviews",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"pr": map[string]any{
						"type":        "string",
						"description": "Pull request number (e.g., '7')",
					},
				},
				"required": []string{"repo", "pr"},
			},
		},
	}
}

// StaticDefinitionPullRequest returns the OpenAI tool schema for github_pull_request without
// requiring an initialised GitHubTools instance.
func StaticDefinitionPullRequest() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_pull_request",
			Description: "Fetch a GitHub pull request with all details including files changed, comments, and reviews",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"pr": map[string]any{
						"type":        "string",
						"description": "Pull request number (e.g., '7')",
					},
				},
				"required": []string{"repo", "pr"},
			},
		},
	}
}

// Release represents a GitHub release.
type Release struct {
	URL       string `json:"url"`
	AssetsURL string `json:"assets_url"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
	ID        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		UserViewType      string `json:"user_view_type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeID          string `json:"node_id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Draft           bool   `json:"draft"`
	Immutable       bool   `json:"immutable"`
	Prerelease      bool   `json:"prerelease"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PublishedAt     string `json:"published_at"`
	Assets          []struct {
		URL      string `json:"url"`
		ID       int    `json:"id"`
		NodeID   string `json:"node_id"`
		Name     string `json:"name"`
		Label    string `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			UserViewType      string `json:"user_view_type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string `json:"content_type"`
		State              string `json:"state"`
		Size               int    `json:"size"`
		Digest             string `json:"digest"`
		DownloadCount      int    `json:"download_count"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}

// FetchRelease fetches a GitHub release by tag name and returns its details in Markdown format.
func (g *GitHubTools) FetchRelease(repo string, tag string) (string, error) {
	owner, repoName, err := parseGitHubRepoURL(repo)
	if err != nil {
		return "", fmt.Errorf("github_release_info: failed to parse repository URL: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "github_release_info", "repo", repo, "tag", tag)

	ctx, cancel := context.WithTimeout(context.Background(), githubDefaultTimeout)
	defer cancel()

	// GitHub API uses /releases/tags/{tag} endpoint
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repoName, tag)

	resp, err := g.fetchWithHeaders(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", fmt.Errorf("github_release_info: failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github_release_info: release %s/%s tag %s returned status %d: %s", owner, repoName, tag, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("github_release_info: failed to read response body: %w", err)
	}

	var releaseData Release
	if err := json.Unmarshal(body, &releaseData); err != nil {
		return "", fmt.Errorf("github_release_info: failed to parse response: %w", err)
	}

	return g.formatReleaseMarkdown(&releaseData), nil
}

// formatReleaseMarkdown formats release data as Markdown.
func (g *GitHubTools) formatReleaseMarkdown(release *Release) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Release: %s\n\n", release.Name)
	fmt.Fprintf(&sb, "**Tag:** %s\n\n", release.TagName)
	fmt.Fprintf(&sb, "**Published:** %s\n\n", release.PublishedAt)
	fmt.Fprintf(&sb, "**Author:** @%s\n\n", release.Author.Login)
	fmt.Fprintf(&sb, "**URL:** %s\n\n", release.HTMLURL)
	fmt.Fprintf(&sb, "**Prerelease:** %v\n\n", release.Prerelease)
	fmt.Fprintf(&sb, "**Draft:** %v\n\n", release.Draft)

	if release.Body != "" {
		fmt.Fprintf(&sb, "## Changelog\n\n%s\n\n", release.Body)
	}

	// Assets
	if len(release.Assets) > 0 {
		fmt.Fprintf(&sb, "## Assets (%d)\n\n", len(release.Assets))
		sb.WriteString("| Name | Size | Downloads | Uploaded |\n")
		sb.WriteString("|------|------|-----------|----------|\n")
		for _, asset := range release.Assets {
			sizeStr := fmt.Sprintf("%d bytes", asset.Size)
			if asset.Size > 1024*1024 {
				sizeStr = fmt.Sprintf("%.2f MB", float64(asset.Size)/(1024*1024))
			} else if asset.Size > 1024 {
				sizeStr = fmt.Sprintf("%.2f KB", float64(asset.Size)/1024)
			}
			fmt.Fprintf(&sb, "| [%s](%s) | %s | %d | %s |\n", asset.Name, asset.BrowserDownloadURL, sizeStr, asset.DownloadCount, asset.CreatedAt)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// DefinitionRelease returns the OpenAI tool schema for github_release_info.
func (g *GitHubTools) DefinitionRelease() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_release_info",
			Description: "Fetch a GitHub release by tag name and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"tag": map[string]any{
						"type":        "string",
						"description": "Release tag name (e.g., 'v1.6.0')",
					},
				},
				"required": []string{"repo", "tag"},
			},
		},
	}
}

// StaticDefinitionRelease returns the OpenAI tool schema for github_release_info without
// requiring an initialised GitHubTools instance.
func StaticDefinitionRelease() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "github_release_info",
			Description: "Fetch a GitHub release by tag name and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repo": map[string]any{
						"type":        "string",
						"description": "GitHub repository URL (e.g., 'https://github.com/owner/repo' or 'owner/repo')",
					},
					"tag": map[string]any{
						"type":        "string",
						"description": "Release tag name (e.g., 'v1.6.0')",
					},
				},
				"required": []string{"repo", "tag"},
			},
		},
	}
}
