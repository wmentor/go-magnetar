package jiraplug

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	webProcStore := testdata.NewWebProcessorStore(t)
	if enableJira {
		webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	}

	return &plugin.State{
		Config:           cfg,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)
	webProcStore := testdata.NewWebProcessorStore(t)

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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

func newMockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestPlugin_WebFetchProc(t *testing.T) {
	t.Parallel()

	const issueBodyGOARCH = `{
		"id": "10001",
		"key": "GOARCH-60",
		"self": "https://jira.example.com/rest/api/2/issue/10001",
		"fields": {
			"summary": "Implement task fetcher",
			"description": "Add robust issue retrieval",
			"status": {"name": "In Progress"},
			"assignee": {"displayName": "Alice"},
			"reporter": {"displayName": "Bob"},
			"project": {"name": "Architecture"},
			"issuetype": {"name": "Task"},
			"created": "2026-01-10T10:00:00Z",
			"updated": "2026-01-11T12:00:00Z"
		}
	}`

	const issueBodyPROJ = `{
		"id": "10002",
		"key": "PROJ-123",
		"self": "https://jira.example.com/rest/api/2/issue/10002",
		"fields": {
			"summary": "Fix login bug",
			"description": "User cannot log in",
			"status": {"name": "Open"},
			"assignee": {"displayName": "Charlie"},
			"reporter": {"displayName": "Dave"},
			"project": {"name": "Project"},
			"issuetype": {"name": "Bug"},
			"created": "2026-01-12T10:00:00Z",
			"updated": "2026-01-12T12:00:00Z"
		}
	}`

	tests := []struct {
		name          string
		baseURL       string
		url           string
		mockSetup     func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config)
		wantContains  []string
		wantErr       error
		wantErrSubstr string
	}{
		{
			name:    "successful fetch with /browse/ URL",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/browse/GOARCH-60",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com")
				cfg.EXPECT().String("jira.api_key").Return("test-api-key")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://jira.example.com/rest/api/2/issue/GOARCH-60", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						return newMockHTTPResponse(http.StatusOK, issueBodyGOARCH), nil
					}).Once()
			},
			wantContains: []string{
				"Issue: GOARCH-60",
				"Summary: Implement task fetcher",
				"Status: In Progress",
				"Assignee: Alice",
				"Reporter: Bob",
			},
		},
		{
			name:    "successful fetch with /issues/ URL, query params and fragment",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/issues/PROJ-123?filter=-1#comment-456",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com")
				cfg.EXPECT().String("jira.api_key").Return("test-api-key")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://jira.example.com/rest/api/2/issue/PROJ-123", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						return newMockHTTPResponse(http.StatusOK, issueBodyPROJ), nil
					}).Once()
			},
			wantContains: []string{
				"Issue: PROJ-123",
				"Summary: Fix login bug",
				"Status: Open",
				"Assignee: Charlie",
				"Reporter: Dave",
			},
		},
		{
			name:    "unmatched base URL",
			baseURL: "https://jira.example.com",
			url:     "https://other-jira.example.com/browse/GOARCH-60",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com").Once()
			},
			wantErr: plugin.ErrNoWebProcessor,
		},
		{
			name:    "non-issue path (project URL)",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/projects/GOARCH",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com").Once()
			},
			wantErr: plugin.ErrNoWebProcessor,
		},
		{
			name:    "empty issue key in /browse/ path",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/browse/",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com").Once()
			},
			wantErr: plugin.ErrNoWebProcessor,
		},
		{
			name:    "empty issue key in /issues/ path with query",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/issues/?filter=all",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com").Once()
			},
			wantErr: plugin.ErrNoWebProcessor,
		},
		{
			name:    "HTTP 404 not found error",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/browse/NOTFOUND-404",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com")
				cfg.EXPECT().String("jira.api_key").Return("test-api-key")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusNotFound, `{"errorMessages":["Issue does not exist"]}`), nil).Once()
			},
			wantErrSubstr: "issue \"NOTFOUND-404\" returned status 404",
		},
		{
			name:    "network connection error",
			baseURL: "https://jira.example.com",
			url:     "https://jira.example.com/browse/NET-500",
			mockSetup: func(t *testing.T, clientFactory *testdata.ClientFactory, client *testdata.Client, cfg *testdata.Config) {
				cfg.EXPECT().String("jira.base_url").Return("https://jira.example.com")
				cfg.EXPECT().String("jira.api_key").Return("test-api-key")

				clientFactory.EXPECT().Get().Return(client).Once()
				clientFactory.EXPECT().Put(client).Once()

				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(nil, errors.New("connection reset by peer")).Once()
			},
			wantErrSubstr: "connection reset by peer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientFactory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			cfgMock := testdata.NewConfig(t)

			if tt.mockSetup != nil {
				tt.mockSetup(t, clientFactory, client, cfgMock)
			}

			p := &Plugin{
				state: &plugin.State{
					Config:           cfgMock,
					Printer:          printer.GetDefault(),
					RequesterFactory: clientFactory,
				},
			}

			result, err := p.webFetchProc(context.Background(), tt.url)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, result)
				return
			}

			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			for _, substr := range tt.wantContains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestPlugin_WebFetchProc_NilState(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	defer func() {
		r := recover()
		require.NotNil(t, r, "webFetchProc should panic when state is nil")
	}()

	_, _ = p.webFetchProc(context.Background(), "https://jira.example.com/browse/GOARCH-60")
}
