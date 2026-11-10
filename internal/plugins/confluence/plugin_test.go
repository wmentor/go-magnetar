package confluenceplugin

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

type mockHub struct{}

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

func TestInitRegistersConfluenceToolsUnlessDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		tools       []string
		errExpected bool
	}{
		{
			name:  "not configured (disabled by default)",
			yaml:  `version: "1.0"`,
			tools: nil,
		},
		{
			name: "explicitly disabled",
			yaml: `version: "1.0"
confluence:
  enable: false
  base_url: https://example.atlassian.net
`,
			tools: nil,
		},
		{
			name: "enabled without base_url (should not register)",
			yaml: `version: "1.0"
confluence:
  enable: true
`,
			tools: nil,
		},
		{
			name: "enabled with base_url only (fetch, edit and search tools by default)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
`,
			tools: []string{"confluence_page_fetch", "confluence_page_edit", "confluence_search"},
		},
		{
			name: "enabled with readonly=false (fetch, edit and search tools)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  readonly: false
`,
			tools: []string{"confluence_page_fetch", "confluence_page_edit", "confluence_search"},
		},
		{
			name: "enabled with readonly=true (fetch and search tools)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  readonly: true
`,
			tools: []string{"confluence_page_fetch", "confluence_search"},
		},
		{
			name: "enabled with api_key and readonly=true (fetch and search tools)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  api_key: test-key
  readonly: true
`,
			tools: []string{"confluence_page_fetch", "confluence_search"},
		},
		{
			name: "enabled with api_key and readonly=false (all tools)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  api_key: test-key
  readonly: false
`,
			tools: []string{"confluence_page_fetch", "confluence_page_edit", "confluence_search"},
		},
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
			if tt.errExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var names []string
			for _, tool := range state.ToolStore.Tools() {
				names = append(names, tool.Name)
			}
			assert.ElementsMatch(t, tt.tools, names)
		})
	}
}

func TestInitToolDescriptions(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  readonly: false
`)
	hub := &mockHub{}
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
	}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Check fetch tool
	fetchTool, err := state.ToolStore.GetTool("confluence_page_fetch")
	require.NoError(t, err)
	fetchDef := fetchTool.Definition()
	assert.Equal(t, "confluence_page_fetch", fetchDef.Function.Name)
	assert.Contains(t, fetchDef.Function.Description, "Fetch a Confluence page by ID")
	assert.Contains(t, fetchDef.Function.Description, "Markdown")

	// Check edit tool is registered when readonly=false
	editTool, err := state.ToolStore.GetTool("confluence_page_edit")
	require.NoError(t, err)
	editDef := editTool.Definition()
	assert.Equal(t, "confluence_page_edit", editDef.Function.Name)
	assert.Contains(t, editDef.Function.Description, "Edit a Confluence page")
	assert.Contains(t, editDef.Function.Description, "Markdown")

	// Check search tool
	searchTool, err := state.ToolStore.GetTool("confluence_search")
	require.NoError(t, err)
	searchDef := searchTool.Definition()
	assert.Equal(t, "confluence_search", searchDef.Function.Name)
	assert.Contains(t, searchDef.Function.Description, "Confluence")
	assert.Contains(t, searchDef.Function.Description, "search")
}

func TestInitToolParametersExist(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  readonly: false
`)
	hub := &mockHub{}
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
	}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Check fetch tool parameters exist
	fetchTool, err := state.ToolStore.GetTool("confluence_page_fetch")
	require.NoError(t, err)
	fetchDef := fetchTool.Definition()
	require.NotNil(t, fetchDef.Function.Parameters)

	// Check edit tool parameters exist
	editTool, err := state.ToolStore.GetTool("confluence_page_edit")
	require.NoError(t, err)
	editDef := editTool.Definition()
	require.NotNil(t, editDef.Function.Parameters)

	// Check search tool parameters exist
	searchTool, err := state.ToolStore.GetTool("confluence_search")
	require.NoError(t, err)
	searchDef := searchTool.Definition()
	require.NotNil(t, searchDef.Function.Parameters)
}

func TestInitWithGlobalReadOnlyFlag(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, `version: "1.0"
confluence:
  enable: true
  base_url: https://example.atlassian.net
  readonly: true
`)
	hub := &mockHub{}
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
		ReadOnly:         false, // global flag is false
	}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Edit tool should still be disabled due to config
	_, err = state.ToolStore.GetTool("confluence_page_edit")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")

	// Verify fetch tool is registered
	fetchTool, err := state.ToolStore.GetTool("confluence_page_fetch")
	require.NoError(t, err)
	assert.NotNil(t, fetchTool)
}

func TestInitToolCallCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedTools int
		expectedEdit  bool
	}{
		{
			name:          "disabled - no tools",
			yaml:          `version: "1.0"`,
			expectedTools: 0,
			expectedEdit:  false,
		},
		{
			name: "fetch and search tools (readonly mode)",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: "https://example.atlassian.net"
  readonly: true
`,
			expectedTools: 2,
			expectedEdit:  false,
		},
		{
			name: "all tools",
			yaml: `version: "1.0"
confluence:
  enable: true
  base_url: "https://example.atlassian.net"
  readonly: false
`,
			expectedTools: 3,
			expectedEdit:  true,
		},
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

			tools := state.ToolStore.Tools()
			assert.Len(t, tools, tt.expectedTools)

			if tt.expectedEdit {
				_, err := state.ToolStore.GetTool("confluence_page_edit")
				assert.NoError(t, err)
			} else {
				_, err := state.ToolStore.GetTool("confluence_page_edit")
				assert.Error(t, err)
			}

			// Check search tool is always registered
			_, errSearch := state.ToolStore.GetTool("confluence_search")
			if tt.expectedTools > 0 {
				assert.NoError(t, errSearch)
			} else {
				assert.Error(t, errSearch)
				assert.Contains(t, errSearch.Error(), "tool not found")
			}
		})
	}
}

func TestInitConfigurationPrecedence(t *testing.T) {
	t.Parallel()

	// Test that enable=false overrides base_url
	cfg := loadConfig(t, `version: "1.0"
confluence:
  enable: false
  base_url: https://example.atlassian.net
`)
	hub := &mockHub{}
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
	}
	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)
	assert.Empty(t, state.ToolStore.Tools())

	// Test that empty base_url prevents registration even with enable=true
	cfg = loadConfig(t, `version: "1.0"
confluence:
  enable: true
  base_url: ""
`)
	state = &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(),
		RequesterFactory: requester.NewFactory(),
		Printer:          printer.GetDefault(),
	}
	err = (&Plugin{}).Init(state, hub)
	require.NoError(t, err)
	assert.Empty(t, state.ToolStore.Tools())
}
