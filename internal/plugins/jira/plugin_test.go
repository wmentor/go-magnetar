package jiraplug

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/mask"
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

func createTestState(t *testing.T, configYaml string, enableJira bool) *plugin.State {
	t.Helper()

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	if err := writeConfig(configPath, configYaml, enableJira); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfg)

	return &plugin.State{
		Config:           cfg,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}
}

func writeConfig(path string, configYaml string, enableJira bool) error {
	// Build complete config
	cfg := `version: "1.0"`
	if enableJira {
		cfg += `
jira:
  enable: true
  base_url: "https://jira.example.com"
  api_key: "test-key"
  readonly: false
`
	}
	if configYaml != "" {
		cfg += configYaml
	}

	return os.WriteFile(path, []byte(cfg), 0o644)
}

func TestPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	// Create test state with jira enabled
	state := createTestState(t, "", true)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// Verify tools were registered
	tools := state.ToolStore.Tools()
	assert.GreaterOrEqual(t, len(tools), 1, "at least one tool should be registered")

	// Verify jira_task_get tool is registered
	_, err = state.ToolStore.GetTool("jira_task_get")
	assert.NoError(t, err, "jira_task_get tool should be registered")

	// Verify jira_task_search tool is registered
	_, err = state.ToolStore.GetTool("jira_task_search")
	assert.NoError(t, err, "jira_task_search tool should be registered")

	// Verify jira_issue_create tool is registered (not in readonly mode)
	_, err = state.ToolStore.GetTool("jira_issue_create")
	assert.NoError(t, err, "jira_issue_create tool should be registered")
}

func TestPlugin_Init_Disabled(t *testing.T) {
	t.Parallel()

	// Create test state with jira disabled
	state := createTestState(t, "", false)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// When disabled, no tools should be registered
	tools := state.ToolStore.Tools()
	assert.Equal(t, 0, len(tools), "no tools should be registered when jira is disabled")
}

func TestPlugin_Init_MissingJiraSection(t *testing.T) {
	t.Parallel()

	// Create minimal config without jira section
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfgLoaded)

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}
	hub := &mockHub{}

	// Create plugin and init
	pPlugin := &Plugin{}
	err = pPlugin.Init(state, hub)
	require.NoError(t, err)

	// When jira section is missing, no tools should be registered
	tools := state.ToolStore.Tools()
	assert.Equal(t, 0, len(tools), "no tools should be registered when jira config is missing")
}

func TestPlugin_Init_MissingAPIKey(t *testing.T) {
	t.Parallel()

	// Create config with jira enabled but missing api_key
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  enable: true
  base_url: "https://jira.example.com"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfgLoaded)

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}
	hub := &mockHub{}

	// Create plugin and init
	pPlugin := &Plugin{}
	err = pPlugin.Init(state, hub)
	require.NoError(t, err)

	// Tools should be registered but will fail on execution without API key
	tools := state.ToolStore.Tools()
	assert.GreaterOrEqual(t, len(tools), 1, "tools should be registered even without API key")
}

func TestPlugin_Init_ReadonlyMode(t *testing.T) {
	t.Parallel()

	// Create config with jira enabled in readonly mode
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  enable: true
  base_url: "https://jira.example.com"
  api_key: "test-key"
  readonly: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfgLoaded)

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}
	hub := &mockHub{}

	// Create plugin and init
	pPlugin := &Plugin{}
	err = pPlugin.Init(state, hub)
	require.NoError(t, err)

	// Verify tools were registered
	tools := state.ToolStore.Tools()
	assert.GreaterOrEqual(t, len(tools), 1, "at least one tool should be registered")

	// Verify jira_task_get tool is registered
	_, err = state.ToolStore.GetTool("jira_task_get")
	assert.NoError(t, err, "jira_task_get tool should be registered")

	// Verify jira_task_search tool is registered
	_, err = state.ToolStore.GetTool("jira_task_search")
	assert.NoError(t, err, "jira_task_search tool should be registered")

	// Verify jira_issue_create tool is NOT registered in readonly mode
	_, err = state.ToolStore.GetTool("jira_issue_create")
	assert.Error(t, err, "jira_issue_create tool should NOT be registered in readonly mode")
}
