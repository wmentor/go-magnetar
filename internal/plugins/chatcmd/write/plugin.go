package writeplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.write", &Plugin{})
}

// Plugin registers the /save chat command.
type Plugin struct{}

// Message is the conversation turn the write command reads.
type Message struct {
	Role    string
	Content string
}

// Result is the markdown diagnostic returned to the chat.
type Result struct {
	Stdout string
}

// RunWrite picks the last assistant answer and writes it to filename.
// Diagnostics are returned as markdown in Result.Stdout, never printed to os.Stdout.
func RunWrite(filename string, msgs []Message) Result {
	if filename == "" {
		return Result{Stdout: "Usage: `/write <filename>`"}
	}
	if len(msgs) < 2 {
		return Result{Stdout: "Nothing to save: no conversation history"}
	}
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "assistant" {
		lastMsg = msgs[len(msgs)-2]
	}
	content := lastMsg.Content
	if content == "" {
		return Result{Stdout: "Nothing to save: last assistant message is empty"}
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return Result{Stdout: fmt.Sprintf("Error saving file: `%s`\n\n```\n%v\n```", filename, err)}
	}
	return Result{
		Stdout: fmt.Sprintf("✅ **Answer saved**\n\n- File: `%s`\n- Content size: %d bytes\n", filename, len(content)),
	}
}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "write",
		Help:    "Save the last assistant answer to a file.",
		Aliases: []string{"w"},
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			filename := filepath.Clean(common.ExpandHome(params.Args))
			raw := params.AgentHandler.Messages()
			msgs := make([]Message, 0, len(raw))
			for _, m := range raw {
				msgs = append(msgs, Message{Role: m.Role, Content: m.Content})
			}
			return &plugin.ChatCommandResponse{Stdout: RunWrite(filename, msgs).Stdout}, nil
		},
	})
	return nil
}
