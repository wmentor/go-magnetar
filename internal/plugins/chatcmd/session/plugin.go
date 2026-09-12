package sessionplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

var (
	out io.Writer = os.Stdout
)

func init() {
	plugin.Register("chatcmd.save", &Plugin{})
}

// Plugin registers the /save.session chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "session.save",
		Aliases: []string{},
		Help:    "Save current conversation session to a file.",
		Execute: func(_ context.Context, a plugin.AgentHandle, args string) error {
			if args == "" {
				fmt.Fprintln(out, "Usage: /session.save <filename>")
				return nil
			}

			filename := filepath.Clean(args)
			msgs := a.Messages()

			data, err := json.MarshalIndent(msgs, "", "  ")
			if err != nil {
				fmt.Fprintf(out, "Error marshaling messages: %v\n", err)
				return nil
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				fmt.Fprintf(out, "Error saving file: %v\n", err)
				return nil
			}

			fmt.Fprintf(out, "Session saved to %s\n", filename)
			return nil
		},
	})
	return nil
}
