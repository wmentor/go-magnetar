package telegramplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/telegram"
)

func init() {
	plugin.Register("telegram", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *telegram.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	// Only register the tool if telegram.enable is true
	if !s.Config.Bool("telegram.enable") {
		return nil
	}

	p.tool = telegram.New(s.Config, s.Printer, s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)

	return nil
}
