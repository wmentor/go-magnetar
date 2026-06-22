package web

import (
	"context"
	"crypto/tls"
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
	if cfg.WebFetch.BaseURL != "" {
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

// StaticDefinition returns the OpenAI tool schema for web_fetch without
// requiring an initialised WebTools instance. Used by the plugin for lazy init.
func StaticDefinition() openai.Tool {
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
