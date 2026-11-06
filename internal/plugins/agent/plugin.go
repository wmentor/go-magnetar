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
	tools []*agent.Tool
}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.state = state
	p.tools = make([]*agent.Tool, 0)

	// Register a tool for each available agent
	for _, agentDef := range state.AgentStore.List() {
		if agentDef.Disable {
			continue
		}

		tool := agent.NewTool(agentDef.Name, state)
		p.tools = append(p.tools, tool)
		state.ToolStore.Register(tool.Name(), tool.Description(), tool.Exec, tools.OptionAgent())
	}

	return nil
}
