package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/codec/html"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
	"github.com/wmentor/go-magnetar/internal/tools/github"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab"
	"github.com/wmentor/go-magnetar/internal/tools/jira"
	"github.com/wmentor/go-magnetar/internal/tools/rss"
)

const (
	defaultTimeout = time.Minute
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

// WebTools provides web fetching operations as LLM tools.
type WebTools struct {
	cfg   *config.Config
	state *plugin.State
}

// New creates a new WebTools instance.
func New(cfg *config.Config, root *os.Root, state *plugin.State) (*WebTools, error) {
	return &WebTools{
		cfg:   cfg,
		state: state,
	}, nil
}

// fetchURLWithMediaType retrieves content from a URL and returns (body, content_type, error).
func (w *WebTools) fetchURLWithMediaType(url string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   defaultTimeout,
		Transport: tr,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("web_fetch: failed to create request for %q: %w", url, err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("web_fetch: failed to fetch URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("web_fetch: URL %q returned status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", "", fmt.Errorf("web_fetch: decode URL %q error: %w", url, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", "", fmt.Errorf("web_fetch: failed to read response body: %w", err)
	}

	return string(body), contentType, nil
}

// WebFetch fetches a web page, preprocesses it (if HTML), and returns the cleaned content.
func (w *WebTools) WebFetch(url string) (string, error) {
	return w.webFetch(url, true)
}

func (w *WebTools) webFetch(url string, isFetch bool) (string, error) {
	if isFetch {
		printer.ToolCall(printer.IconSearch, "web_fetch", "url", url)
	}

	if w.cfg.String("confluence.base_url") != "" && w.cfg.Bool("confluence.enable") {
		if strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/spaces/") ||
			strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/x/") ||
			strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/p/") {
			pageID, err := confluence.ExtractPageIDURL(url)
			if err == nil && pageID != "" {
				isShortID := strings.Contains(url, "/x/") || strings.Contains(url, "/p/")
				confluencePlugin := confluence.New(w.cfg, w.state)
				return confluencePlugin.FetchPage(pageID, isShortID)
			}
		}
	}

	if w.cfg.String("jira.base_url") != "" && w.cfg.Bool("jira.enable") {
		if strings.HasPrefix(url, w.cfg.String("jira.base_url")) && (strings.Contains(url, "/browse/") || strings.Contains(url, "/issues/")) {
			issueKey, err := extractIssueKeyFromJIRAURL(url)
			if err == nil && issueKey != "" {
				return jira.New(w.cfg).FetchIssue(issueKey)
			}
		}
	}

	if w.cfg.String("gitlab.base_url") != "" && w.cfg.Bool("gitlab.enable") {
		if strings.HasPrefix(url, w.cfg.String("gitlab.base_url")) && strings.Contains(url, "/-/merge_requests/") {
			projectPath, issueID, err := extractProjectAndMergeRequestFromGitLabURL(url, w.cfg.String("gitlab.base_url"))
			if err == nil && projectPath != "" && issueID != "" {
				return w.fetchGitLabMergeRequest(projectPath, issueID)
			}
		}
	}

	if w.cfg.String("github.base_url") != "" && w.cfg.Bool("github.enable") { //nolint:nestif // TODO.
		if strings.Contains(url, "github.com/") {
			if strings.Contains(url, "/blob/") {
				owner, repo, branch, file, err := extractGitHubFileURL(url)
				if err == nil && owner != "" && repo != "" && file != "" {
					return github.New(w.cfg).FetchFile(owner+"/"+repo, branch, file)
				}
			}
			if strings.Contains(url, "/tree/") || strings.Contains(url, "/commits/") || strings.HasSuffix(url, "/") {
				owner, repo, branch, path, err := extractGitHubTreeURL(url)
				if err == nil && owner != "" && repo != "" {
					if branch == "" {
						branch = "main"
					}
					return github.New(w.cfg).FetchTree(owner+"/"+repo, branch, path)
				}
			}
			if strings.Contains(url, "/issues/") {
				owner, repo, issueNum, err := extractGitHubIssueURL(url)
				if err == nil && owner != "" && repo != "" && issueNum != "" {
					return github.New(w.cfg).FetchIssue(owner+"/"+repo, issueNum)
				}
			}
			if strings.Contains(url, "/milestone/") {
				owner, repo, milestoneNum, err := extractGitHubMilestoneURL(url)
				if err == nil && owner != "" && repo != "" && milestoneNum != "" {
					return github.New(w.cfg).FetchMilestone(owner+"/"+repo, milestoneNum)
				}
			}
			if strings.Contains(url, "/advisories/") || strings.Contains(url, "/security/advisories") {
				if strings.Contains(url, "/security/advisories") && !strings.Contains(url, "/advisories/GHSA-") {
					owner, repo, err := extractGitHubRepoURL(url)
					if err == nil && owner != "" && repo != "" {
						return github.New(w.cfg).FetchRepoAdvisories(owner + "/" + repo)
					}
				} else {
					advisoryID, err := extractGitHubAdvisoryURL(url)
					if err == nil && advisoryID != "" {
						return github.New(w.cfg).FetchAdvisory(advisoryID)
					}
				}
			}
			owner, repo, err := extractGitHubRepoURL(url)
			if err == nil && owner != "" && repo != "" {
				return github.New(w.cfg).FetchRepository(owner + "/" + repo)
			}
		}
	}

	content, contentType, err := w.fetchURLWithMediaType(url)
	if err != nil {
		if isFetch {
			printer.ToolCall(printer.IconError, "web_fetch: failed to fetch URL", "url", url, "err", err)
			return "", fmt.Errorf("web_fetch: failed to fetch URL %q", url)
		}
		return "", errors.New("web_search: failed")
	}

	contentType = strings.ToLower(contentType)

	if strings.Contains(contentType, "text/html") {
		codec := &html.Codec{}
		return codec.ProcessContent(content, url)
	}

	for _, rssType := range []string{"application/rss+xml", "text/xml", "application/xml"} {
		if strings.Contains(contentType, rssType) {
			if r, e := rss.Decode(strings.NewReader(content)); e == nil {
				if len(r.Channel.Items) == 0 {
					break
				}
				return r.String(), nil
			} else {
				break
			}
		}
	}

	return content, nil
}

// WebSearch executes web search and returns results in Markdown format.
func (w *WebTools) WebSearch(query string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_search", "query", query)
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", encodedQuery)
	return w.webFetch(searchURL, false)
}

// extractIssueKeyFromJIRAURL extracts the issue key (e.g., GOARCH-60) from a JIRA URL.
func extractIssueKeyFromJIRAURL(url string) (string, error) {
	// Handle URLs with /browse/
	if _, after, ok := strings.Cut(url, "/browse/"); ok {
		idPart := after
		if idx2 := strings.Index(idPart, "/"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "?"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "#"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idPart == "" {
			return "", fmt.Errorf("issue key is empty")
		}
		return idPart, nil
	}

	// Handle URLs with /issues/
	if _, after, ok := strings.Cut(url, "/issues/"); ok {
		idPart := after
		if idx2 := strings.Index(idPart, "/"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "?"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "#"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idPart == "" {
			return "", fmt.Errorf("issue key is empty")
		}
		return idPart, nil
	}

	return "", fmt.Errorf("not a JIRA issue URL")
}

// Definition returns the OpenAI tool schema for web_fetch.
func (w *WebTools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "web_fetch",
			Description: "Fetch a web page and return clean Markdown content",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL of the web page to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// DefinitionSearch returns the OpenAI tool schema for web_search.
func (w *WebTools) DefinitionSearch() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "web_search",
			Description: "Execute web search and return results in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query string",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (w *WebTools) Dispatch(name string, args string) string {
	switch name {
	case "web_fetch":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("web_fetch: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, err := w.WebFetch(params.URL)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "web_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("web_search: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, err := w.WebSearch(params.Query)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	default:
		printer.ToolCall(printer.IconError, "unknown tool", "name", name)
		return "error: unknown tool " + name
	}
}

// extractProjectAndMergeRequestFromGitLabURL extracts the project path and merge request ID from a GitLab URL.
func extractProjectAndMergeRequestFromGitLabURL(url string, baseURL string) (string, string, error) {
	// Handle URLs with /-/merge_requests/
	if before, after, ok := strings.Cut(url, "/-/merge_requests/"); ok {
		pathPart := before
		// Extract project path (remove base URL)

		pathPart = strings.TrimPrefix(pathPart, baseURL)

		idPart := after
		if idx2 := strings.Index(idPart, "/"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "?"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "#"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idPart == "" {
			return "", "", fmt.Errorf("merge request ID is empty")
		}

		// Extract project path from the URL
		projectPath := strings.TrimPrefix(pathPart, "/")

		return projectPath, idPart, nil
	}

	// Handle URLs with /-/
	if idx := strings.Index(url, "/-/"); idx != -1 {
		pathPart := url[:idx+2]
		idPart := url[idx+2:]
		if idx2 := strings.Index(idPart, "/"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "?"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idx2 := strings.Index(idPart, "#"); idx2 != -1 {
			idPart = idPart[:idx2]
		}
		if idPart == "" {
			return "", "", fmt.Errorf("merge request ID is empty")
		}

		// Extract project path
		projectPath := strings.TrimPrefix(pathPart, "/")
		if idx2 := strings.Index(projectPath, "/"); idx2 != -1 {
			projectPath = projectPath[idx2+1:]
		}

		return projectPath, idPart, nil
	}

	return "", "", fmt.Errorf("not a GitLab merge request URL")
}

// fetchGitLabMergeRequest fetches a GitLab merge request and returns its content.
func (w *WebTools) fetchGitLabMergeRequest(projectPath string, mrID string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab merge request", "project_path", projectPath, "mr_id", mrID)
	return gitlab.New(w.cfg).FetchMergeRequest(projectPath, mrID)
}

// extractGitHubRepoURL extracts owner and repo from GitHub repository URL.
func extractGitHubRepoURL(url string) (string, string, error) {
	// Remove protocol prefix if present
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimPrefix(url, "github.com/")

	// Split into owner and repo
	parts := strings.Split(strings.Trim(url, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repository URL format")
	}

	owner := parts[0]
	repo := parts[1]

	// Remove .git suffix if present
	repo = strings.TrimSuffix(repo, ".git")

	return owner, repo, nil
}

// githubURLPath returns the escaped path of a github.com URL without the query string,
// fragment and trailing slash. It returns an error for any other host.
func githubURLPath(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com":
	default:
		return "", fmt.Errorf("not a GitHub URL")
	}

	return strings.TrimRight(u.EscapedPath(), "/"), nil
}

// extractGitHubTreeURL extracts owner, repo, branch, and path from GitHub tree URL.
func extractGitHubTreeURL(rawURL string) (string, string, string, string, error) {
	// Pattern: https://github.com/owner/repo/tree/branch/path or https://github.com/owner/repo/commits/branch
	// Match: /{owner}/{repo}/(tree|commits)/{branch}[/path]
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", "", err
	}

	re := regexp.MustCompile(`^/([^/]+)/([^/]+)/(tree|commits)/([^/]+)(?:/(.+))?$`)
	matches := re.FindStringSubmatch(path)
	if matches != nil {
		// matches[5] (the path) is empty when the URL has no path after the branch.
		return matches[1], matches[2], matches[4], matches[5], nil
	}

	return "", "", "", "", fmt.Errorf("not a GitHub tree/commits URL")
}

// extractGitHubFileURL extracts owner, repo, branch, and file path from GitHub blob URL.
func extractGitHubFileURL(rawURL string) (string, string, string, string, error) {
	// Pattern: https://github.com/owner/repo/blob/branch/path/to/file
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", "", err
	}

	re := regexp.MustCompile(`^/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
	matches := re.FindStringSubmatch(path)
	if matches != nil {
		owner := matches[1]
		repo := matches[2]
		branch := matches[3]
		file := matches[4]
		return owner, repo, branch, file, nil
	}

	return "", "", "", "", fmt.Errorf("not a GitHub blob URL")
}

func extractGitHubIssueURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	re := regexp.MustCompile(`^/([^/]+)/([^/]+)/issues/(\d+)$`)
	matches := re.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", fmt.Errorf("not a GitHub issue URL")
}

func extractGitHubMilestoneURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	re := regexp.MustCompile(`^/([^/]+)/([^/]+)/milestone/(\d+)$`)
	matches := re.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", fmt.Errorf("not a GitHub milestone URL")
}

func extractGitHubAdvisoryURL(url string) (string, error) {
	if strings.Contains(url, "/advisories/") {
		parts := strings.Split(url, "/advisories/")
		if len(parts) > 1 {
			idPart := parts[1]
			if idx := strings.Index(idPart, "/"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idx := strings.Index(idPart, "?"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idx := strings.Index(idPart, "#"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idPart == "" {
				return "", fmt.Errorf("advisory ID is empty")
			}
			return idPart, nil
		}
	}

	return "", fmt.Errorf("not a GitHub advisory URL")
}

// StaticDefinition returns the OpenAI tool schema for web_fetch without
// requiring an initialised WebTools instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "web_fetch",
			Description: "Fetch a web page and return clean Markdown content. Supports " +
				"Confluence pages, JIRA issues, GitLab merge requests, and GitHub repositories, issues, and milestones.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL of the web page, Confluence page, JIRA issue, GitLab merge request, or GitHub repository to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// StaticDefinitionSearch returns the OpenAI tool schema for web_search without
// requiring an initialised WebTools instance. Used by the plugin for lazy init.
func StaticDefinitionSearch() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "web_search",
			Description: "Execute web search and return results in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query string",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}
