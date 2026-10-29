package webfetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/codec/html"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
	"github.com/wmentor/go-magnetar/internal/tools/github"
	gitlabfile "github.com/wmentor/go-magnetar/internal/tools/gitlab/file"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/mr"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/repo"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/tree"
	"github.com/wmentor/go-magnetar/internal/tools/jira"
	"github.com/wmentor/go-magnetar/internal/tools/rss"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

const (
	webFetchDefaultTimeout = time.Minute
	userAgent              = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

// Tool provides web_fetch operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	state         *plugin.State
}

// NewTool creates a new Tool instance.
func NewTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		state:         state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "web_fetch"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Fetch a web page and return clean Markdown content. " +
		"Supports Confluence pages, JIRA issues, GitLab repositories/files/tree/merge requests, " +
		"and GitHub repositories, issues, milestones, pull requests, releases, and advisories."
}

// Params represents parameters for the web_fetch tool.
type Params struct {
	URL string `json:"url" jsonschema_description:"URL of the web page to fetch."`
}

// WebFetch fetches a web page, preprocesses it (if HTML), and returns the cleaned content.
func (t *Tool) WebFetch(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "web_fetch", "url", params.URL)

	result, err := t.WebFetchEngine(ctx, params.URL, true)
	if err != nil {
		t.printer.Error("web_fetch: failed to fetch URL", "url", params.URL, "err", err)
		return nil, fmt.Errorf("web_fetch: failed to fetch URL %q: %w", params.URL, err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: result,
	}, nil
}

