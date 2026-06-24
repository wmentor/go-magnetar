package versionplugin

import (
	"context"
	"debug/buildinfo"
	"fmt"
	"os"

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
			f, err := os.Open(os.Args[0])
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error opening binary: %v\n", err)
				return nil
			}
			defer f.Close()

			bi, err := buildinfo.Read(f)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error reading build info: %v\n", err)
				return nil
			}

			fmt.Fprintf(os.Stdout, "Version: %s\n", bi.Main.Version)
			return nil
		},
	})
	return nil
}
