package web

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/codec"
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

// fetchURLWithContentType retrieves content from a URL and returns (body, content_type, error).
func (w *WebTools) fetchURLWithContentType(url string) ([]byte, string, error) {
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
		return nil, "", fmt.Errorf("web_fetch: failed to create request for %q: %w", url, err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("web_fetch: failed to fetch URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("web_fetch: URL %q returned status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	var dataReader io.Reader = resp.Body

	mimeType, err := GetMimeType(contentType)
	if err != nil || !codec.IsBinaryFormat(mimeType) {
		utf8, err1 := charset.NewReader(resp.Body, contentType)
		if err1 != nil {
			return nil, "", fmt.Errorf("web_fetch: decode URL %q error: %w", url, err1)
		}
		dataReader = utf8
	}

	body, err := io.ReadAll(dataReader)
	if err != nil {
		return nil, "", fmt.Errorf("web_fetch: failed to read response body: %w", err)
	}

	return body, contentType, nil
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

	//nolint:nestif // TODO.
	if w.cfg.String("gitlab.base_url") != "" && w.cfg.Bool("gitlab.enable") {
		if strings.HasPrefix(url, w.cfg.String("gitlab.base_url")) {
			// Check for merge request
			if strings.Contains(url, "/-/merge_requests/") || strings.Contains(url, "/-/mr/") {
				projectPath, issueID, err := extractProjectAndMergeRequestFromGitLabURL(url)
				if err == nil && projectPath != "" && issueID != "" {
					return w.fetchGitLabMergeRequest(projectPath, issueID)
				}
			}

			// Check for file (blob)
			if strings.Contains(url, "/-/blob/") {
				projectPath, branch, file, err := extractGitLabFileURL(url)
				if err == nil && projectPath != "" && file != "" {
					if branch == "" {
						branch = "main"
					}
					return w.fetchGitLabFile(projectPath, branch, file)
				}
			}

			// Check for tree (directory)
			if strings.Contains(url, "/-/tree/") || strings.Contains(url, "/-/commits/") {
				projectPath, branch, path, err := extractGitLabTreeURL(url)
				if err == nil && projectPath != "" {
					if branch == "" {
						branch = "main"
					}
					return w.fetchGitLabTree(projectPath, branch, path)
				}
			}

			// Check for repository (root level)
			if !strings.Contains(url, "/-/") {
				projectPath, err := extractGitLabRepoURL(url)
				if err == nil && projectPath != "" {
					return w.fetchGitLabRepository(projectPath)
				}
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
			// Check for pull request before tree/commits to avoid conflicts with /pull/123/commits
			if isGitHubPullRequestURL(url) {
				owner, repo, prNum, err := extractGitHubPullRequestURL(url)
				if err == nil && owner != "" && repo != "" && prNum != "" {
					return github.New(w.cfg).FetchPullRequest(owner+"/"+repo, prNum)
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
			if strings.Contains(url, "/releases/tag/") {
				owner, repo, tag, err := extractGitHubReleaseURL(url)
				if err == nil && owner != "" && repo != "" && tag != "" {
					return github.New(w.cfg).FetchRelease(owner+"/"+repo, tag)
				}
			}
			owner, repo, err := extractGitHubRepoURL(url)
			if err == nil && owner != "" && repo != "" {
				return github.New(w.cfg).FetchRepository(owner + "/" + repo)
			}
		}
	}

	content, contentType, err := w.fetchURLWithContentType(url)
	if err != nil {
		if isFetch {
			printer.ToolCall(printer.IconError, "web_fetch: failed to fetch URL", "url", url, "err", err)
			return "", fmt.Errorf("web_fetch: failed to fetch URL %q", url)
		}
		return "", errors.New("web_search: failed")
	}

	mimeType, err := GetMimeType(contentType)
	if err != nil {
		if isFetch {
			printer.ToolCall(printer.IconError, "web_fetch: failed to parse MIME type", "url", url, "err", err)
			return "", fmt.Errorf("web_fetch: failed to parse MIME type for URL %q", url)
		}
		return "", errors.New("web_search: failed")
	}

	if mimeCodec, err := codec.GetCodecByMIME(mimeType); err == nil {
		if result, e := mimeCodecWork(mimeCodec, content); e == nil {
			return result, nil
		} else {
			fmt.Println(result)
			return "", fmt.Errorf("decode data error: %w", e)
		}
	}

	if strings.Contains(mimeType, "text/html") {
		codec := &html.Codec{}
		return codec.ProcessContent(string(content), url)
	}

	for _, rssType := range []string{"application/rss+xml", "text/xml", "application/xml"} {
		if strings.Contains(mimeType, rssType) {
			if r, e := rss.Decode(bytes.NewReader(content)); e == nil {
				if len(r.Channel.Items) == 0 {
					break
				}
				return r.String(), nil
			} else {
				break
			}
		}
	}

	return string(content), nil
}

func mimeCodecWork(myCodec codec.Codec, content []byte) (string, error) {
	if myCodec == nil {
		return "", errors.New("codec is nil")
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "magnetar-")
	if err != nil {
		return "", fmt.Errorf("create temp file error: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to the temporary file
	if _, err := tmpFile.Write(content); err != nil {
		return "", fmt.Errorf("write temp file error: %w", err)
	}

	// Close the file before reading
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file error: %w", err)
	}

	// Read the file using the codec
	return myCodec.ReadFile(tmpFile.Name())
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
	for _, path := range []string{"/browse/", "/issues/"} {
		_, idPart, ok := strings.Cut(url, path)
		if !ok {
			continue
		}
		if idx := strings.IndexAny(idPart, "/?#"); idx != -1 {
			idPart = idPart[:idx]
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
func extractProjectAndMergeRequestFromGitLabURL(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Handle URLs with /-/merge_requests/ and the short /-/mr/ form
	for _, marker := range []string{"/-/merge_requests/", "/-/mr/"} {
		if !strings.Contains(u.Path, marker) {
			continue
		}

		// Split path at marker
		parts := strings.SplitN(u.Path, marker, 2)
		if len(parts) < 2 {
			continue
		}

		// Extract project path (before marker)
		projectPath := strings.TrimPrefix(parts[0], "/")

		// Extract MR ID (after marker)
		idPart := parts[1]
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

		return projectPath, idPart, nil
	}

	return "", "", fmt.Errorf("not a GitLab merge request URL")
}

// fetchGitLabMergeRequest fetches a GitLab merge request and returns its content.
func (w *WebTools) fetchGitLabMergeRequest(projectPath string, mrID string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab merge request", "project_path", projectPath, "mr_id", mrID)
	return gitlab.New(w.cfg).FetchMergeRequest(projectPath, mrID)
}

// fetchGitLabRepository fetches a GitLab repository and returns its content.
func (w *WebTools) fetchGitLabRepository(projectPath string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab repository", "project_path", projectPath)
	return gitlab.New(w.cfg).FetchRepository(projectPath)
}

// fetchGitLabFile fetches a GitLab file and returns its content.
func (w *WebTools) fetchGitLabFile(projectPath string, branch string, file string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab file", "project_path", projectPath, "branch", branch, "file", file)
	return gitlab.New(w.cfg).FetchFile(projectPath, branch, file)
}

// fetchGitLabTree fetches a GitLab directory tree and returns its content.
func (w *WebTools) fetchGitLabTree(projectPath string, branch string, path string) (string, error) {
	printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab tree", "project_path", projectPath, "branch", branch, "path", path)
	return gitlab.New(w.cfg).FetchTree(projectPath, branch, path)
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

// StaticDefinition returns the OpenAI tool schema for web_fetch without
// requiring an initialised WebTools instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "web_fetch",
			Description: "Fetch a web page and return clean Markdown content. Supports " +
				"Confluence pages, JIRA issues, GitLab repositories/files/tree/merge requests, and GitHub repositories, issues, and milestones.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL of the web page, Confluence page, JIRA issue, GitLab repository/file/tree/merge request, or GitHub repository to fetch",
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
