package helpplugin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.help", &Plugin{})
}

// Plugin registers the /help chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "help",
		Aliases: []string{"h"},
		Help:    "Show this help message.",
		Execute: func(_ context.Context, _ plugin.AgentHandle, _ string) (*plugin.ChatCommandResponse, error) {
			printHelp()
			return plugin.NopChatCommandResponse, nil
		},
	})
	return nil
}

func printHelp() {
	cmds := plugin.ChatCommands()

	// Sort alphabetically for stable output.
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})

	var sb strings.Builder
	sb.WriteString("Available chat commands:\n")
	for _, c := range cmds {
		fmt.Fprintf(&sb, "    /%-12s %s", c.Name, c.Help)
		if len(c.Aliases) > 0 {
			aliases := make([]string, len(c.Aliases))
			for i, a := range c.Aliases {
				aliases[i] = "/" + a
			}
			fmt.Fprintf(&sb, " (aliases: %s)", strings.Join(aliases, ", "))
		}
		fmt.Fprintln(&sb, "")
	}
	sb.WriteString("\nKeyboard shortcuts:\n")
	sb.WriteString("    ↑/↓          navigate command history\n")
	fmt.Fprint(os.Stdout, sb.String()+"\n")
}