func (t *Tool) WebFetchEngine(ctx context.Context, urlStr string, isFetch bool) (string, error) {
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}

	if t.cfg.String("confluence.base_url") != "" && t.cfg.Bool("confluence.enable") {
		if strings.HasPrefix(urlStr, t.cfg.String("confluence.base_url")+"/spaces/") ||
			strings.HasPrefix(urlStr, t.cfg.String("confluence.base_url")+"/x/") ||
			strings.HasPrefix(urlStr, t.cfg.String("confluence.base_url")+"/p/") {
			pageID, err := confluence.ExtractPageIDURL(urlStr)
			if err == nil && pageID != "" {
				isShortID := strings.Contains(urlStr, "/x/") || strings.Contains(urlStr, "/p/")
				// Use FetchTool directly
				fetchTool := confluence.NewFetchTool(t.cfg, t.printer, t.clientFactory)
				result, err := fetchTool.FetchPage(context.Background(), &confluence.FetchPageParams{
					PageID:  pageID,
					IsShort: isShortID,
				})
				if err != nil {
					t.printer.Error("confluence_page_fetch: failed to fetch page", "url", urlStr, "err", err)
					return "", fmt.Errorf("confluence_page_fetch: failed to fetch page %q: %w", urlStr, err)
				}
				if resp, ok := result.(*tools.Response); ok {
					return resp.Result, nil
				}
				return "", fmt.Errorf("confluence_page_fetch: unexpected result type: %T", result)
			}
		}
	}

	if t.cfg.String("jira.base_url") != "" && t.cfg.Bool("jira.enable") {
		if strings.HasPrefix(urlStr, t.cfg.String("jira.base_url")) && (strings.Contains(urlStr, "/browse/") || strings.Contains(urlStr, "/issues/")) {
			issueKey, err := web.ExtractIssueKeyFromJIRAURL(urlStr)
			if err == nil && issueKey != "" {
				return jira.New(t.cfg, t.printer, t.clientFactory).FetchIssue(issueKey)
			}
		}
	}

	//nolint:nestif // TODO.
	if t.cfg.String("gitlab.base_url") != "" && t.cfg.Bool("gitlab.enable") {
		if strings.HasPrefix(urlStr, t.cfg.String("gitlab.base_url")) {
			// Check for merge request
			if strings.Contains(urlStr, "/-/merge_requests/") || strings.Contains(urlStr, "/-/mr/") {
				projectPath, issueID, err := web.ExtractProjectAndMergeRequestFromGitLabURL(urlStr)
				if err == nil && projectPath != "" && issueID != "" {
					return t.fetchGitLabMergeRequest(projectPath, issueID)
				}
			}

			// Check for file (blob)
			if strings.Contains(urlStr, "/-/blob/") {
				projectPath, branch, file, err := web.ExtractGitLabFileURL(urlStr)
				if err == nil && projectPath != "" && file != "" {
					if branch == "" {
						branch = "main"
					}
					return t.fetchGitLabFile(projectPath, branch, file)
				}
			}

			// Check for tree (directory)
			if strings.Contains(urlStr, "/-/tree/") || strings.Contains(urlStr, "/-/commits/") {
				projectPath, branch, path, err := web.ExtractGitLabTreeURL(urlStr)
				if err == nil && projectPath != "" {
					if branch == "" {
						branch = "main"
					}
					return t.fetchGitLabTree(projectPath, branch, path)
				}
			}

			// Check for repository (root level)
			if !strings.Contains(urlStr, "/-/") {
				projectPath, err := web.ExtractGitLabRepoURL(urlStr)
				if err == nil && projectPath != "" {
					return t.fetchGitLabRepository(projectPath)
				}
			}
		}
	}

	if t.cfg.String("github.base_url") != "" && t.cfg.Bool("github.enable") { //nolint:nestif // TODO.
		if strings.Contains(urlStr, "github.com/") {
			if strings.Contains(urlStr, "/blob/") {
				owner, repo, branch, file, err := web.ExtractGitHubFileURL(urlStr)
				if err == nil && owner != "" && repo != "" && file != "" {
					return github.New(t.cfg).FetchFile(owner+"/"+repo, branch, file)
				}
			}
			// Check for pull request before tree/commits to avoid conflicts with /pull/123/commits
			if web.IsGitHubPullRequestURL(urlStr) {
				owner, repo, prNum, err := web.ExtractGitHubPullRequestURL(urlStr)
				if err == nil && owner != "" && repo != "" && prNum != "" {
					return github.New(t.cfg).FetchPullRequest(owner+"/"+repo, prNum)
				}
			}
			if strings.Contains(urlStr, "/tree/") || strings.Contains(urlStr, "/commits/") || strings.HasSuffix(urlStr, "/") {
				owner, repo, branch, path, err := web.ExtractGitHubTreeURL(urlStr)
				if err == nil && owner != "" && repo != "" {
					if branch == "" {
						branch = "main"
					}
					return github.New(t.cfg).FetchTree(owner+"/"+repo, branch, path)
				}
			}
			if strings.Contains(urlStr, "/issues/") {
				owner, repo, issueNum, err := web.ExtractGitHubIssueURL(urlStr)
				if err == nil && owner != "" && repo != "" && issueNum != "" {
					return github.New(t.cfg).FetchIssue(owner+"/"+repo, issueNum)
				}
			}
			if strings.Contains(urlStr, "/milestone/") {
				owner, repo, milestoneNum, err := web.ExtractGitHubMilestoneURL(urlStr)
				if err == nil && owner != "" && repo != "" && milestoneNum != "" {
					return github.New(t.cfg).FetchMilestone(owner+"/"+repo, milestoneNum)
				}
			}
			if strings.Contains(urlStr, "/advisories/") || strings.Contains(urlStr, "/security/advisories") {
				if strings.Contains(urlStr, "/security/advisories") && !strings.Contains(urlStr, "/advisories/GHSA-") {
					owner, repo, err := web.ExtractGitHubRepoURL(urlStr)
					if err == nil && owner != "" && repo != "" {
						return github.New(t.cfg).FetchRepoAdvisories(owner + "/" + repo)
					}
				} else {
					advisoryID, err := web.ExtractGitHubAdvisoryURL(urlStr)
					if err == nil && advisoryID != "" {
						return github.New(t.cfg).FetchAdvisory(advisoryID)
					}
				}
			}
			if strings.Contains(urlStr, "/releases/tag/") {
				owner, repo, tag, err := web.ExtractGitHubReleaseURL(urlStr)
				if err == nil && owner != "" && repo != "" && tag != "" {
					return github.New(t.cfg).FetchRelease(owner+"/"+repo, tag)
				}
			}
			owner, repo, err := web.ExtractGitHubRepoURL(urlStr)
			if err == nil && owner != "" && repo != "" {
				return github.New(t.cfg).FetchRepository(owner + "/" + repo)
			}
		}
	}

	body, contentType, err := t.fetchURLWithContentType(ctx, urlStr)
	if err != nil {
		if isFetch {
			t.printer.Error("web_fetch: failed to fetch URL", "url", urlStr, "err", err)
			return "", fmt.Errorf("web_fetch: failed to fetch URL %q", urlStr)
		}
		return "", errors.New("web_search: failed")
	}

	mimeType, err := web.GetMimeType(contentType)
	if err != nil {
		if isFetch {
			t.printer.Error("web_fetch: failed to parse MIME type", "url", urlStr, "err", err)
			return "", fmt.Errorf("web_fetch: failed to parse MIME type for URL %q", urlStr)
		}
		return "", errors.New("web_search: failed")
	}

	if mimeCodec, err := codec.GetCodecByMIME(mimeType); err == nil {
		result, e := t.mimeCodecWork(mimeCodec, body)
		if e != nil {
			return "", fmt.Errorf("decode data error: %w", e)
		}
		return result, nil
	}

	if strings.Contains(mimeType, "text/html") {
		codec := &html.Codec{}
		return codec.ProcessContent(string(body), urlStr)
	}

	for _, rssType := range []string{"application/rss+xml", "text/xml", "application/xml"} {
		if strings.Contains(mimeType, rssType) {
			if r, e := rss.Decode(bytes.NewReader(body)); e == nil {
				if len(r.Channel.Items) == 0 {
					break
				}
				return r.String(), nil
			} else {
				break
			}
		}
	}

	return string(body), nil
}

