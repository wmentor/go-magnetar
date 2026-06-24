package versionplugin

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.version", &Plugin{})
}

// Plugin registers the /version chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "version",
		Help: "Show the current program version.",
		Execute: func(_ context.Context, a plugin.AgentHandle, args string) error {
			if info, ok := debug.ReadBuildInfo(); ok {
				fmt.Printf("Version: %s\n", info.Main.Version)
			} else {
				fmt.Println("Version: dev")
			}
			return nil
		},
	})
	return nil
}
