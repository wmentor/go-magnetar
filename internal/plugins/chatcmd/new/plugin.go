package newplugin

import (
	"context"
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.new", &Plugin{})
}

// Plugin registers the /new chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "new",
		Help: "Start a new session and clear conversation history.",
		Execute: func(_ context.Context, a plugin.AgentHandle, _ string) error {
			a.Reset()
			fmt.Fprintln(os.Stdout, "New session started. Context cleared.")
			return nil
		},
	})
	return nil
}
