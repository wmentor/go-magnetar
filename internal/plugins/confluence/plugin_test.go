package confluenceplugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/mask"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
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
			maskService := mask.NewService(cfg)
			webProcStore := testdata.NewWebProcessorStore(t)
			if tt.tools != nil {
				webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
			}
			state := &plugin.State{
				Config:           cfg,
				ToolStore:        tools.NewStore(maskService),
				RequesterFactory: testdata.NewClientFactory(t),
				Printer:          printer.GetDefault(),
				WebProcStore:     webProcStore,
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
	maskService := mask.NewService(cfg)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: testdata.NewClientFactory(t),
		Printer:          printer.GetDefault(),
		WebProcStore:     webProcStore,
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
	maskService := mask.NewService(cfg)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: testdata.NewClientFactory(t),
		Printer:          printer.GetDefault(),
		WebProcStore:     webProcStore,
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
	maskService := mask.NewService(cfg)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: testdata.NewClientFactory(t),
		Printer:          printer.GetDefault(),
		WebProcStore:     webProcStore,
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
			maskService := mask.NewService(cfg)
			webProcStore := testdata.NewWebProcessorStore(t)
			// Only expect AddProcessorFunc if tools are expected to be registered
			if tt.expectedTools > 0 {
				webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
			}
			state := &plugin.State{
				Config:           cfg,
				ToolStore:        tools.NewStore(maskService),
				RequesterFactory: testdata.NewClientFactory(t),
				Printer:          printer.GetDefault(),
				WebProcStore:     webProcStore,
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
	maskService := mask.NewService(cfg)
	webProcStore := testdata.NewWebProcessorStore(t)
	// No AddProcessorFunc expected because enable=false
	state := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: testdata.NewClientFactory(t),
		Printer:          printer.GetDefault(),
		WebProcStore:     webProcStore,
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
	webProcStore2 := testdata.NewWebProcessorStore(t)
	// No AddProcessorFunc expected because base_url is empty
	state2 := &plugin.State{
		Config:           cfg,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: testdata.NewClientFactory(t),
		Printer:          printer.GetDefault(),
		WebProcStore:     webProcStore2,
	}
	err = (&Plugin{}).Init(state2, hub)
	require.NoError(t, err)
	assert.Empty(t, state2.ToolStore.Tools())
}

func TestPlugin_webFetch(t *testing.T) {
	t.Parallel()

	const pageBody = `{
		"id": "12345",
		"title": "Test Page",
		"space": {"key": "TEAM"},
		"version": {"number": 7, "when": "2026-01-02T03:04:05.000Z"},
		"body": {"storage": {"value": "<p>Page content</p>"}},
		"history": {
			"createdBy": {"username": "alice", "displayName": "Alice Example"},
			"createdDate": "2025-12-31T12:00:00.000Z"
		}
	}`

	tests := []struct {
		name           string
		baseURL        string
		url            string
		mockSetup      func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config)
		expectedOutput string
		expectedError  string
	}{
		{
			name:    "standard URL with numeric page ID",
			baseURL: "https://confluence.example.com",
			url:     "https://confluence.example.com/spaces/TEAM/pages/12345/Test Page",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
				config.EXPECT().String("confluence.api_key").Return("test-token")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://confluence.example.com/rest/api/content/12345?expand=body.storage,space,version,history.createdBy", req.URL.String())
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(pageBody)),
						}, nil
					}).Once()
			},
			expectedOutput: "Confluence page ID: 12345\nSpace: TEAM\nTitle: Test Page\nCreated: 2025-12-31T12:00:00.000Z\nAuthor: Alice Example (alice)\nUpdated: 2026-01-02T03:04:05.000Z\nVersion: 7\nBody:\n<p>Page content</p>",
		},
		{
			name:    "short URL with /x/ path",
			baseURL: "https://confluence.example.com",
			url:     "https://confluence.example.com/x/A4HhC",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
				config.EXPECT().String("confluence.api_key").Return("test-token")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://confluence.example.com/rest/api/content/148996355?expand=body.storage,space,version,history.createdBy", req.URL.String())
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(pageBody)),
						}, nil
					}).Once()
			},
			expectedOutput: "Confluence page ID: 12345\nSpace: TEAM\nTitle: Test Page\nCreated: 2025-12-31T12:00:00.000Z\nAuthor: Alice Example (alice)\nUpdated: 2026-01-02T03:04:05.000Z\nVersion: 7\nBody:\n<p>Page content</p>",
		},
		{
			name:    "share URL with /p/ path",
			baseURL: "https://confluence.example.com",
			url:     "https://confluence.example.com/p/XYZ789",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
				config.EXPECT().String("confluence.api_key").Return("test-token")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://confluence.example.com/rest/api/content/4084958813?expand=body.storage,space,version,history.createdBy", req.URL.String())
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						// Use a custom page body with the correct ID for XYZ789
						pageBody := `{
							"id": "4084958813",
							"title": "Test Page",
							"space": {"key": "TEAM"},
							"version": {"number": 7, "when": "2026-01-02T03:04:05.000Z"},
							"body": {"storage": {"value": "<p>Page content</p>"}},
							"history": {
								"createdBy": {"username": "alice", "displayName": "Alice Example"},
								"createdDate": "2025-12-31T12:00:00.000Z"
							}
						}`
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(pageBody)),
						}, nil
					}).Once()
			},
			expectedOutput: "Confluence page ID: 4084958813\nSpace: TEAM\nTitle: Test Page\nCreated: 2025-12-31T12:00:00.000Z\nAuthor: Alice Example (alice)\nUpdated: 2026-01-02T03:04:05.000Z\nVersion: 7\nBody:\n<p>Page content</p>",
		},
		{
			name:    "URL without confluence prefix",
			baseURL: "https://confluence.example.com",
			url:     "https://other-site.com/page/123",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
			},
			expectedError: "web processor skip URL",
		},
		{
			name:    "network error",
			baseURL: "https://confluence.example.com",
			url:     "https://confluence.example.com/spaces/TEAM/pages/12345",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
				config.EXPECT().String("confluence.api_key").Return("test-token")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).Return(nil, errors.New("connection refused")).Once()
			},
			expectedError: "confluence_page_fetch: failed to fetch page",
		},
		{
			name:    "HTTP error response",
			baseURL: "https://confluence.example.com",
			url:     "https://confluence.example.com/spaces/TEAM/pages/12345",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, config *testdata.Config) {
				config.EXPECT().String("confluence.base_url").Return("https://confluence.example.com")
				config.EXPECT().String("confluence.api_key").Return("test-token")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).Return(&http.Response{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"message":"forbidden"}`)),
				}, nil).Once()
			},
			expectedError: "confluence_page_fetch: page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := loadConfig(t, fmt.Sprintf(`version: "1.0"
confluence:
  enable: true
  base_url: %q
  api_key: "test-token"
`, tt.baseURL))

			clientFactory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			config := testdata.NewConfig(t)

			// Configure mock expectations
			if tt.mockSetup != nil {
				tt.mockSetup(t, clientFactory, client, config)
			}

			// Create plugin with mock config
			maskService := mask.NewService(cfg)
			state := &plugin.State{
				Config:           config,
				ToolStore:        tools.NewStore(maskService),
				RequesterFactory: clientFactory,
				Printer:          printer.GetDefault(),
				WebProcStore:     testdata.NewWebProcessorStore(t),
			}
			pluginInstance := &Plugin{}
			pluginInstance.state = state

			// Create a custom config that returns our mock config
			state.Config = config

			// Initialize fetch tool
			pluginInstance.fetchTool = confluence.NewFetchTool(config, state.Printer, state.RequesterFactory)
			state.ToolStore.Register(pluginInstance.fetchTool.Name(), pluginInstance.fetchTool.Description(), pluginInstance.fetchTool.Exec)

			// Call webFetch
			result, err := pluginInstance.webFetch(context.Background(), tt.url)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Empty(t, result)
				return
			}

			require.NoError(t, err)
			require.Contains(t, result, tt.expectedOutput)
		})
	}
}

func TestPlugin_webFetch_NilState(t *testing.T) {
	t.Parallel()

	pluginInstance := &Plugin{}
	// pluginInstance.state is nil

	// This should panic because webFetch does not check for nil state
	// To prevent panic, we need to recover
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("webFetch did not panic with nil state")
		}
	}()

	_, _ = pluginInstance.webFetch(context.Background(), "https://confluence.example.com/spaces/TEAM/pages/12345")
}
