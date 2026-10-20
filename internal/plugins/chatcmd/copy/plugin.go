package copyplugin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/response"
)

// ErrInvalidIndex is returned when the /copy argument is not a non-negative integer.
var ErrInvalidIndex = errors.New("usage: /copy [index] — index must be a non-negative integer (0 = most recent response)")

func init() {
	plugin.Register("chatcmd.copy", &Plugin{})
}

var (
	writeAll func(text string) error = clipboard.WriteAll
)

// Plugin registers the /copy chat command.
type Plugin struct {
	responseStore plugin.ResponseStore
}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.responseStore = state.ResponseStore
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "copy",
		Aliases: []string{"c"},
		Help:    "Copy a response to clipboard. Optional index: 0=last (default), 1=previous, …",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			index, err := parseCopyIndex(params.Args)
			if err != nil {
				return stdoutResponse(err.Error()), nil
			}

			if p.responseStore == nil {
				return stdoutResponse("Nothing to copy: response store is unavailable"), nil
			}

			content, err := p.responseStore.Last(index)
			if errors.Is(err, response.ErrOutOfBounds) {
				return stdoutResponse(fmt.Sprintf("Nothing to copy: no response found at index %d", index)), nil
			}
			if err != nil {
				return nil, fmt.Errorf("read response history: %w", err)
			}
			if content == "" {
				return stdoutResponse("Nothing to copy: selected response is empty"), nil
			}

			if err := writeAll(content); err != nil {
				return stdoutResponse(fmt.Sprintf("Error copying to clipboard: %v", err)), nil
			}

			return stdoutResponse("Response copied to clipboard"), nil
		},
	})
	return nil
}

func stdoutResponse(message string) *plugin.ChatCommandResponse {
	return &plugin.ChatCommandResponse{Stdout: message + "\n"}
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

