package exitplugin

import (
	"context"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.exit", &Plugin{})
}

// Plugin registers the /exit chat command.
type Plugin struct{}

func (p *Plugin) Init(_ plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "exit",
		Aliases: []string{"quit"},
		Help:    "End the session and exit.",
		Execute: func(_ context.Context, _ plugin.AgentHandle, _ string) error {
			return plugin.ErrExit
		},
	})
	return nil
}
