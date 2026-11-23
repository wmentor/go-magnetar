package compactplugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.compact", &Plugin{})
}

// Plugin registers the /compact chat command.
type Plugin struct{}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "compact",
		Help: "Compress conversation history via summarizer ([soft|full]).",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			arg := strings.ToLower(strings.TrimSpace(params.Args))
			switch arg {
			case "", "soft", "full", "compact":
			default:
				var sb strings.Builder
				fmt.Fprintln(&sb, "Usage: /compact [soft|full]")
				return &plugin.ChatCommandResponse{Stdout: sb.String()}, nil
			}

			before := len(params.AgentHandler.Messages())
			var err error
			if arg == "" {
				err = params.AgentHandler.Compact()
			} else {
				err = params.AgentHandler.Compact(arg)
			}
			if err != nil {
				state.Printer.Error("chat: manual compaction failed", "err", err)
				var sb strings.Builder
				fmt.Fprintln(&sb, "Error: compaction failed — see logs for details.")
				return &plugin.ChatCommandResponse{Stdout: sb.String()}, nil
			}
			after := len(params.AgentHandler.Messages())

			var sb strings.Builder
			fmt.Fprintf(&sb, "Context compacted: %d → %d messages.\n\n", before, after)
			return &plugin.ChatCommandResponse{Stdout: sb.String()}, nil
		},
	})
	return nil
}
