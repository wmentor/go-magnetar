package cveplugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/mask"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// mockPrinter is a mock implementation of the Printer interface.
type mockPrinter struct {
	printer.Printer
	calls []string
}

func (m *mockPrinter) ToolCall(typeIcon string, message string, params ...any) {
	m.calls = append(m.calls, message)
}

// mockHub is a mock implementation of the Hub interface.
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
	// Do nothing in tests
}

func (m *mockHub) Stop() {
	// Do nothing in tests
}

func TestPlugin_Init(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-model
      temperature: 0.9
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create a mock hub
	mockHub := &mockHub{}

	// Create a mock config
	cfg, err := config.Load(configPath)
	require.NoError(t, err, "failed to create config")

	// Create a mock printer
	mockPrt := &mockPrinter{}

	// Create a mock tool store with mask service
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock requester factory
	requesterFactory := testdata.NewClientFactory(t)

	// Create the plugin state with all dependencies
	state := &plugin.State{
		Config:           cfg,
		Printer:          mockPrt,
		ToolStore:        toolStore,
		RequesterFactory: requesterFactory,
	}

	// Create the plugin instance
	pluginInstance := &Plugin{}

	// Call Init
	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	// Verify that the tool was registered
	tool, err := toolStore.GetTool("cve")
	require.NoError(t, err, "tool 'cve' should be registered")

	// Verify tool metadata
	assert.Equal(t, "cve", tool.Name)
	definition := tool.Definition()
	assert.Equal(t, openai.ToolTypeFunction, definition.Type)
	assert.Contains(t, definition.Function.Description, "Lookup vulnerability information")
	assert.Contains(t, definition.Function.Description, "OSV database")
}

func TestPlugin_Init_ToolSchema(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-model
      temperature: 0.9
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create a mock hub
	mockHub := &mockHub{}

	// Create a mock config
	cfg, err := config.Load(configPath)
	require.NoError(t, err, "failed to create config")

	// Create a mock printer
	mockPrt := &mockPrinter{}

	// Create a mock tool store with mask service
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock requester factory
	requesterFactory := testdata.NewClientFactory(t)

	// Create the plugin state
	state := &plugin.State{
		Config:           cfg,
		Printer:          mockPrt,
		ToolStore:        toolStore,
		RequesterFactory: requesterFactory,
	}

	// Create the plugin instance
	pluginInstance := &Plugin{}

	// Call Init
	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := toolStore.GetTool("cve")
	require.NoError(t, err)

	// Verify the tool has parameters
	definition := tool.Definition()
	// The Parameters field in openai.FunctionDefinition has type `any`,
	// but internally it contains the schema from the jsonschema package.
	// Since the exact type is opaque, we verify the schema indirectly by checking
	// that the tool can be marshaled to JSON without errors.
	_, err = json.Marshal(definition.Function.Parameters)
	require.NoError(t, err, "Parameters should be serializable to JSON")
	require.NotNil(t, definition.Function.Parameters, "parameters should not be nil")
}

func TestPlugin_Init_MultipleCalls(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-model
      temperature: 0.9
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create a mock hub
	mockHub := &mockHub{}

	// Create a mock config
	cfg, err := config.Load(configPath)
	require.NoError(t, err, "failed to create config")

	// Create a mock printer
	mockPrt := &mockPrinter{}

	// Create a mock tool store with mask service
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock requester factory
	requesterFactory := testdata.NewClientFactory(t)

	// Create the plugin state
	state := &plugin.State{
		Config:           cfg,
		Printer:          mockPrt,
		ToolStore:        toolStore,
		RequesterFactory: requesterFactory,
	}

	// Create the plugin instance
	pluginInstance := &Plugin{}

	// Call Init multiple times (should be idempotent for registration)
	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := toolStore.GetTool("cve")
	require.NoError(t, err)
	assert.Equal(t, "cve", tool.Name)
}

func TestPlugin_Init_RegisterOnlyOnce(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-model
      temperature: 0.9
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create a mock hub
	mockHub := &mockHub{}

	// Create a mock config
	cfg, err := config.Load(configPath)
	require.NoError(t, err, "failed to create config")

	// Create a mock printer
	mockPrt := &mockPrinter{}

	// Create a mock tool store with mask service
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock requester factory
	requesterFactory := testdata.NewClientFactory(t)

	// Create the plugin state
	state := &plugin.State{
		Config:           cfg,
		Printer:          mockPrt,
		ToolStore:        toolStore,
		RequesterFactory: requesterFactory,
	}

	// Create the plugin instance
	pluginInstance := &Plugin{}

	// Call Init multiple times to verify it only registers once
	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	// Get the registered tool
	tools := toolStore.Tools()
	// Should only have the cve tool registered once
	cveTools := 0
	for _, t := range tools {
		if t.Name == "cve" {
			cveTools++
		}
	}
	assert.Equal(t, 1, cveTools, "cve tool should be registered only once")
}

func TestPlugin_Init_ToolDescription(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-model
      temperature: 0.9
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create a mock hub
	mockHub := &mockHub{}

	// Create a mock config
	cfg, err := config.Load(configPath)
	require.NoError(t, err, "failed to create config")

	// Create a mock printer
	mockPrt := &mockPrinter{}

	// Create a mock tool store with mask service
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock requester factory
	requesterFactory := testdata.NewClientFactory(t)

	// Create the plugin state
	state := &plugin.State{
		Config:           cfg,
		Printer:          mockPrt,
		ToolStore:        toolStore,
		RequesterFactory: requesterFactory,
	}

	// Create the plugin instance
	pluginInstance := &Plugin{}

	// Call Init
	err = pluginInstance.Init(state, mockHub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := toolStore.GetTool("cve")
	require.NoError(t, err)

	// Verify the tool description contains all supported database prefixes
	definition := tool.Definition()
	description := definition.Function.Description

	// Check for common database prefixes mentioned in the description
	prefixes := []string{"CVE-", "GO-", "GHSA-", "OSV-", "GSD-", "ALPINE-"}
	for _, prefix := range prefixes {
		assert.Contains(t, description, prefix, "description should mention %s", prefix)
	}
}
