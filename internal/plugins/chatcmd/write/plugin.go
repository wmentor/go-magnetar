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

// expandHome resolves "~" in user-supplied paths. It is a variable so tests
// can redirect expansion into a temporary directory.
var expandHome = common.ExpandHome

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
		Execute: p.execute,
	})
	return nil
}

func (p *Plugin) execute(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := params.Args
	if args == "" {
		return &plugin.ChatCommandResponse{Stdout: "Usage: `/write <filename>`\n"}, nil
	}

	filename := filepath.Clean(expandHome(args))
	msgs := params.AgentHandler.Messages()

	content, reason := selectAnswer(msgs)
	if content == "" {
		return &plugin.ChatCommandResponse{Stdout: formatEmptyDiagnostic(reason)}, nil
	}

	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return &plugin.ChatCommandResponse{Stdout: formatErrorDiagnostic(filename, err)}, nil
	}

	return &plugin.ChatCommandResponse{Stdout: formatSuccessDiagnostic(filename, len(content))}, nil
}

// selectAnswer picks the content to save from the conversation history. When
// there is nothing to save it returns an empty content string plus the reason.
func selectAnswer(msgs []openai.ChatCompletionMessage) (string, string) {
	if len(msgs) < 2 {
		return "", "no conversation history"
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != openai.ChatMessageRoleAssistant {
		lastMsg = msgs[len(msgs)-2]
	}

	if lastMsg.Content == "" {
		return "", "last assistant message is empty"
	}

	return lastMsg.Content, ""
}

// formatSuccessDiagnostic renders the save confirmation as markdown so the
// chat agent's standard rendering pipeline displays it like other commands.
func formatSuccessDiagnostic(filename string, size int) string {
	return fmt.Sprintf("## Write\n\nSaved %d bytes to `%s`\n", size, filename)
}

// formatEmptyDiagnostic reports that there was nothing to save.
func formatEmptyDiagnostic(reason string) string {
	return fmt.Sprintf("⚠️ Nothing to save: %s\n", reason)
}

// formatErrorDiagnostic reports a failed write.
func formatErrorDiagnostic(filename string, err error) string {
	return fmt.Sprintf("## Write\n\n❌ Failed to save to `%s`: %v\n", filename, err)
}
