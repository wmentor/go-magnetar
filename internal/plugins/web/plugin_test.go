package webplugin

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

func TestInitRegistersWebToolsUnlessDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		tools []string
	}{
		{"not configured", "version: \"1.0\"\n", []string{"web_fetch", "web_search"}},
		{"enabled", "version: \"1.0\"\nweb:\n  disable:\n    fetch: false\n    search: false\n", []string{"web_fetch", "web_search"}},
		{"fetch disabled", "version: \"1.0\"\nweb:\n  disable:\n    fetch: true\n", []string{"web_search"}},
		{"search disabled", "version: \"1.0\"\nweb:\n  disable:\n    search: true\n", []string{"web_fetch"}},
		{"both disabled", "version: \"1.0\"\nweb:\n  disable:\n    fetch: true\n    search: true\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadConfig(t, tt.yaml)
			hub := &mockHub{}
			state := &plugin.State{
				Config:           cfg,
				ToolStore:        tools.NewStore(),
				RequesterFactory: requester.NewFactory(),
				Printer:          printer.GetDefault(),
			}
			err := (&Plugin{}).Init(state, hub)
			require.NoError(t, err)
			var names []string
			for _, tool := range state.ToolStore.Tools() {
				names = append(names, tool.Name)
			}
			assert.Equal(t, tt.tools, names)
		})
	}
}
