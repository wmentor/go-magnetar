package websearch

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/web/webfetch"
)

// Tool provides web_search operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	state         *plugin.State
	limiter       *rate.Limiter
}

// NewTool creates a new Tool instance.
func NewTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		state:         state,
	}
}

func (t *Tool) SetLimiter(limiter *rate.Limiter) {
	t.limiter = limiter
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "web_search"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Execute web search and return results in Markdown format"
}

// Params represents parameters for the web_search tool.
type Params struct {
	Query string `json:"query" jsonschema_description:"Search query string for web search"`
}

// WebSearch executes web search and returns results in Markdown format.
func (t *Tool) WebSearch(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	if t.limiter != nil {
		if err := t.limiter.Wait(ctx); err != nil {
			return nil, errors.Wrap(err, "rate limiter error")
		}
	}

	t.printer.ToolCall(printer.IconSearch, "web_search", "query", params.Query)
	encodedQuery := url.QueryEscape(params.Query)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", encodedQuery)

	// Use web_fetch to get the search results page
	fetchTool := webfetch.NewTool(t.cfg, t.printer, t.clientFactory, t.state)
	result, err := fetchTool.WebFetchEngine(ctx, searchURL, false)
	if err != nil {
		t.printer.Error("web_search: failed to execute search", "query", params.Query, "err", err)
		return nil, fmt.Errorf("web_search: failed to execute search for query %q: %w", params.Query, err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: result,
	}, nil
}

