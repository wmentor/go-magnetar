package lastresponse

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/lastresponse"
)

func init() {
	plugin.Register("lastresponse", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *lastresponse.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s
	p.tool = lastresponse.New(s.Config, s.Printer, s.RequesterFactory, s.ResponseStore)
	// Register using the modern tool store API
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)
	return nil
}
