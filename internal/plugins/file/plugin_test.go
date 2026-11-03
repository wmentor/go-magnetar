package fileplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// mockHub is a minimal no-op implementation of the plugin.Hub interface.
type mockHub struct{}

func (h *mockHub) RegisterChatCommand(plugin.ChatCommand)       {}
func (h *mockHub) RegisterPreprocessor(plugin.PreprocessorFunc) {}
func (h *mockHub) Go(func(ctx context.Context))                 {}
func (h *mockHub) Stop()                                        {}

// newTestState creates a minimal plugin.State required for Plugin.Init.
func newTestState(t *testing.T) *plugin.State {
	t.Helper()

	return &plugin.State{
		ToolStore: tools.NewStore(),
	}
}

func TestPlugin_Init(t *testing.T) {
	t.Parallel()

	state := newTestState(t)
	hub := &mockHub{}
	p := &Plugin{}

	err := p.Init(state, hub)
	require.NoError(t, err)

	assert.Same(t, state, p.state, "plugin state must match the injected state")
	assert.NotNil(t, p.tool, "plugin read tool must be initialized")
}

func TestPlugin_ToolRegistration(t *testing.T) {
	t.Parallel()

	state := newTestState(t)
	hub := &mockHub{}
	p := &Plugin{}

	err := p.Init(state, hub)
	require.NoError(t, err)

	registeredTools := state.ToolStore.Tools()
	require.Len(t, registeredTools, 5, "expected exactly 5 tools to be registered")

	tests := []struct {
		name           string
		toolName       string
		wantSequential bool
	}{
		{
			name:           "file_read tool",
			toolName:       "file_read",
			wantSequential: false,
		},
		{
			name:           "file_list tool",
			toolName:       "file_list",
			wantSequential: false,
		},
		{
			name:           "file_write tool is sequential",
			toolName:       "file_write",
			wantSequential: true,
		},
		{
			name:           "file_edit tool",
			toolName:       "file_edit",
			wantSequential: false,
		},
		{
			name:           "file_edit_multiple tool is sequential",
			toolName:       "file_edit_multiple",
			wantSequential: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tool, err := state.ToolStore.GetTool(tt.toolName)
			require.NoError(t, err, "tool %q must be registered in ToolStore", tt.toolName)
			assert.Equal(t, tt.toolName, tool.Name)
			assert.NotNil(t, tool.Execute, "tool %q must have an Execute handler", tt.toolName)
			assert.Equal(t, tt.wantSequential, tool.Sequential, "tool %q sequential option mismatch", tt.toolName)
		})
	}
}
