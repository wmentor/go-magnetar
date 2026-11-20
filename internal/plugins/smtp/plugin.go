package smtpplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/smtp"
)

func init() {
	plugin.Register("smtp", &Plugin{})
}

// Plugin wraps the SMTP tool and registers it as an LLM tool when enabled.
type Plugin struct {
	state *plugin.State
	tool  *smtp.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("smtp.enable") {
		return nil
	}

	p.tool = smtp.New(s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec, tools.OptionSequential())

	return nil
}
