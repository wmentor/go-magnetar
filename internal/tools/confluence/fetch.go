package confluence

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
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
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// NewFetchTool creates a new FetchTool instance for confluence_page_fetch.
func NewFetchTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *FetchTool {
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
