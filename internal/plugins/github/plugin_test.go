package githubplug

import (
	"context"
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	webProcStore := testdata.NewWebProcessorStore(t)
	if enableGithub {
		webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	}

	return &plugin.State{
		Config:           cfg,
		Printer:          p,
		ToolStore:        toolStore,
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)
	toolStore := tools.NewStore(maskService)
	webProcStore := testdata.NewWebProcessorStore(t)
	// WebProcStore is not needed when github section is missing

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        toolStore,
		RequesterFactory: clientFactory,
		WebProcStore:     webProcStore,
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)
	toolStore := tools.NewStore(maskService)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        toolStore,
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

func TestPlugin_execWebFetch_SkipsNonGitHubURLs(t *testing.T) {
	t.Parallel()

	state := createTestState(t, "", true)
	p := &Plugin{}
	require.NoError(t, p.Init(state, &mockHub{}))

	tests := []struct {
		name string
		url  string
	}{
		{name: "empty url", url: ""},
		{name: "plain text", url: "not a url"},
		{name: "other host", url: "https://example.com/owner/repo/issues/1"},
		{name: "gitlab host", url: "https://gitlab.com/owner/repo/-/merge_requests/1"},
		{name: "github without trailing slash", url: "https://github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := p.execWebFetch(context.Background(), tt.url)
			require.ErrorIs(t, err, plugin.ErrNoWebProcessor)
			assert.Empty(t, got)
		})
	}
}

