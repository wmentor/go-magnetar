package lastresponse

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type mockHub struct {
	tools []plugin.LLMTool
}

func (h *mockHub) RegisterTool(tool plugin.LLMTool)             { h.tools = append(h.tools, tool) }
func (h *mockHub) RegisterChatCommand(plugin.ChatCommand)       {}
func (h *mockHub) RegisterPreprocessor(plugin.PreprocessorFunc) {}
func (h *mockHub) Go(func(ctx context.Context))                 {}
func (h *mockHub) Stop()                                        {}

func loadConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return cfg
}

func TestInitRegistersTool(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, "version: \"1.0\"\n")
	hub := &mockHub{}
	responseStore := response.New(plugin.DefaultResponseStoreSize)
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ResponseStore:    responseStore,
	}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Check that the tool was registered
	var names []string
	for _, tool := range state.ToolStore.Tools() {
		names = append(names, tool.Name)
	}
	assert.Contains(t, names, "last_response")
}

func TestToolDefinition(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, "version: \"1.0\"\n")
	responseStore := response.New(plugin.DefaultResponseStoreSize)
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ResponseStore:    responseStore,
	}
	hub := &mockHub{}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("last_response")
	require.NoError(t, err)

	// Check tool definition
	def := tool.Definition()
	assert.Equal(t, "last_response", def.Function.Name)

	// Check description mentions user explicitly
	desc := def.Function.Description
	assert.Contains(t, desc, "explicitly requested")
	assert.Contains(t, desc, "What was your last answer?")
}

func TestToolExecute(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, "version: \"1.0\"\n")
	responseStore := response.New(plugin.DefaultResponseStoreSize)
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ResponseStore:    responseStore,
	}
	hub := &mockHub{}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Push some responses
	responseStore.Push("First response")
	responseStore.Push("Second response")
	responseStore.Push("Third response")

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("last_response")
	require.NoError(t, err)

	// Test index 0 (most recent)
	result := tool.Execute(context.Background(), []byte(`{"index": 0}`))
	assert.Contains(t, result, "Third response")
	assert.Contains(t, result, "index 0")

	// Test index 1 (second most recent)
	result = tool.Execute(context.Background(), []byte(`{"index": 1}`))
	assert.Contains(t, result, "Second response")
	assert.Contains(t, result, "index 1")

	// Test index 2 (third most recent)
	result = tool.Execute(context.Background(), []byte(`{"index": 2}`))
	assert.Contains(t, result, "First response")
	assert.Contains(t, result, "index 2")
}

func TestToolExecuteInvalidIndex(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, "version: \"1.0\"\n")
	responseStore := response.New(plugin.DefaultResponseStoreSize)
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ResponseStore:    responseStore,
	}
	hub := &mockHub{}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("last_response")
	require.NoError(t, err)

	// Test negative index
	result := tool.Execute(context.Background(), []byte(`{"index": -1}`))
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "index must be non-negative")

	// Test index out of bounds (empty store)
	result = tool.Execute(context.Background(), []byte(`{"index": 0}`))
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "no response found at the specified index")
}

func TestToolExecuteInvalidJSON(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, "version: \"1.0\"\n")
	responseStore := response.New(plugin.DefaultResponseStoreSize)
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ResponseStore:    responseStore,
	}
	hub := &mockHub{}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("last_response")
	require.NoError(t, err)

	// Test invalid JSON
	result := tool.Execute(context.Background(), []byte("invalid json"))
	assert.Contains(t, result, "json_error")
}
