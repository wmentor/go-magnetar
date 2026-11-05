package agentplugin

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// mockHub is a mock implementation of plugin.Hub interface
type mockHub struct {
	chatCommands  []plugin.ChatCommand
	preprocessors []plugin.PreprocessorFunc
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.chatCommands = append(m.chatCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	m.preprocessors = append(m.preprocessors, fn)
}

func (m *mockHub) Go(f func(ctx context.Context)) {
	// No-op for testing
}

func (m *mockHub) Stop() {
	// No-op for testing
}

func createTestState(t *testing.T) *plugin.State {
	t.Helper()

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	return &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(),
		RequesterFactory: clientFactory,
	}
}

func TestPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// Verify tool was registered
	tools := state.ToolStore.Tools()
	assert.GreaterOrEqual(t, len(tools), 1, "at least one tool should be registered")

	// Verify agent tool is registered
	_, err = state.ToolStore.GetTool("agent")
	assert.NoError(t, err, "agent tool should be registered")
}

func TestPlugin_InitToolDefinition(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent")
	require.NoError(t, err)

	// Check tool definition
	def := tool.Definition()
	assert.Equal(t, "agent", def.Function.Name)

	// Check description
	desc := def.Function.Description
	assert.Contains(t, desc, "Invoke")
	assert.Contains(t, desc, "subagent")
}

func TestPlugin_InitToolIsAgent(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent")
	require.NoError(t, err)

	// Check that IsAgent is set to true
	assert.True(t, tool.IsAgent, "agent tool should be marked as agent")
}

func TestPlugin_InitToolExecute(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent")
	require.NoError(t, err)

	// Test with nil params
	result := tool.Execute(context.Background(), nil)
	assert.Contains(t, result, "error")

	// Test with empty agent name
	result = tool.Execute(context.Background(), []byte(`{"agent": "", "message": "test"}`))
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "agent name is required")

	// Test with empty message
	result = tool.Execute(context.Background(), []byte(`{"agent": "default", "message": ""}`))
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "message is required")
}
