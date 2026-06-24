package readonlyplugin

import (
	"context"
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.readonly", &Plugin{})
}

type Plugin struct {
	readonly bool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.readonly = false
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "readonly",
		Help:    "Enable or disable read-only mode (on|off).",
		Execute: p.execute,
	})
	return nil
}

func (p *Plugin) execute(_ context.Context, _ plugin.AgentHandle, args string) error {
	switch args {
	case "on":
		p.readonly = true
		fmt.Fprintln(os.Stdout, "Read-only mode enabled.")
	case "off":
		p.readonly = false
		fmt.Fprintln(os.Stdout, "Read-only mode disabled.")
	default:
		if p.readonly {
			fmt.Fprintln(os.Stdout, "Read-only mode is currently ON.")
		} else {
			fmt.Fprintln(os.Stdout, "Read-only mode is currently OFF.")
		}
		return nil
	}
	return nil
}
