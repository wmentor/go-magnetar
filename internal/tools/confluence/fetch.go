package confluence

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// const confluenceDefaultTimeout is now in tools.go

// Page represents a Confluence page structure.
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

// FetchPageParams represents parameters for the confluence_page_fetch tool.
type FetchPageParams struct {
	PageID  string `json:"page_id" jsonschema_description:"Confluence page ID (numeric ID or short ID)"`
	IsShort bool   `json:"is_short" jsonschema_description:"Whether the page_id is a short ID (e.g., AgA5)"`
}

// FetchTool provides confluence_page_fetch operation as LLM tool.
type FetchTool struct {
	cfg           plugin.Config
	printer       printer.Printer
	clientFactory plugin.ClientFactory
}

// NewFetchTool creates a new FetchTool instance for confluence_page_fetch.
func NewFetchTool(cfg plugin.Config, p printer.Printer, clientFactory plugin.ClientFactory) *FetchTool {
	return &FetchTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *FetchTool) Name() string {
	return "confluence_page_fetch"
}

// Description returns the tool description.
func (t *FetchTool) Description() string {
	return "Fetch a Confluence page by ID and return its content in Markdown format"
}

// FetchPage fetches a Confluence page by ID and returns its content in Markdown.
func (t *FetchTool) FetchPage(ctx context.Context, params *FetchPageParams) (any, error) {
	if params == nil {
		return nil, fmt.Errorf("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "confluence_page_fetch", "page_id", params.PageID, "is_short", params.IsShort)

	page, err := t.fetchPage(ctx, params.PageID, params.IsShort)
	if err != nil {
		return nil, fmt.Errorf("confluence_page_fetch: failed to fetch page: %w", err)
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "Confluence page ID: %s\n", page.ID)
	fmt.Fprintf(&sb, "Space: %s\n", page.Space.Key)
	fmt.Fprintf(&sb, "Title: %s\n", page.Title)
	fmt.Fprintf(&sb, "Created: %s\n", page.History.Created)
	fmt.Fprintf(&sb, "Author: %s (%s)\n", page.History.CreatedBy.DisplayName, page.History.CreatedBy.Username)
	fmt.Fprintf(&sb, "Updated: %s\n", page.Version.Updated)
	fmt.Fprintf(&sb, "Version: %d\n", page.Version.Number)
	fmt.Fprintf(&sb, "Body:\n%s\n", page.Body.Storage.Value)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}

func (t *FetchTool) fetchPage(ctx context.Context, pageID string, isShortID bool) (*Page, error) {
	return fetchPage(ctx, t.cfg, t.printer, t.clientFactory, pageID, isShortID)
}

// Exec executes the confluence_page_fetch tool and returns the result.
func (t *FetchTool) Exec(ctx context.Context, params *FetchPageParams) (any, error) {
	return t.FetchPage(ctx, params)
}
