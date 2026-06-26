package web

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	sanitizer "github.com/wmentor/go-magnetar/internal/agent/markdown"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab"
)

const (
	defaultTimeout = time.Minute
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

// WebTools provides web fetching operations as LLM tools.
type WebTools struct {
	cfg          *config.Config
	preprocessor *sanitizer.Preprocessor
}

// New creates a new WebTools instance.
func New(cfg *config.Config, root *os.Root) (*WebTools, error) {
	var preprocessor *sanitizer.Preprocessor
	if cfg.String("webfetch.base_url") != "" {
		p, err := sanitizer.New(cfg, root)
		if err != nil {
			return nil, fmt.Errorf("web: failed to create preprocessor: %w", err)
		}
		preprocessor = p
	}

	return &WebTools{
		cfg:          cfg,
		preprocessor: preprocessor,
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
		return "", "", fmt.Errorf("web: failed to create request for %q: %w", url, err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("web: failed to fetch URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("web: URL %q returned status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", "", fmt.Errorf("web: decode URL %q error: %w", url, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", "", fmt.Errorf("web: failed to read response body: %w", err)
	}

	return string(body), contentType, nil
}

func (w *WebTools) preprocessMarkdown(markdownStr string) (string, error) {
	if w.preprocessor == nil {
		return markdownStr, nil
	}
	return w.preprocessor.ProcessMDString(markdownStr)
}

// WebFetch fetches a web page, preprocesses it (if HTML), and returns the cleaned content.
func (w *WebTools) WebFetch(url string) (string, error) {
	slog.Debug("webfetch: fetching URL", "url", url)

	if w.cfg.String("confluence.base_url") != "" {
		if strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/spaces/") || strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/x/") || strings.HasPrefix(url, w.cfg.String("confluence.base_url")+"/p/") {
			pageID, err := extractPageIDFromConfluenceURL(url)
			if err == nil && pageID != "" {
				isShortID := strings.Contains(url, "/x/") || strings.Contains(url, "/p/")
				return w.fetchConfluencePage(pageID, isShortID)
			}
		}
	}

	if w.cfg.String("jira.base_url") != "" {
		if strings.HasPrefix(url, w.cfg.String("jira.base_url")) && (strings.Contains(url, "/browse/") || strings.Contains(url, "/issues/")) {
			issueKey, err := extractIssueKeyFromJIRAURL(url)
			if err == nil && issueKey != "" {
				return w.fetchJIRAIssue(issueKey)
			}
		}
	}

	if w.cfg.String("gitlab.base_url") != "" {
		if strings.HasPrefix(url, w.cfg.String("gitlab.base_url")) && strings.Contains(url, "/-/merge_requests/") {
			projectPath, issueID, err := extractProjectAndMergeRequestFromGitLabURL(url, w.cfg.String("gitlab.base_url"))
			if err == nil && projectPath != "" && issueID != "" {
				return w.fetchGitLabMergeRequest(projectPath, issueID)
			}
		}
	}

	content, contentType, err := w.fetchURLWithMediaType(url)
	if err != nil {
		slog.Error("webfetch: failed to fetch URL", "url", url, "err", err)
		return "", fmt.Errorf("webfetch: failed to fetch URL %q", url)
	}

	if contentType != "" && strings.Contains(strings.ToLower(contentType), "text/html") {
		slog.Debug("webfetch: HTML detected, preprocessing", "url", url, "content_type", contentType)

		content, err := CleanHTML(content)
		if err != nil {
			return "", fmt.Errorf("webfetch: URL %q clean html error: %w", url, err)
		}

		content, err = ProcessReadability(content, url)
		if err != nil {
			return "", fmt.Errorf("webfetch: URL %q error: %w", url, err)
		}

		content, err = HTMLToMarkdown(content)
		if err != nil {
			return "", fmt.Errorf("webfetch: URL %q html to markdown error: %w", url, err)
		}

		content, err = w.preprocessMarkdown(content)
		if err != nil {
			slog.Error("webfetch: preprocessing failed", "url", url, "err", err)
			return "", fmt.Errorf("webfetch: preprocessing failed for URL %q", url)
		}

		slog.Debug("webfetch: done", "url", url)
		return content, nil
	}

	slog.Debug("webfetch: done (non-HTML content)", "url", url, "content_type", contentType)
	return content, nil
}

// extractIssueKeyFromJIRAURL extracts the issue key (e.g., GOARCH-60) from a JIRA URL.
func extractIssueKeyFromJIRAURL(url string) (string, error) {
	// Handle URLs with /browse/
	if idx := strings.Index(url, "/browse/"); idx != -1 {
		idPart := url[idx+8:]
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
	if idx := strings.Index(url, "/issues/"); idx != -1 {
		idPart := url[idx+8:]
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

// extractPageIDFromConfluenceURL parses a Confluence URL and returns the page ID.
func extractPageIDFromConfluenceURL(url string) (string, error) {
	// Handle short link: .../x/{page_id}
	if idx := strings.Index(url, "/x/"); idx != -1 {
		idPart := url[idx+3:]
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
			return "", fmt.Errorf("page ID is empty")
		}
		return idPart, nil
	}

	// Handle share link: .../p/{page_id}
	if idx := strings.Index(url, "/p/"); idx != -1 {
		idPart := url[idx+3:]
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
			return "", fmt.Errorf("page ID is empty")
		}
		return idPart, nil
	}

	// Handle standard URL: .../spaces/{space}/pages/{page_id}[/{suffix}]
	parts := strings.Split(url, "/pages/")
	if len(parts) != 2 {
		return "", fmt.Errorf("not a Confluence page URL")
	}

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
		return "", fmt.Errorf("page ID is empty")
	}

	return idPart, nil
}

// decodeShortPageID decodes a Confluence short page ID (e.g., "A4HhC") to numeric ID using Base64.
// Confluence pads the code to 11 chars with 'A' and adds '=' for padding, then decodes to 32-bit LE integer.
func decodeShortPageID(shortCode string) (int64, error) {
	// 1. Pad to 11 characters with 'A' and add '='
	paddedCode := shortCode
	if len(paddedCode) < 11 {
		paddedCode = paddedCode + strings.Repeat("A", 11-len(paddedCode))
	}
	paddedCode += "="

	// 2. Decode Base64 to bytes
	decoded, err := base64.StdEncoding.DecodeString(paddedCode)
	if err != nil {
		return 0, fmt.Errorf("web: failed to decode short code %q: %w", shortCode, err)
	}

	// 3. Unpack 32-bit Little-Endian integer using binary package
	if len(decoded) < 4 {
		return 0, fmt.Errorf("web: decoded data too short for page ID")
	}
	pageID := binary.LittleEndian.Uint32(decoded[:4])

	return int64(pageID), nil
}

// resolveShortPageID resolves a short Confluence page code (e.g., AgA5) to numeric ID.
func (w *WebTools) resolveShortPageID(shortCode string) (string, error) {
	slog.Debug("webfetch: resolving short Confluence page ID", "short_code", shortCode)

	pageID, err := decodeShortPageID(shortCode)
	if err != nil {
		return "", err
	}

	slog.Debug("webfetch: decoded short page ID", "short", shortCode, "numeric", pageID)
	return fmt.Sprintf("%d", pageID), nil
}

// fetchConfluencePage fetches a Confluence page by ID and returns its content in Markdown.
func (w *WebTools) fetchConfluencePage(pageID string, isShortID bool) (string, error) {
	slog.Debug("webfetch: detected Confluence page", "page_id", pageID, "is_short_id", isShortID)

	if isShortID {
		numID, err := w.resolveShortPageID(pageID)
		if err == nil {
			pageID = numID
			slog.Debug("webfetch: resolved short page ID", "short", pageID, "numeric", numID)
		} else {
			slog.Debug("webfetch: failed to resolve short page ID, trying as numeric", "page_id", pageID, "err", err)
		}
	}

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

	apiURL := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,version.history", w.cfg.String("confluence.base_url"), pageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("web: failed to create Confluence request for page %q: %w", pageID, err)
	}

	req.Header.Set("Authorization", "Bearer "+w.cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web: failed to fetch Confluence page %q: %w", pageID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web: Confluence page %q returned status %d", pageID, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", fmt.Errorf("web: decode Confluence page %q error: %w", pageID, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", fmt.Errorf("web: failed to read Confluence response body: %w", err)
	}

	var result struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Version struct {
			Number  int    `json:"number"`
			Author  string `json:"author"`
			Updated string `json:"when"`
		} `json:"version"`
		Body struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("web: failed to parse Confluence response: %w", err)
	}

	slog.Debug("webfetch: Confluence page fetched", "page_id", pageID)
	return fmt.Sprintf("Title: %s\nVersion: %d\nAuthor: %s\nUpdated: %s\nBody:\n%s", result.Title, result.Version.Number, result.Version.Author, result.Version.Updated, result.Body.Storage.Value), nil
}

