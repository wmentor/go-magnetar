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
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/codec/html"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/rss"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

const (
	webFetchDefaultTimeout = time.Minute
	userAgent              = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

// Tool provides web_fetch operation as LLM tool.
type Tool struct {
	cfg           plugin.Config
	printer       printer.Printer
	clientFactory plugin.ClientFactory
	state         *plugin.State
}

// NewTool creates a new Tool instance.
func NewTool(cfg plugin.Config, p printer.Printer, clientFactory plugin.ClientFactory, state *plugin.State) *Tool {
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

// NameMCP returns the tool name for the MCP server.
func (t *Tool) NameMCP() string {
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
	ctx, cancel := context.WithTimeout(ctx, webFetchDefaultTimeout)
	defer cancel()

	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}

	webRet, webErr := t.state.WebProcStore.Process(ctx, urlStr)
	if webErr == nil {
		return webRet, nil
	}
	if !errors.Is(webErr, plugin.ErrNoWebProcessor) {
		return "", errors.Wrap(webErr, "web fetch enigine")
	}

	body, contentType, err := t.fetchURLWithContentType(ctx, urlStr)
	if err != nil {
		if isFetch {
			t.printer.Error("web_fetch: failed to fetch URL", "url", urlStr, "err", err)
			return "", fmt.Errorf("web_fetch: failed to fetch URL %q", urlStr)
		}
		t.printer.Error("web_search: failed to fetch URL", "err", err)
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

func (t *Tool) mimeCodecWork(myCodec codec.ReadCodec, content []byte) (string, error) {
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