func newGitHubAPIResponse(t *testing.T, status int, body string) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestPlugin_execWebFetch_RoutesAndErrors(t *testing.T) {
	t.Parallel()

	type mockHTTPResp struct {
		wantURLSubstr string
		status        int
		body          string
		err           error
	}

	tests := []struct {
		name        string
		url         string
		enabled     bool
		responses   []mockHTTPResp
		wantContain []string
		wantErr     string
		wantErrIs   error
	}{
		{
			name:    "file blob URL routes to FetchFile",
			url:     "https://github.com/owner/repo/blob/main/README.md",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/contents/README.md?ref=main",
					status:        http.StatusOK,
					body:          `{"name":"README.md","path":"README.md","encoding":"base64","content":"SGVsbG8gV29ybGQ="}`,
				},
			},
			wantContain: []string{"Hello World"},
		},
		{
			name:    "tree URL routes to FetchTree",
			url:     "https://github.com/owner/repo/tree/main/cmd",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/git/trees/main?path=cmd&recursive=1",
					status:        http.StatusOK,
					body:          `{"sha":"abc123","tree":[{"path":"cmd/main.go","type":"blob","size":120}]}`,
				},
			},
			wantContain: []string{"cmd/main.go"},
		},
		{
			name:    "pull request URL routes to FetchPullRequest",
			url:     "https://github.com/owner/repo/pull/42",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/pulls/42",
					status:        http.StatusOK,
					body:          `{"number":42,"title":"Add feature","state":"open","user":{"login":"alice"},"body":"PR body"}`,
				},
				{
					wantURLSubstr: "/repos/owner/repo/pulls/42/files",
					status:        http.StatusOK,
					body:          `[{"filename":"main.go","status":"modified","additions":5,"deletions":2}]`,
				},
				{
					wantURLSubstr: "/repos/owner/repo/pulls/42/comments",
					status:        http.StatusOK,
					body:          `[]`,
				},
				{
					wantURLSubstr: "/repos/owner/repo/pulls/42/reviews",
					status:        http.StatusOK,
					body:          `[]`,
				},
			},
			wantContain: []string{"Add feature", "main.go"},
		},
		{
			name:    "pull requests list URL routes to FetchPullRequestsList",
			url:     "https://github.com/owner/repo/pulls",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/pulls?state=open",
					status:        http.StatusOK,
					body:          `[{"number":10,"title":"Open PR","user":{"login":"bob"},"created_at":"2026-01-01T00:00:00Z"}]`,
				},
			},
			wantContain: []string{"Open PR"},
		},
		{
			name:    "issue URL routes to FetchIssue",
			url:     "https://github.com/owner/repo/issues/7",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/issues/7",
					status:        http.StatusOK,
					body:          `{"number":7,"title":"Bug report","state":"open","comments":0,"user":{"login":"carol"},"body":"Details"}`,
				},
			},
			wantContain: []string{"Bug report", "Details"},
		},
		{
			name:    "open issues list URL routes to FetchOpenIssues",
			url:     "https://github.com/owner/repo/issues",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/issues?state=open",
					status:        http.StatusOK,
					body:          `[{"number":8,"title":"Listed issue","state":"open","user":{"login":"dave"}}]`,
				},
			},
			wantContain: []string{"Listed issue"},
		},
		{
			name:    "milestone URL routes to FetchMilestone",
			url:     "https://github.com/owner/repo/milestone/3",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/milestones/3",
					status:        http.StatusOK,
					body:          `{"number":3,"title":"v2.0","state":"open","open_issues":1,"closed_issues":2}`,
				},
				{
					wantURLSubstr: "/repos/owner/repo/issues?state=all&milestone=3",
					status:        http.StatusOK,
					body:          `[{"number":9,"title":"Milestone task","state":"open","user":{"login":"eve"}}]`,
				},
			},
			wantContain: []string{"v2.0", "Milestone task"},
		},
		{
			name:    "advisory URL routes to FetchAdvisory",
			url:     "https://github.com/advisories/GHSA-1234-5678-9abc",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/advisories/GHSA-1234-5678-9abc",
					status:        http.StatusOK,
					body:          `{"ghsa_id":"GHSA-1234-5678-9abc","summary":"Test Advisory","severity":"high"}`,
				},
			},
			wantContain: []string{"GHSA-1234-5678-9abc", "Test Advisory"},
		},
		{
			name:    "repo security advisories URL routes to FetchRepoAdvisories",
			url:     "https://github.com/owner/repo/security/advisories",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/security-advisories",
					status:        http.StatusOK,
					body:          `[{"ghsa_id":"GHSA-9999-8888-7777","summary":"Repo Advisory","severity":"medium"}]`,
				},
			},
			wantContain: []string{"GHSA-9999-8888-7777", "Repo Advisory"},
		},
		{
			name:    "release tag URL routes to FetchRelease",
			url:     "https://github.com/owner/repo/releases/tag/v1.5.0",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/releases/tags/v1.5.0",
					status:        http.StatusOK,
					body:          `{"tag_name":"v1.5.0","name":"Release v1.5.0","body":"Release notes"}`,
				},
			},
			wantContain: []string{"Release v1.5.0", "Release notes"},
		},
		{
			name:    "repository root URL routes to FetchRepository",
			url:     "https://github.com/owner/repo",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo",
					status:        http.StatusOK,
					body:          `{"full_name":"owner/repo","default_branch":"main","language":"Go","description":"Awesome tool"}`,
				},
				{
					wantURLSubstr: "/repos/owner/repo/readme",
					status:        http.StatusOK,
					body:          `{"encoding":"base64","content":"IyBSZWFkbWU="}`,
				},
			},
			wantContain: []string{"owner/repo", "Awesome tool", "# Readme"},
		},
		{
			name:    "API HTTP 404 returns error",
			url:     "https://github.com/owner/repo/issues/999",
			enabled: true,
			responses: []mockHTTPResp{
				{
					wantURLSubstr: "/repos/owner/repo/issues/999",
					status:        http.StatusNotFound,
					body:          `{"message":"Not Found"}`,
				},
			},
			wantErr: "returned status 404",
		},
		{
			name:    "disabled plugin returns disabled error",
			url:     "https://github.com/owner/repo",
			enabled: false,
			wantErr: "github tool is disabled",
		},
		{
			name:      "single segment GitHub path returns ErrNoWebProcessor",
			url:       "https://github.com/explore",
			enabled:   true,
			wantErrIs: plugin.ErrNoWebProcessor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().Bool("github.enable").Return(tt.enabled).Maybe()
			cfg.EXPECT().String("github.base_url").Return("https://api.github.com").Maybe()
			cfg.EXPECT().String("github.api_key").Return("test-token").Maybe()

			factory := testdata.NewClientFactory(t)
			if len(tt.responses) > 0 {
				client := testdata.NewClient(t)
				factory.EXPECT().Get().Return(client).Times(len(tt.responses))
				factory.EXPECT().Put(client).Times(len(tt.responses))

				idx := 0
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						spec := tt.responses[idx]
						idx++
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/vnd.github+json", req.Header.Get("Accept"))
						require.Contains(t, req.URL.String(), spec.wantURLSubstr)
						if spec.err != nil {
							return nil, spec.err
						}
						return newGitHubAPIResponse(t, spec.status, spec.body), nil
					}).Times(len(tt.responses))
			}

			p := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			got, err := p.execWebFetch(context.Background(), tt.url)
			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Empty(t, got)
				return
			}
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}
