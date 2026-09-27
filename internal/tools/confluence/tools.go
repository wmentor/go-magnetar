package confluence

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const (
	defaultTimeout = time.Minute
)

// ConfluenceTools provides Confluence page fetching as an LLM tool.
type ConfluenceTools struct {
	cfg   *config.Config
	state *plugin.State
}

type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Space struct {
		Key string `json:"key"`
	} `json:"space"`
	Version struct {
		Number  int    `json:"number"`
		Updated string `json:"when"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	History struct {
		CreatedBy struct {
			Username    string `json:"username"`
			DisplayName string `json:"displayName"`
		} `json:"createdBy"`
		Created string `json:"createdDate"`
	} `json:"history"`
}

// New creates a new ConfluenceTools instance.
func New(cfg *config.Config, state *plugin.State) *ConfluenceTools {
	return &ConfluenceTools{cfg: cfg, state: state}
}

func (c *ConfluenceTools) fetchPage(pageID string, isShortID bool) (*Page, error) {
	printer.ToolCall(printer.IconSearch, "confluence: detected Confluence page", "page_id", pageID, "is_short_id", isShortID)

	if isShortID {
		numID, err := ResolveShortPageID(pageID)
		if err == nil {
			pageID = numID
			printer.ToolCall(printer.IconSearch, "confluence: resolved short page ID", "short", pageID, "numeric", numID)
		} else {
			printer.ToolCall(printer.IconError, "confluence: failed to resolve short page ID, trying as numeric",
				"page_id", pageID, "err", err)
			return nil, fmt.Errorf("confluence: failed to result short pageID=%s", pageID)
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

	apiURL := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,space,version,history.createdBy",
		c.cfg.String("confluence.base_url"), pageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("confluence: failed to create Confluence request for page %q: %w", pageID, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluence: failed to fetch Confluence page %q: %w", pageID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("confluence: Confluence page %q returned status %d", pageID, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return nil, fmt.Errorf("confluence: decode Confluence page %q error: %w", pageID, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("confluence: failed to read Confluence response body: %w", err)
	}

	page := &Page{}

	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("confluence: failed to parse Confluence response: %w", err)
	}

	return page, nil
}

// FetchPage fetches a Confluence page by ID and returns its content in Markdown.
func (c *ConfluenceTools) FetchPage(pageID string, isShortID bool) (string, error) {
	page, err := c.fetchPage(pageID, isShortID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("Confluence page ID: ")
	sb.WriteString(page.ID)
	sb.WriteRune('\n')

	sb.WriteString("Space: ")
	sb.WriteString(page.Space.Key)
	sb.WriteRune('\n')

	sb.WriteString("Title: ")
	sb.WriteString(page.Title)
	sb.WriteRune('\n')

	sb.WriteString("Created: ")
	sb.WriteString(page.History.Created)
	sb.WriteRune('\n')

	sb.WriteString("Author: ")
	sb.WriteString(page.History.CreatedBy.DisplayName)
	sb.WriteString(" (")
	sb.WriteString(page.History.CreatedBy.Username)
	sb.WriteString(")\n")

	sb.WriteString("Updated: ")
	sb.WriteString(page.Version.Updated)
	sb.WriteRune('\n')

	fmt.Fprintf(&sb, "Version: %d\n", page.Version.Number)

	sb.WriteString("Body:\n")
	sb.WriteString(page.Body.Storage.Value)

	return sb.String(), nil
}

// resolveShortPageID resolves a short Confluence page code (e.g., AgA5) to numeric ID.
func ResolveShortPageID(shortCode string) (string, error) {
	printer.ToolCall(printer.IconSearch, "confluence: resolving short Confluence page ID", "short_code", shortCode)

	pageID, err := decodeShortPageID(shortCode)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(pageID, 10), nil
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
		return 0, fmt.Errorf("confluence: failed to decode short code %q: %w", shortCode, err)
	}

	// 3. Unpack 32-bit Little-Endian integer using binary package
	if len(decoded) < 4 {
		return 0, errors.New("confluence: decoded data too short for page ID")
	}
	pageID := binary.LittleEndian.Uint32(decoded[:4])

	return int64(pageID), nil
}

// EditPage edits a Confluence page by ID with Markdown content.
func (c *ConfluenceTools) EditPage(pageID string, markdownContent string) (string, error) {
	printer.ToolCall(printer.IconSave, "confluence_page_edit", "page_id", pageID)
	if c.state.ReadOnly {
		printer.ToolCall(printer.IconError, "confluence_page_edit: read-only mode", "page_id", pageID)
		return "", errors.New("confluence_page_edit: forbidden read-only mode")
	}

	// Fetch current page to get space key, title, and version
	page, err := c.fetchPage(pageID, false)
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to fetch page: %w", err)
	}

	// Convert Markdown to HTML using goldmark
	var buf strings.Builder
	markdown := goldmark.New(
		goldmark.WithExtensions(extension.Typographer),
	)
	err = markdown.Convert([]byte(markdownContent), &buf)
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to convert markdown to HTML: %w", err)
	}
	htmlContent := strings.TrimSpace(buf.String())

	printer.ToolCall(printer.IconSave, "confluence_page_edit: converted markdown to HTML", "page_id", pageID)

	// Prepare update request
	updateData := struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Title string `json:"title"`
		Space struct {
			Key string `json:"key"`
		} `json:"space"`
		Body struct {
			Storage struct {
				Value string `json:"value"`
				Repr  string `json:"representation"`
			} `json:"storage"`
		} `json:"body"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}{
		ID:    pageID,
		Type:  "page",
		Title: page.Title,
		Space: struct {
			Key string `json:"key"`
		}{Key: page.Space.Key},
		Body: struct {
			Storage struct {
				Value string `json:"value"`
				Repr  string `json:"representation"`
			} `json:"storage"`
		}{
			Storage: struct {
				Value string `json:"value"`
				Repr  string `json:"representation"`
			}{
				Value: htmlContent,
				Repr:  "storage",
			},
		},
		Version: struct {
			Number int `json:"number"`
		}{
			Number: page.Version.Number + 1,
		},
	}

	// Marshal to JSON
	bodyBytes, err := json.Marshal(updateData)
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to marshal update data: %w", err)
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

	apiURL := fmt.Sprintf("%s/rest/api/content/%s", c.cfg.String("confluence.base_url"), pageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to update page %q: %w", pageID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("confluence_page_edit: page %q update returned status %d: %s", pageID, resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to read response: %w", err)
	}

	var result struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("confluence_page_edit: failed to parse response: %w", err)
	}

	printer.ToolCall(printer.IconSave, "confluence_page_edit: page updated successfully",
		"page_id", pageID,
		"version", result.Version.Number,
		"title", result.Title)

	return fmt.Sprintf("Page updated successfully:\nID: %s\nTitle: %s\nVersion: %d", result.ID, result.Title, result.Version.Number), nil
}

