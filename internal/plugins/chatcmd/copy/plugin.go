package copyplugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

// ErrInvalidIndex is returned when the /copy argument is not a non-negative integer.
var ErrInvalidIndex = errors.New("usage: /copy [index] — index must be a non-negative integer (0 = last answer)")

func init() {
	plugin.Register("chatcmd.copy", &Plugin{})
}

var (
	out      io.Writer               = os.Stdout
	writeAll func(text string) error = clipboard.WriteAll
)

// Plugin registers the /copy chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "copy",
		Aliases: []string{"c"},
		Help:    "Copy an assistant answer to clipboard. Optional index: 0=last (default), 1=previous, …",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			index, err := parseCopyIndex(params.Args)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				return plugin.NopChatCommandResponse, nil
			}

			answers := assistantAnswers(params.AgentHandler.Messages())
			if len(answers) == 0 {
				fmt.Fprintln(out, "Nothing to copy: no conversation history")
				return plugin.NopChatCommandResponse, nil
			}
			if index >= len(answers) {
				fmt.Fprintf(out, "Nothing to copy: only %d assistant answer(s) in history\n", len(answers))
				return plugin.NopChatCommandResponse, nil
			}

			content := answers[index]
			if content == "" {
				fmt.Fprintln(out, "Nothing to copy: selected assistant message is empty")
				return plugin.NopChatCommandResponse, nil
			}

			if err := writeAll(content); err != nil {
				fmt.Fprintf(out, "Error copying to clipboard: %v\n", err)
				return plugin.NopChatCommandResponse, nil
			}

			fmt.Fprintln(out, "Answer copied to clipboard")
			return plugin.NopChatCommandResponse, nil
		},
	})
	return nil
}

// assistantAnswers returns the contents of assistant answers shown to the user,
// newest-first (index 0 = last answer). Assistant messages that carry tool calls
// are intermediate steps, not answers, so they are skipped.
func assistantAnswers(msgs []openai.ChatCompletionMessage) []string {
	var out []string
	for _, msg := range slices.Backward(msgs) {
		if msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) == 0 {
			out = append(out, msg.Content)
		}
	}
	return out
}

func parseCopyIndex(args string) (int, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(args)
	if err != nil || n < 0 {
		return 0, ErrInvalidIndex
	}
	return n, nil
}
