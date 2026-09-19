package copyplugin

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/atotto/clipboard"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.copy", &Plugin{})
}

var (
	out io.Writer = os.Stdout
)

// Plugin registers the /copy chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "copy",
		Aliases: []string{"c"},
		Help:    "Copy the last assistant answer to clipboard.",
		Execute: func(_ context.Context, a plugin.AgentHandle, _ string) error {
			msgs := a.Messages()

			if len(msgs) < 2 {
				fmt.Fprintln(out, "Nothing to copy: no conversation history")
				return nil
			}

			lastMsg := msgs[len(msgs)-1]
			if lastMsg.Role != "assistant" {
				lastMsg = msgs[len(msgs)-2]
			}

			content := lastMsg.Content
			if content == "" {
				fmt.Fprintln(out, "Nothing to copy: last assistant message is empty")
				return nil
			}

			if err := clipboard.WriteAll(content); err != nil {
				fmt.Fprintf(out, "Error copying to clipboard: %v\n", err)
				return nil
			}

			fmt.Fprintln(out, "Answer copied to clipboard")
			return nil
		},
	})
	return nil
}
