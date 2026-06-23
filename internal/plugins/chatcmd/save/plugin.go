package saveplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.save", &Plugin{})
}

// Plugin registers the /save chat command.
type Plugin struct{}

func (p *Plugin) Init(_ plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "save",
		Help:    "Save the last assistant answer to a file.",
		Aliases: []string{"s"},
		Execute: func(_ context.Context, a plugin.AgentHandle, args string) error {
			if args == "" {
				fmt.Fprintln(os.Stdout, "Usage: /save <filename>")
				return nil
			}

			filename := filepath.Clean(args)
			msgs := a.Messages()

			if len(msgs) < 2 {
				fmt.Fprintln(os.Stdout, "Nothing to save: no conversation history")
				return nil
			}

			lastMsg := msgs[len(msgs)-1]
			if lastMsg.Role != "assistant" {
				lastMsg = msgs[len(msgs)-2]
			}

			content := lastMsg.Content
			if content == "" {
				fmt.Fprintln(os.Stdout, "Nothing to save: last assistant message is empty")
				return nil
			}

			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stdout, "Error saving file: %v\n", err)
				return nil
			}

			fmt.Fprintf(os.Stdout, "Answer saved to %s\n", filename)
			return nil
		},
	})
	return nil
}
