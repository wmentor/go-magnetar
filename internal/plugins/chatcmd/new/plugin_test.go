package newplugin

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

type mockAgent struct {
	resetCalled bool
}

func (a *mockAgent) Messages() []openai.ChatCompletionMessage {
	return nil
}

func (a *mockAgent) SetMessages(msgs []openai.ChatCompletionMessage) {
}

func (a *mockAgent) Config() *config.Config {
	return nil
}

func (a *mockAgent) Compact(_ ...string) error {
	return nil
}

func (a *mockAgent) Reset() {
	a.resetCalled = true
}

func (a *mockAgent) Reconfigure() {
}

type mockHub struct {
	commands []plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.commands = append(h.commands, cmd)
}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
}

func (h *mockHub) Go(f func(context.Context)) {
}

func (h *mockHub) Stop() {
}

func TestInitRegistersNewCommand(t *testing.T) {
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

			// Verify command name and help text
			assert.Equal(t, "new", cmd.Name)
			assert.Len(t, cmd.Aliases, 0)
			assert.Equal(t, "Start a new session and clear conversation history.", cmd.Help)

			// Test command execution
			agent := &mockAgent{}
			input := &plugin.ChatCommandInput{
				AgentHandler: agent,
			}
			resp, err := cmd.Execute(context.Background(), input)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, agent.resetCalled)
			assert.Equal(t, plugin.NopChatCommandResponse, resp)
		})
	}
}

func TestNewCommandResetsAgentHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"command resets agent history"},
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
			agent := &mockAgent{}
			input := &plugin.ChatCommandInput{
				AgentHandler: agent,
			}

			// Verify reset is called
			resp, err := cmd.Execute(context.Background(), input)
			require.NoError(t, err)
			assert.True(t, agent.resetCalled)
			assert.Equal(t, plugin.NopChatCommandResponse, resp)
		})
	}
}
