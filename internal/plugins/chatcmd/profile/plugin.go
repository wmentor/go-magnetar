package profileplugin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.profile", &Plugin{})
}

// Plugin registers the /profile chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "profile",
		Help: "Show current profile or switch to a different profile.",
		Execute: func(ctx context.Context, a plugin.AgentHandle, args string) (*plugin.ChatCommandResponse, error) {
			if args == "" {
				printCurrentProfile(a.Config(), a.Messages())
				return plugin.NopChatCommandResponse, nil
			}

			if err := a.Config().SetProfile(args); err != nil {
				return nil, errors.Wrap(err, "set profile")
			}

			a.Reconfigure()

			fmt.Printf("Switched to profile: %s\n", args)
			return plugin.NopChatCommandResponse, nil
		},
	})
	return nil
}

func printCurrentProfile(cfg *config.Config, messages []openai.ChatCompletionMessage) {
	profiles := cfg.Profiles()
	current := cfg.String("profile")
	model := cfg.ProfileParamString("llm.model")

	var sb strings.Builder
	fmt.Fprintf(&sb, "Available profiles:\n")
	for _, p := range profiles {
		prefix := "  "
		if p == current {
			prefix = "*>"
		}
		fmt.Fprintf(&sb, "%s %s\n", prefix, p)
	}

	fmt.Fprintf(&sb, "\nCurrent profile: %s\n", current)
	fmt.Fprintf(&sb, "LLM model: %s\n", model)
	fmt.Fprintf(&sb, "Total messages: %d\n", len(messages))

	fmt.Fprint(os.Stdout, sb.String())
}
