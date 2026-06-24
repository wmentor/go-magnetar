package lessplugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.less", &Plugin{})
}

// Plugin registers the /less chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "less",
		Help:    "View the last assistant answer using less.",
		Aliases: []string{"l"},
		Execute: func(_ context.Context, a plugin.AgentHandle, args string) error {
			msgs := a.Messages()

			if len(msgs) < 2 {
				fmt.Fprintln(os.Stdout, "Nothing to view: no conversation history")
				return nil
			}

			lastMsg := msgs[len(msgs)-1]
			if lastMsg.Role != "assistant" {
				lastMsg = msgs[len(msgs)-2]
			}

			content := lastMsg.Content
			if content == "" {
				fmt.Fprintln(os.Stdout, "Nothing to view: last assistant message is empty")
				return nil
			}

			stdin := bytes.NewBuffer(nil)
			stdin.WriteString(content)

			cmd := exec.Command("less")
			cmd.Stdin = stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stdout, "Error running less: %v\n", err)
				return nil
			}

			return nil
		},
	})
	return nil
}
