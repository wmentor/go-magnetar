package compactplugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.compact", &Plugin{})
}

// Plugin registers the /compact chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "compact",
		Help: "Compress conversation history via summarizer.",
		Execute: func(_ context.Context, a plugin.AgentHandle, _ string) error {
			before := len(a.Messages())
			if err := a.Compact(); err != nil {
				slog.Error("chat: manual compaction failed", "err", err)
				fmt.Fprintln(os.Stdout, "Error: compaction failed — see logs for details.")
				return nil
			}
			after := len(a.Messages())
			fmt.Fprintf(os.Stdout, "Context compacted: %d → %d messages.\n\n", before, after)
			return nil
		},
	})
	return nil
}
