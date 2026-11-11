package fileplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/mask"
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

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`version: "1.0"`), 0o644))

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	maskService := mask.NewService(cfg)
	return &plugin.State{
		Config:    cfg,
		ToolStore: tools.NewStore(maskService),
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
	require.Len(t, registeredTools, 4, "expected exactly 4 tools to be registered")

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
			name:           "edit tool is sequential",
			toolName:       "edit",
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
