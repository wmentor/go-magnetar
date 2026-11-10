package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const confluenceSearchDefaultTimeout = 30 * time.Second

// SearchResult represents a search result from Confluence API.
type SearchResult struct {
	Content struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Status   string `json:"status"`
		Title    string `json:"title"`
		Position int    `json:"position"`
		Links    struct {
			WebUI  string `json:"webui"`
			TinyUI string `json:"tinyui"`
			Self   string `json:"self"`
		} `json:"_links"`
		Expandable struct {
			Container   string `json:"container"`
			Metadata    string `json:"metadata"`
			Extensions  string `json:"extensions"`
			Operations  string `json:"operations"`
			Children    string `json:"children"`
			History     string `json:"history"`
			Ancestors   string `json:"ancestors"`
			Body        string `json:"body"`
			Version     string `json:"version"`
			Descendants string `json:"descendants"`
			Space       string `json:"space"`
		} `json:"_expandable"`
	} `json:"content"`
	Title                 string `json:"title"`
	Excerpt               string `json:"excerpt"`
	URL                   string `json:"url"`
	ResultGlobalContainer struct {
		Title      string `json:"title"`
		DisplayURL string `json:"displayUrl"`
	} `json:"resultGlobalContainer"`
	EntityID             string `json:"entityId"`
	EntityType           string `json:"entityType"`
	IconCSSClass         string `json:"iconCssClass"`
	LastModified         string `json:"lastModified"`
	FriendlyLastModified string `json:"friendlyLastModified"`
	Timestamp            int64  `json:"timestamp"`
}

// SearchParams represents parameters for the confluence_search tool.
type SearchParams struct {
	Query string `json:"query" jsonschema_description:"Search query string for Confluence search (CQL syntax)"`
	Limit int    `json:"limit" jsonschema_description:"Maximum number of results to return (default: 25)"`
}

// SearchTool provides confluence_search operation as LLM tool.
type SearchTool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// NewSearchTool creates a new SearchTool instance for confluence_search.
func NewSearchTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *SearchTool {
	return &SearchTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return "confluence_search"
}

// Description returns the tool description.
func (t *SearchTool) Description() string {
	return "search Confluence pages using CQL (Confluence Query Language) and return results in Markdown format"
}

// Search executes Confluence search and returns results in Markdown format.
func (t *SearchTool) Search(ctx context.Context, params *SearchParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	if t.cfg.String("confluence.base_url") == "" {
		return nil, errors.New("confluence base URL is not configured")
	}

	if t.cfg.String("confluence.api_key") == "" {
		return nil, errors.New("confluence API key is not configured")
	}

	// Set default limit if not specified
	limit := params.Limit
	if limit == 0 {
		limit = 25
	}

	t.printer.ToolCall(printer.IconSearch, "confluence_search", "query", params.Query, "limit", limit)

	// URL-encode the query
	encodedQuery := url.QueryEscape(params.Query)

	// Build the search URL
	// CQL format: text ~ "query" and type=page order by favorite desc
	baseURL := strings.TrimSuffix(t.cfg.String("confluence.base_url"), "/")
	searchURL := fmt.Sprintf("%s/rest/api/search?cql=%s+order+by+favorite+desc&limit=%d",
		baseURL, encodedQuery, limit)

	ctx, cancel := context.WithTimeout(ctx, confluenceSearchDefaultTimeout)
	defer cancel()

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("confluence_search: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.String("confluence.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluence_search: failed to execute search for query %q: %w", params.Query, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence_search: search API returned status %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("confluence_search: decode search results error: %w", err)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("confluence_search: failed to read response body: %w", err)
	}

	var searchResponse SearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("confluence_search: failed to parse response: %w", err)
	}

	// Format results as Markdown
	result := t.formatSearchResults(&searchResponse, baseURL)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: result,
	}, nil
}

// Exec executes the confluence_search tool and returns the result.
func (t *SearchTool) Exec(ctx context.Context, params *SearchParams) (any, error) {
	return t.Search(ctx, params)
}

// SearchResponse represents the Confluence search API response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Start   int            `json:"start"`
	Limit   int            `json:"limit"`
	Size    int            `json:"size"`
}

// formatSearchResults formats search results as Markdown.
func (t *SearchTool) formatSearchResults(response *SearchResponse, baseURL string) string {
	if response == nil || len(response.Results) == 0 {
		return "## Confluence Search Results\n\nNo results found."
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "## Confluence Search Results\n\n")
	fmt.Fprintf(&sb, "**Total Results:** %d\n\n", len(response.Results))

	for i, result := range response.Results {
		// Build the full URL by prepending base URL
		fullURL := baseURL + result.URL

		fmt.Fprintf(&sb, "### %d. %s\n\n", i+1, result.Title)
		fmt.Fprintf(&sb, "**URL:** [%s](%s)\n\n", fullURL, fullURL)

		if result.Excerpt != "" {
			fmt.Fprintf(&sb, "**Excerpt:**\n\n%s\n\n", result.Excerpt)
		}

		if result.Content.ID != "" {
			fmt.Fprintf(&sb, "**Page ID:** %s\n\n", result.Content.ID)
		}

		if result.Content.Expandable.Space != "" {
			fmt.Fprintf(&sb, "**Space:** %s\n\n", result.Content.Expandable.Space)
		}

		if result.Content.Type != "" {
			fmt.Fprintf(&sb, "**Type:** %s\n\n", result.Content.Type)
		}

		if result.Content.Status != "" {
			fmt.Fprintf(&sb, "**Status:** %s\n\n", result.Content.Status)
		}

		if result.FriendlyLastModified != "" {
			fmt.Fprintf(&sb, "**Last Modified:** %s\n\n", result.FriendlyLastModified)
		}

		if result.ResultGlobalContainer.Title != "" {
			fmt.Fprintf(&sb, "**Container:** %s\n\n", result.ResultGlobalContainer.Title)
		}

		if i < len(response.Results)-1 {
			sb.WriteString("---\n\n")
		}
	}

	return sb.String()
}
