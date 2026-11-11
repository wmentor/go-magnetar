package gitlabplug

import (
	"context"
	"os"
	"path/filepath"
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

func TestPluginInit(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `version: "1.0"
gitlab:
  enable: false
  base_url: https://gitlab.example.com
  api_key: test-key
`
	if err := os.WriteFile(path, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	require.NoError(t, err)

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock hub
	hub := &mockHub{}

	// Create plugin with properly initialized State
	pluginInstance := &Plugin{
		state: &plugin.State{
			Config:           cfg,
			Printer:          p,
			ToolStore:        toolStore,
			RequesterFactory: clientFactory,
		},
	}

	err = pluginInstance.Init(pluginInstance.state, hub)
	assert.NoError(t, err)
}

func TestPluginInitWithDisabledGitLab(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `version: "1.0"
gitlab:
  enable: false
  base_url: https://gitlab.example.com
  api_key: test-key
`
	if err := os.WriteFile(path, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	require.NoError(t, err)

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	hub := &mockHub{}

	pluginInstance := &Plugin{
		state: &plugin.State{
			Config:           cfg,
			Printer:          p,
			ToolStore:        toolStore,
			RequesterFactory: clientFactory,
		},
	}

	err = pluginInstance.Init(pluginInstance.state, hub)
	assert.NoError(t, err)
}

func TestPluginInitWithEmptyBaseURL(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `version: "1.0"
gitlab:
  enable: true
  base_url: ""
  api_key: test-key
`
	if err := os.WriteFile(path, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	require.NoError(t, err)

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	hub := &mockHub{}

	pluginInstance := &Plugin{
		state: &plugin.State{
			Config:           cfg,
			Printer:          p,
			ToolStore:        toolStore,
			RequesterFactory: clientFactory,
		},
	}

	err = pluginInstance.Init(pluginInstance.state, hub)
	assert.NoError(t, err)
}

func TestPluginInitWithEnabledGitLab(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `version: "1.0"
gitlab:
  enable: true
  base_url: https://gitlab.example.com
  api_key: test-key
`
	if err := os.WriteFile(path, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	require.NoError(t, err)

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	hub := &mockHub{}

	pluginInstance := &Plugin{
		state: &plugin.State{
			Config:           cfg,
			Printer:          p,
			ToolStore:        toolStore,
			RequesterFactory: clientFactory,
		},
	}

	err = pluginInstance.Init(pluginInstance.state, hub)
	assert.NoError(t, err)

	// When gitlab.enable is true and gitlab.base_url is set,
	// the Init method should register the tools with the tool store.
	// Verify that tools were registered.
	tools := toolStore.Tools()
	assert.Equal(t, 4, len(tools), "expected 4 tools to be registered")
}

type mockHub struct{}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	// No-op for testing
}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	// No-op for testing
}

func (h *mockHub) Go(f func(ctx context.Context)) {
	// No-op for testing
}

func (h *mockHub) Stop() {
	// No-op for testing
}
