package confluence

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const confluenceDefaultTimeout = time.Minute

// ConfluenceTools provides backward compatibility for existing code.
// Deprecated: Use FetchTool or EditTool instead.
type ConfluenceTools struct{}

// New creates a new ConfluenceTools instance.
// Deprecated: Use NewFetchTool or NewEditTool instead.
func New(cfg *config.Config, state *plugin.State) *ConfluenceTools {
	// Deprecated: This function is kept for backward compatibility but should not be used.
	// Use NewFetchTool or NewEditTool instead.
	return &ConfluenceTools{}
}

// ResolveShortPageID resolves a short Confluence page code (e.g., AgA5) to numeric ID.
func ResolveShortPageID(shortCode string) (string, error) {
	pageID, err := decodeShortPageID(shortCode)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", pageID), nil
}

// decodeShortPageID decodes a Confluence short page ID (e.g., "A4HhC") to numeric ID using Base64.
// Confluence pads the code to 11 chars with 'A' and adds '=' for padding, then decodes to 32-bit LE integer.
func decodeShortPageID(shortCode string) (int64, error) {
	// 1. Replace Confluence-specific URL-safe characters with standard Base64 characters
	paddedCode := strings.ReplaceAll(shortCode, "_", "+")
	paddedCode = strings.ReplaceAll(paddedCode, "-", "/")

	// 2. Pad to 11 characters with 'A' and add '='
	if len(paddedCode) < 11 {
		paddedCode = paddedCode + strings.Repeat("A", 11-len(paddedCode))
	}
	paddedCode += "="

	// 3. Decode Base64 to bytes
	decoded, err := base64.StdEncoding.DecodeString(paddedCode)
	if err != nil {
		return 0, fmt.Errorf("confluence: failed to decode short code %q: %w", shortCode, err)
	}

	// 4. Unpack 32-bit Little-Endian integer using binary package
	if len(decoded) < 4 {
		return 0, fmt.Errorf("confluence: decoded data too short for page ID")
	}
	pageID := binary.LittleEndian.Uint32(decoded[:4])

	return int64(pageID), nil
}

// HTMLFixTags fixes HTML tags for Confluence compatibility.
func HTMLFixTags(content string) string {
	forReplace := map[string]string{
		"<br>":  "<br/>",
		"<hr>":  "<hr/>",
		"<img>": "<img/>",
	}

	for k, v := range forReplace {
		content = strings.ReplaceAll(content, k, v)
	}

	forRemove := []string{"</br>", "</hr>", "</img>"}
	for _, rec := range forRemove {
		content = strings.ReplaceAll(content, rec, "")
	}

	return content
}

// ConvertMarkdown converts Markdown content to HTML for Confluence.
func ConvertMarkdown(markdownContent string) (string, error) {
	// Convert Markdown to HTML using goldmark
	var buf strings.Builder
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML including Confluence tags
		),
	)
	err := markdown.Convert([]byte(markdownContent), &buf)
	if err != nil {
		return "", fmt.Errorf("confluence: failed to convert markdown to HTML: %w", err)
	}

	return HTMLFixTags(buf.String()), nil
}

// FetchPageURL parses a Confluence URL and returns the page ID.
func FetchPageURL(url string) (string, error) {
	// Handle short link: .../x/{page_id}
	if _, after, ok := strings.Cut(url, "/x/"); ok {
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
			return "", fmt.Errorf("page ID is empty")
		}
		return idPart, nil
	}

	// Handle share link: .../p/{page_id}
	if _, after, ok := strings.Cut(url, "/p/"); ok {
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

// ExtractPageIDURL parses a Confluence URL and returns the page ID.
// Deprecated: Use FetchPageURL instead.
func ExtractPageIDURL(url string) (string, error) {
	return FetchPageURL(url)
}

// fetchPage fetches a Confluence page by ID and returns the parsed page.
// This is a shared helper for both FetchTool and EditTool.
func fetchPage(
	ctx context.Context,
	cfg *config.Config,
	p printer.Printer,
	clientFactory requester.ClientFactory,
	pageID string,
	isShortID bool,
) (*Page, error) {
	if isShortID {
		numID, err := ResolveShortPageID(pageID)
		if err == nil {
			pageID = numID
			p.ToolCall(printer.IconSearch, "confluence_page_fetch: resolved short page ID", "short", pageID, "numeric", numID)
		} else {
			p.ToolCall(printer.IconError, "confluence_page_fetch: failed to resolve short page ID, trying as numeric",
				"page_id", pageID, "err", err)
			return nil, fmt.Errorf("confluence_page_fetch: failed to resolve short page ID=%s: %w", pageID, err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, confluenceDefaultTimeout)
	defer cancel()

	client := clientFactory.Get()
	defer clientFactory.Put(client)

	apiURL := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,space,version,history.createdBy",
		cfg.String("confluence.base_url"), pageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: failed to create request for page %q: %w", pageID, err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: failed to fetch page %q: %w", pageID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("confluence_page_fetch: page %q returned status %d", pageID, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: decode page %q error: %w", pageID, err)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: failed to read response body: %w", err)
	}

	page := &Page{}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: failed to parse response: %w", err)
	}

	return page, nil
}

// StaticDefinitionFetch returns the OpenAI tool schema for confluence_page_fetch without
// requiring an initialised FetchTool instance. Used by the plugin for lazy init.
func StaticDefinitionFetch() any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "confluence_page_fetch",
			"description": "Fetch a Confluence page by ID and return its content in Markdown format",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{
						"type":        "string",
						"description": "Confluence page ID (numeric ID or short ID)",
					},
					"is_short": map[string]any{
						"type":        "boolean",
						"description": "Whether the page_id is a short ID (e.g., AgA5)",
					},
				},
				"required": []string{"page_id"},
			},
		},
	}
}

// StaticDefinitionEdit returns the OpenAI tool schema for confluence_page_edit without
// requiring an initialised EditTool instance. Used by the plugin for lazy init.
func StaticDefinitionEdit() any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "confluence_page_edit",
			"description": "Edit a Confluence page by ID with Markdown content. The content will be converted to HTML and saved to the page.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{
						"type":        "string",
						"description": "Confluence page ID (numeric ID)",
					},
					"markdown": map[string]any{
						"type":        "string",
						"description": "New page content in Markdown format",
					},
				},
				"required": []string{"page_id", "markdown"},
			},
		},
	}
}
