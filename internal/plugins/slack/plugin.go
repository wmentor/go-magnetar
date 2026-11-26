package slackplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/slack"
)

func init() {
	plugin.Register("slack", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *slack.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("slack.enable") {
		return nil
	}

	p.tool = slack.New(s.Config, s.Printer, s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)

	return nil
}
