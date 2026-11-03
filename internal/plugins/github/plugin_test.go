package githubplug

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

func createTestState(t *testing.T, configYaml string, enableGithub bool) *plugin.State {
	t.Helper()

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	if err := writeConfig(configPath, configYaml, enableGithub); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	return &plugin.State{
		Config:           cfg,
		Printer:          p,
		ToolStore:        tools.NewStore(),
		RequesterFactory: clientFactory,
	}
}

func writeConfig(path string, configYaml string, enableGithub bool) error {
	// Build complete config
	cfg := `version: "1.0"`
	if enableGithub {
		cfg += `
github:
  enable: true
  base_url: "https://api.github.com"
  api_key: "test-key"
`
	}
	if configYaml != "" {
		cfg += configYaml
	}

	return os.WriteFile(path, []byte(cfg), 0o644)
}

func TestPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	// Create test state with github enabled
	state := createTestState(t, "", true)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// Verify tools were registered
	tools := state.ToolStore.Tools()
	assert.GreaterOrEqual(t, len(tools), 1, "at least one tool should be registered")

	// Verify github_repo tool is registered
	_, err = state.ToolStore.GetTool("github_repo")
	assert.NoError(t, err, "github_repo tool should be registered")

	// Verify github_file tool is registered
	_, err = state.ToolStore.GetTool("github_file")
	assert.NoError(t, err, "github_file tool should be registered")

	// Verify github_tree tool is registered
	_, err = state.ToolStore.GetTool("github_tree")
	assert.NoError(t, err, "github_tree tool should be registered")

	// Verify github_issue tool is registered
	_, err = state.ToolStore.GetTool("github_issue")
	assert.NoError(t, err, "github_issue tool should be registered")

	// Verify github_milestone tool is registered
	_, err = state.ToolStore.GetTool("github_milestone")
	assert.NoError(t, err, "github_milestone tool should be registered")

	// Verify github_pull_request tool is registered
	_, err = state.ToolStore.GetTool("github_pull_request")
	assert.NoError(t, err, "github_pull_request tool should be registered")

	// Verify github_release_info tool is registered
	_, err = state.ToolStore.GetTool("github_release_info")
	assert.NoError(t, err, "github_release_info tool should be registered")
}

func TestPlugin_Init_Disabled(t *testing.T) {
	t.Parallel()

	// Create test state with github disabled
	state := createTestState(t, "", false)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// When disabled, no tools should be registered
	tools := state.ToolStore.Tools()
	assert.Equal(t, 0, len(tools), "no tools should be registered when github is disabled")
}

func TestPlugin_Init_MissingGitHubSection(t *testing.T) {
	t.Parallel()

	// Create minimal config without github section
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(),
		RequesterFactory: clientFactory,
	}
	hub := &mockHub{}

	// Create plugin and init
	pPlugin := &Plugin{}
	err = pPlugin.Init(state, hub)
	require.NoError(t, err)

	// When github section is missing, no tools should be registered
	tools := state.ToolStore.Tools()
	assert.Equal(t, 0, len(tools), "no tools should be registered when github config is missing")
}

func TestPlugin_Init_MissingAPIKey(t *testing.T) {
	t.Parallel()

	// Create config with github enabled but missing api_key
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
github:
  enable: true
  base_url: "https://api.github.com"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(),
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