// fetchJIRAIssue fetches a JIRA issue by its key (e.g., GOARCH-60) and returns its content in Markdown.
func (w *WebTools) fetchJIRAIssue(issueKey string) (string, error) {
	slog.Debug("webfetch: detected JIRA issue", "issue_key", issueKey)

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

	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s", w.cfg.String("jira.base_url"), issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("web: failed to create JIRA request for issue %q: %w", issueKey, err)
	}

	req.Header.Set("Authorization", "Bearer "+w.cfg.String("jira.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web: failed to fetch JIRA issue %q: %w", issueKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web: JIRA issue %q returned status %d", issueKey, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", fmt.Errorf("web: decode JIRA issue %q error: %w", issueKey, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", fmt.Errorf("web: failed to read JIRA response body: %w", err)
	}

	var result struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Self   string `json:"self"`
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Status      struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
			Reporter struct {
				DisplayName string `json:"displayName"`
			} `json:"reporter"`
			Project struct {
				Name string `json:"name"`
			} `json:"project"`
			Type struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Created string `json:"created"`
			Updated string `json:"updated"`
			Comment struct {
				Comments []struct {
					Author struct {
						DisplayName string `json:"displayName"`
					} `json:"author"`
					Body    string `json:"body"`
					Created string `json:"created"`
				} `json:"comments"`
			} `json:"comment"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("web: failed to parse JIRA response: %w", err)
	}

	slog.Debug("webfetch: JIRA issue fetched", "issue_key", issueKey)

	var sb strings.Builder
	sb.WriteString("Issue: ")
	sb.WriteString(result.Key)
	sb.WriteString("\nSummary: ")
	sb.WriteString(result.Fields.Summary)
	sb.WriteString("\nStatus: ")
	sb.WriteString(result.Fields.Status.Name)
	sb.WriteString("\nAssignee: ")
	sb.WriteString(result.Fields.Assignee.DisplayName)
	sb.WriteString("\nReporter: ")
	sb.WriteString(result.Fields.Reporter.DisplayName)
	sb.WriteString("\nProject: ")
	sb.WriteString(result.Fields.Project.Name)
	sb.WriteString("\nType: ")
	sb.WriteString(result.Fields.Type.Name)
	sb.WriteString("\nCreated: ")
	sb.WriteString(result.Fields.Created)
	sb.WriteString("\nUpdated: ")
	sb.WriteString(result.Fields.Updated)
	sb.WriteString("\nDescription:\n")
	sb.WriteString(result.Fields.Description)

	if len(result.Fields.Comment.Comments) > 0 {
		sb.WriteString("\n\nComments:\n")
		for i, comment := range result.Fields.Comment.Comments {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, comment.Author.DisplayName, comment.Created))
			sb.WriteString(comment.Body)
			if i < len(result.Fields.Comment.Comments)-1 {
				sb.WriteString("\n\n")
			}
		}
	}

	return sb.String(), nil
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

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (w *WebTools) Dispatch(name string, args string) string {
	switch name {
	case "web_fetch":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("web_fetch: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, err := w.WebFetch(params.URL)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	default:
		return "error: unknown tool " + name
	}
}

// extractProjectAndMergeRequestFromGitLabURL extracts the project path and merge request ID from a GitLab URL.
func extractProjectAndMergeRequestFromGitLabURL(url string, baseURL string) (string, string, error) {
	// Handle URLs with /-/merge_requests/
	if idx := strings.Index(url, "/-/merge_requests/"); idx != -1 {
		pathPart := url[:idx]
		// Extract project path (remove base URL)

		pathPart = strings.TrimPrefix(pathPart, baseURL)

		idPart := url[idx+18:]
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
	slog.Debug("webfetch: detected GitLab merge request", "project_path", projectPath, "mr_id", mrID)

	gitLabTools := gitlab.New(w.cfg)
	return gitLabTools.FetchMergeRequest(projectPath, mrID)
}

// StaticDefinition returns the OpenAI tool schema for web_fetch without
// requiring an initialised WebTools instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "web_fetch",
			Description: "Fetch a web page and return clean Markdown content. Supports Confluence pages, JIRA issues, and GitLab merge requests.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL of the web page, Confluence page, JIRA issue, or GitLab merge request to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}
