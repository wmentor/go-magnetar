package confluenceplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func init() {
	plugin.Register("confluence", &Plugin{})
}

// Plugin wraps the Confluence tools and exposes confluence_page_fetch and confluence_page_edit as LLM tools.
type Plugin struct {
	state     *plugin.State
	fetchTool *confluence.FetchTool
	editTool  *confluence.EditTool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("confluence.enable") || s.Config.String("confluence.base_url") == "" {
		return nil
	}

	// Initialize fetch tool
	p.fetchTool = confluence.NewFetchTool(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(p.fetchTool.Name(), p.fetchTool.Description(), p.fetchTool.Exec)

	// Initialize and register edit tool if not in read-only mode
	if !s.Config.Bool("confluence.readonly") {
		p.editTool = confluence.NewEditTool(s.Config, s.Printer, s.RequesterFactory, s)
		s.ToolStore.Register(p.editTool.Name(), p.editTool.Description(), p.editTool.Exec)
	}

	return nil
}
