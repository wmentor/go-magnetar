package agentplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/agent"
)

func init() {
	plugin.Register("agent", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *agent.Tool
}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.state = state
	p.tool = agent.NewTool(state.Config, state.Printer, state.RequesterFactory, state)
	state.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec, tools.OptionAgent())
	return nil
}
