package webplugin

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/web/webfetch"
	"github.com/wmentor/go-magnetar/internal/tools/web/websearch"
)

func init() {
	plugin.Register("web", &Plugin{})
}

const (
	webSearchTimeInterval = time.Second * 4
)

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	// Register web_fetch tool
	if !s.Config.Bool("web.disable.fetch") {
		fetchTool := webfetch.NewTool(s.Config, s.Printer, s.RequesterFactory, s)
		s.ToolStore.Register(fetchTool.Name(), fetchTool.Description(), fetchTool.WebFetch)
	}

	// Register web_search tool
	if !s.Config.Bool("web.disable.search") {
		searchTool := websearch.NewTool(s.Config, s.Printer, s.RequesterFactory, s)
		limit := rate.Every(webSearchTimeInterval)
		limiter := rate.NewLimiter(limit, 1)
		searchTool.SetLimiter(limiter)
		s.ToolStore.Register(searchTool.Name(), searchTool.Description(), searchTool.WebSearch)
	}

	return nil
}
