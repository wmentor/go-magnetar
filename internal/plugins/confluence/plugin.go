package confluenceplugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func init() {
	plugin.Register("confluence", &Plugin{})
}

// Plugin wraps the Confluence tools and exposes confluence_page_fetch, confluence_page_edit, and confluence_search as LLM tools.
type Plugin struct {
	state      *plugin.State
	fetchTool  *confluence.FetchTool
	editTool   *confluence.EditTool
	searchTool *confluence.SearchTool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("confluence.enable") || s.Config.String("confluence.base_url") == "" {
		return nil
	}

	cfg := s.ConcreteConfig()

	// Initialize fetch tool
	p.fetchTool = confluence.NewFetchTool(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(p.fetchTool.Name(), p.fetchTool.Description(), p.fetchTool.Exec)
	s.WebProcStore.AddProcessorFunc(p.webFetch)

	// Initialize and register edit tool if not in read-only mode
	if !s.Config.Bool("confluence.readonly") {
		p.editTool = confluence.NewEditTool(cfg, s.Printer, s.RequesterFactory, s)
		s.ToolStore.Register(p.editTool.Name(), p.editTool.Description(), p.editTool.Exec)
	}

	// Initialize and register search tool
	p.searchTool = confluence.NewSearchTool(cfg, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(p.searchTool.Name(), p.searchTool.Description(), p.searchTool.Exec)

	return nil
}

func (p *Plugin) webFetch(ctx context.Context, srcURL string) (string, error) {
	cfg := p.state.Config

	if strings.HasPrefix(srcURL, cfg.String("confluence.base_url")+"/spaces/") ||
		strings.HasPrefix(srcURL, cfg.String("confluence.base_url")+"/x/") ||
		strings.HasPrefix(srcURL, cfg.String("confluence.base_url")+"/p/") {
		pageID, err := confluence.ExtractPageIDURL(srcURL)
		if err == nil && pageID != "" {
			isShortID := strings.Contains(srcURL, "/x/") || strings.Contains(srcURL, "/p/")
			// Use FetchTool directly
			fetchTool := confluence.NewFetchTool(cfg, p.state.Printer, p.state.RequesterFactory)
			result, err := fetchTool.FetchPage(ctx, &confluence.FetchPageParams{
				PageID:  pageID,
				IsShort: isShortID,
			})
			if err != nil {
				p.state.Printer.Error("confluence_page_fetch: failed to fetch page", "url", srcURL, "err", err)
				return "", fmt.Errorf("confluence_page_fetch: failed to fetch page %q: %w", srcURL, err)
			}
			if resp, ok := result.(*tools.Response); ok {
				return resp.Result, nil
			}
			return "", fmt.Errorf("confluence_page_fetch: unexpected result type: %T", result)
		}
	}

	return "", plugin.ErrNoWebProcessor
}