// fetchURLWithContentType retrieves content from a URL and returns (body, content_type, error).
func (t *Tool) fetchURLWithContentType(ctx context.Context, u string) ([]byte, string, error) {
	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("web_fetch: failed to create request for %q: %w", u, err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("web_fetch: failed to fetch URL %q: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("web_fetch: URL %q returned status %d", u, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	var dataReader io.Reader = resp.Body

	mimeType, err := web.GetMimeType(contentType)
	if err != nil || !codec.IsBinaryFormat(mimeType) {
		utf8, err1 := charset.NewReader(resp.Body, contentType)
		if err1 != nil {
			return nil, "", fmt.Errorf("web_fetch: decode URL %q error: %w", u, err1)
		}
		dataReader = utf8
	}

	body, err := io.ReadAll(dataReader)
	if err != nil {
		return nil, "", fmt.Errorf("web_fetch: failed to read response body: %w", err)
	}

	return body, contentType, nil
}

func (t *Tool) mimeCodecWork(myCodec codec.Codec, content []byte) (string, error) {
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

// fetchGitLabMergeRequest fetches a GitLab merge request and returns its content.
func (t *Tool) fetchGitLabMergeRequest(projectPath string, mrID string) (string, error) {
	t.printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab merge request", "project_path", projectPath, "mr_id", mrID)
	tool := mr.New(t.cfg, t.printer, t.clientFactory)
	result, err := tool.Exec(context.Background(), &mr.GitLabMRParams{
		ProjectPath: projectPath,
		MRID:        mrID,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabRepository fetches a GitLab repository and returns its content.
func (t *Tool) fetchGitLabRepository(projectPath string) (string, error) {
	t.printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab repository", "project_path", projectPath)
	tool := repo.New(t.cfg, t.printer, t.clientFactory)
	result, err := tool.Exec(context.Background(), &repo.GitLabRepoParams{
		ProjectPath: projectPath,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabFile fetches a GitLab file and returns its content.
func (t *Tool) fetchGitLabFile(projectPath string, branch string, file string) (string, error) {
	t.printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab file", "project_path", projectPath, "branch", branch, "file", file)
	tool := gitlabfile.New(t.cfg, t.printer, t.clientFactory)
	result, err := tool.Exec(context.Background(), &gitlabfile.GitLabFileParams{
		ProjectPath: projectPath,
		Branch:      branch,
		File:        file,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// fetchGitLabTree fetches a GitLab directory tree and returns its content.
func (t *Tool) fetchGitLabTree(projectPath string, branch string, path string) (string, error) {
	t.printer.ToolCall(printer.IconSearch, "web_fetch: fetch GitLab tree", "project_path", projectPath, "branch", branch, "path", path)
	tool := tree.New(t.cfg, t.printer, t.clientFactory)
	result, err := tool.Exec(context.Background(), &tree.GitLabTreeParams{
		ProjectPath: projectPath,
		Branch:      branch,
		Path:        path,
	})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// StaticDefinition returns the OpenAI tool schema for web_fetch without
// requiring an initialised Tool instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "web_fetch",
			Description: "Fetch a web page and return clean Markdown content. " +
				"Supports Confluence pages, JIRA issues, GitLab repositories/files/tree/merge requests, " +
				"and GitHub repositories, issues, milestones, pull requests, releases, and advisories.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type": "string",
						"description": "URL of the web page, Confluence page, JIRA issue, " +
							"GitLab repository/file/tree/merge request, or GitHub repository to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// WebFetchString executes the web_fetch tool with the given JSON args and returns the result as a string.
func (t *Tool) WebFetchString(args string) (string, error) {
	var params Params
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return fmt.Sprintf("error: failed to parse arguments: %v", err), nil
	}

	ctx := context.Background()
	result, err := t.WebFetch(ctx, &params)
	if err != nil {
		return fmt.Sprintf("error: %v", err), nil
	}

	// Extract content from response
	if resp, ok := result.(tools.Response); ok {
		return resp.Result, nil
	}
	if resp, ok := result.(map[string]any); ok {
		if r, ok := resp["result"]; ok {
			return fmt.Sprintf("%v", r), nil
		}
	}
	return fmt.Sprintf("%v", result), nil
}
