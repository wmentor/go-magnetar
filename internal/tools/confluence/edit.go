package confluence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// EditPageParams represents parameters for the confluence_page_edit tool.
type EditPageParams struct {
	PageID   string `json:"page_id" jsonschema_description:"Confluence page ID (numeric ID)"`
	Markdown string `json:"markdown" jsonschema_description:"New page content in Markdown format"`
}

// EditTool provides confluence_page_edit operation as LLM tool.
type EditTool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	state         *plugin.State
}

// NewEditTool creates a new EditTool instance for confluence_page_edit.
func NewEditTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, state *plugin.State) *EditTool {
	return &EditTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		state:         state,
	}
}

// Name returns the tool name.
func (t *EditTool) Name() string {
	return "confluence_page_edit"
}

// Description returns the tool description.
func (t *EditTool) Description() string {
	return "Edit a Confluence page by ID with Markdown content. The content will be converted to HTML and saved to the page."
}

// EditPage edits a Confluence page by ID with Markdown content.
func (t *EditTool) EditPage(ctx context.Context, params *EditPageParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSave, "confluence_page_edit", "page_id", params.PageID)

	if t.state.ReadOnly || t.cfg.Bool("confluence.readonly") {
		t.printer.Error("confluence_page_edit: read-only mode enabled", "page_id", params.PageID)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: read-only mode is enabled",
		}, nil
	}

	// Fetch current page to get space key, title, and version
	page, err := t.fetchPage(ctx, params.PageID, false)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to fetch page: %w", err)
	}

	// Convert Markdown to HTML using goldmark
	htmlContent, err := ConvertMarkdown(params.Markdown)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to convert markdown to HTML: %w", err)
	}

	t.printer.ToolCall(printer.IconSave, "confluence_page_edit: converted markdown to HTML", "page_id", params.PageID)

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
		ID:    params.PageID,
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
		return nil, fmt.Errorf("confluence_page_edit: failed to marshal update data: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, confluenceDefaultTimeout)
	defer cancel()

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	apiURL := fmt.Sprintf("%s/rest/api/content/%s", t.cfg.String("confluence.base_url"), params.PageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to update page %q: %w", params.PageID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence_page_edit: page %q update returned status %d: %s", params.PageID, resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to read response: %w", err)
	}

	var result struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("confluence_page_edit: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconSave, "confluence_page_edit: page updated successfully",
		"page_id", params.PageID,
		"version", result.Version.Number,
		"title", result.Title)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: fmt.Sprintf("Page updated successfully:\nID: %s\nTitle: %s\nVersion: %d", result.ID, result.Title, result.Version.Number),
	}, nil
}

func (t *EditTool) fetchPage(ctx context.Context, pageID string, isShortID bool) (*Page, error) {
	return fetchPage(ctx, t.cfg, t.printer, t.clientFactory, pageID, isShortID)
}

// Exec executes the confluence_page_edit tool and returns the result.
func (t *EditTool) Exec(ctx context.Context, params *EditPageParams) (any, error) {
	return t.EditPage(ctx, params)
}
