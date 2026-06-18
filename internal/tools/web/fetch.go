package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/agent/html"
	"github.com/wmentor/go-magnetar/internal/config"
)

const defaultTimeout = time.Minute

// WebTools provides web fetching operations as LLM tools.
type WebTools struct {
	cfg          *config.Config
	preprocessor *html.Preprocessor
}

// New creates a new WebTools instance.
func New(cfg *config.Config) (*WebTools, error) {
	preprocessor, err := html.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("web: failed to create preprocessor: %w", err)
	}

	return &WebTools{
		cfg:          cfg,
		preprocessor: preprocessor,
	}, nil
}

// fetchURL retrieves the HTML content from a URL.
func (w *WebTools) fetchURL(url string) (string, error) {
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
		return "", fmt.Errorf("web: failed to create request for %q: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web: failed to fetch URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web: URL %q returned status %d", url, resp.StatusCode)
	}

	utf8, err1 := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err1 != nil {
		return "", fmt.Errorf("web: decode URL %q error: %w", url, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", fmt.Errorf("web: failed to read response body: %w", err)
	}

	return string(body), nil
}

// preprocessHTML preprocesses HTML content using the preprocessor agent.
func (w *WebTools) preprocessHTML(htmlStr string) (string, error) {
	return w.preprocessor.ProcessHTMLString(htmlStr)
}

// WebFetch fetches a web page, preprocesses it, and returns the cleaned Markdown content.
func (w *WebTools) WebFetch(url string) (string, error) {
	slog.Info("webfetch: fetching URL", "url", url)

	htmlContent, err := w.fetchURL(url)
	if err != nil {
		slog.Error("webfetch: failed to fetch URL", "url", url, "err", err)
		return "", fmt.Errorf("webfetch: failed to fetch URL %q", url)
	}

	slog.Debug("webfetch: HTML fetched, preprocessing", "url", url)

	markdown, err := w.preprocessHTML(htmlContent)
	if err != nil {
		slog.Error("webfetch: preprocessing failed", "url", url, "err", err)
		return "", fmt.Errorf("webfetch: preprocessing failed for URL %q", url)
	}

	slog.Info("webfetch: done", "url", url)
	return markdown, nil
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
