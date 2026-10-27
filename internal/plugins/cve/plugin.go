package cveplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/cve"
)

func init() {
	plugin.Register("cve", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *cve.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s
	p.tool = cve.NewTool(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.FetchVulnerability)
	return nil
}
