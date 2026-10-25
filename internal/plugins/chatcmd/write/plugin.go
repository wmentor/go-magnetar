package writeplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.write", &Plugin{})
}

// Plugin registers the /write chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "write",
		Help:    "Save the last assistant answer to a file.",
		Aliases: []string{"w"},
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			return execute(params.AgentHandler.Messages(), params.Args), nil
		},
	})
	return nil
}

// execute saves the last assistant message to the file named by args and
// returns the diagnostics as markdown in ChatCommandResponse.Stdout, so the
// caller decides where they go (and tests can read them).
func execute(msgs []openai.ChatCompletionMessage, args string) *plugin.ChatCommandResponse {
	if args == "" {
		return &plugin.ChatCommandResponse{Stdout: "Usage: `/write <filename>`"}
	}

	if len(msgs) < 2 {
		return &plugin.ChatCommandResponse{Stdout: "Nothing to save: no conversation history"}
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "assistant" {
		lastMsg = msgs[len(msgs)-2]
	}

	if lastMsg.Content == "" {
		return &plugin.ChatCommandResponse{Stdout: "Nothing to save: last assistant message is empty"}
	}

	filename := filepath.Clean(common.ExpandHome(args))
	if err := os.WriteFile(filename, []byte(lastMsg.Content), 0o644); err != nil {
		return &plugin.ChatCommandResponse{Stdout: fmt.Sprintf("**Error saving file:** %v", err)}
	}

	return &plugin.ChatCommandResponse{Stdout: fmt.Sprintf("Answer saved to `%s`", filename)}
}
