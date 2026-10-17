package helpplugin

import (
	"context"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

type commandHub struct {
	plugin.Hub
	command plugin.ChatCommand
}

func (h *commandHub) RegisterChatCommand(command plugin.ChatCommand) {
	h.command = command
}

func TestHelpResponse(t *testing.T) {
	t.Parallel()
	hub := &commandHub{}
	if err := (&Plugin{}).Init(nil, hub); err != nil {
		t.Fatal(err)
	}
	if hub.command.Name != "help" || len(hub.command.Aliases) != 1 || hub.command.Aliases[0] != "h" {
		t.Fatalf("unexpected help registration: %+v", hub.command)
	}
	response, err := hub.command.Execute(context.Background(), &plugin.ChatCommandInput{})
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || !strings.Contains(response.Stdout, "## Available chat commands\n") ||
		!strings.Contains(response.Stdout, "## Keyboard shortcuts\n") {
		t.Fatalf("help was not returned as markdown: %+v", response)
	}
	if response.Content != "" || response.Exit {
		t.Fatalf("help should only write to the user: %+v", response)
	}
}

func TestFormatHelp(t *testing.T) {
	t.Parallel()
	got := formatHelp([]plugin.ChatCommand{
		{Name: "zeta", Help: "Last command."},
		{Name: "alpha", Help: "First command.", Aliases: []string{"a", "first"}},
		{Name: "empty"},
	})
	want := "## Available chat commands\n\n" +
		"- `/alpha`: First command. (aliases: `/a`, `/first`)\n" +
		"- `/empty`: \n" +
		"- `/zeta`: Last command.\n" +
		"\n## Keyboard shortcuts\n\n- ↑/↓: navigate command history\n"
	if got != want {
		t.Fatalf("unexpected markdown:\ngot: %q\nwant: %q", got, want)
	}
}

func TestFormatHelpWithoutCommands(t *testing.T) {
	t.Parallel()
	got := formatHelp(nil)
	if strings.Contains(got, "`/") || !strings.Contains(got, "navigate command history") {
		t.Fatalf("unexpected help without commands: %q", got)
	}
}
