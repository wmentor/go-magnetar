package toolsplugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func init() {
	plugin.Register("chatcmd.tools", &Plugin{})
}

// Plugin registers the /tools chat command.
type Plugin struct{}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "tools",
		Aliases: []string{"t"},
		Help:    "List all available tools and their descriptions",
		Execute: func(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			return &plugin.ChatCommandResponse{
				Stdout: listTools(state.ToolStore),
			}, nil
		},
	})
	return nil
}

func listTools(toolStore *tools.Store) string {
	if toolStore == nil {
		return "Available tools:\n"
	}

	toolList := toolStore.Tools()
	if len(toolList) == 0 {
		return "Available tools:\n"
	}

	sort.Slice(toolList, func(i, j int) bool {
		return toolList[i].Name < toolList[j].Name
	})

	var sb strings.Builder
	sb.WriteString("Available tools:\n")

	for _, tool := range toolList {
		desc := ""
		if tool.Definition != nil {
			def := tool.Definition()
			if def.Function != nil {
				desc = def.Function.Description
			}
		}

		if desc != "" {
			fmt.Fprintf(&sb, "  * %s - %s\n", tool.Name, desc)
		} else {
			fmt.Fprintf(&sb, "  * %s\n", tool.Name)
		}
	}

	return sb.String()
}
