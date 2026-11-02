package exitplugin

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

type mockHub struct {
	commands []plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.commands = append(h.commands, cmd)
}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {}

func (h *mockHub) Go(f func(ctx context.Context)) {}

func (h *mockHub) Stop() {}

func TestInitRegistersExitCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"success", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Helper()

			hub := &mockHub{}
			state := &plugin.State{}

			err := (&Plugin{}).Init(state, hub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify command was registered
			assert.Len(t, hub.commands, 1)
			cmd := hub.commands[0]

			// Verify command name and aliases
			assert.Equal(t, "exit", cmd.Name)
			assert.Contains(t, cmd.Aliases, "quit")
			assert.Contains(t, cmd.Aliases, "q")
			assert.Equal(t, "End the session and exit.", cmd.Help)

			// Test command execution
			input := &plugin.ChatCommandInput{}
			resp, err := cmd.Execute(context.Background(), input)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Exit)
			assert.Equal(t, "Good bye!", resp.Stdout)
		})
	}
}

func TestCommandAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		checkName string
		wantExit  bool
	}{
		{"primary name", "exit", true},
		{"quit alias", "quit", true},
		{"q alias", "q", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Helper()

			hub := &mockHub{}
			state := &plugin.State{}
			require.NoError(t, (&Plugin{}).Init(state, hub))
			require.Len(t, hub.commands, 1)

			cmd := hub.commands[0]

			// Verify the command has the expected name or alias
			isName := cmd.Name == tt.checkName
			isAlias := slices.Contains(cmd.Aliases, tt.checkName)
			assert.True(t, isName || isAlias, "name or alias should be '%s'", tt.checkName)

			resp, err := cmd.Execute(context.Background(), &plugin.ChatCommandInput{})
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Exit)
		})
	}
}
