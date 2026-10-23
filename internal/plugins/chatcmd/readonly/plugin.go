package readonlyplugin

import (
	"context"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.readonly", &Plugin{})
}

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "readonly",
		Help:    "Enable or disable read-only mode (on|off).",
		Execute: p.execute,
	})
	return nil
}

func (p *Plugin) execute(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := params.Args
	switch args {
	case "on":
		p.state.ReadOnly = true
		return &plugin.ChatCommandResponse{
			Stdout: "Read-only mode enabled.\n",
		}, nil
	case "off":
		p.state.ReadOnly = false
		return &plugin.ChatCommandResponse{
			Stdout: "Read-only mode disabled.\n",
		}, nil

	default:
		if p.state.ReadOnly {
			return &plugin.ChatCommandResponse{
				Stdout: "Read-only mode is currently ON.\n",
			}, nil
		} else {
			return &plugin.ChatCommandResponse{
				Stdout: "Read-only mode is currently OFF.\n",
			}, nil
		}
	}
}