// EditPageInput represents the input parameters for EditPage.
type EditPageInput struct {
	PageID   string `json:"page_id"`
	Markdown string `json:"markdown"`
}

// DefinitionEdit returns the OpenAI tool schema for confluence_page_edit.
func (c *ConfluenceTools) DefinitionEdit() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "confluence_page_edit",
			Description: "Edit a Confluence page by ID with Markdown content. The content will be converted to HTML and saved to the page.",
			Parameters: map[string]any{
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

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (c *ConfluenceTools) Dispatch(name string, args string) string {
	switch name {
	case "confluence_page_edit":
		var params EditPageInput
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.ToolCall(printer.IconError, "confluence_page_edit: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result, err := c.EditPage(params.PageID, params.Markdown)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return result

	default:
		return "error: unknown tool " + name
	}
}

// StaticDefinitionEdit returns the OpenAI tool schema for confluence_page_edit without
// requiring an initialised ConfluenceTools instance. Used by the plugin for lazy init.
func StaticDefinitionEdit() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "confluence_page_edit",
			Description: "Edit a Confluence page by ID with Markdown content. The content will be converted to HTML and saved to the page.",
			Parameters: map[string]any{
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

// ExtractPageIDURL parses a Confluence URL and returns the page ID.
func ExtractPageIDURL(url string) (string, error) {
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
