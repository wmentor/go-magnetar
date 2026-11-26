package gitlabplug

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)

	// Create a mock hub
	hub := &mockHub{}

	// Create plugin with properly initialized State
	// WebProcStore is not needed when gitlab is disabled (enable: false)
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	hub := &mockHub{}

	// WebProcStore is not needed when gitlab is disabled
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	hub := &mockHub{}

	// WebProcStore is not needed when base_url is empty
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
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfg)
	toolStore := tools.NewStore(maskService)
	webProcStore := testdata.NewWebProcessorStore(t)
	webProcStore.EXPECT().AddProcessorFunc(mock.Anything).Return()
	hub := &mockHub{}

	pluginInstance := &Plugin{
		state: &plugin.State{
			Config:           cfg,
			Printer:          p,
			ToolStore:        toolStore,
			RequesterFactory: clientFactory,
			WebProcStore:     webProcStore,
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

func newMergeRequestListResponse(t *testing.T, status int, body string) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchGitLabMergeRequestsList(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"
	const mrListBody = `[
		{
			"iid": 7,
			"title": "Add feature",
			"description": "details",
			"state": "opened",
			"created_at": "2026-01-02T03:04:05Z",
			"web_url": "https://gitlab.example.com/group/proj/-/merge_requests/7",
			"source_branch": "feature",
			"target_branch": "main",
			"author": {"username": "alice", "name": "Alice"}
		}
	]`

	tests := []struct {
		name         string
		projectPath  string
		baseURL      string
		response     *http.Response
		requestErr   error
		doRequest    bool
		wantAPIKey   bool
		wantErr      string
		wantContent  []string
		wantURLParts []string
	}{
		{
			name:        "valid project path returns markdown",
			projectPath: "group/proj",
			baseURL:     baseURL,
			response:    newMergeRequestListResponse(t, http.StatusOK, mrListBody),
			doRequest:   true,
			wantAPIKey:  true,
			wantContent: []string{
				"# Merge Requests (Opened)",
				"Project: group/proj",
				"Total: 1",
				"## MR !7",
				"**Title:** Add feature",
				"**Author:** alice",
				"**State:** opened",
			},
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/merge_requests",
				"state=opened",
			},
		},
		{
			name:        "empty project path returns an API error",
			projectPath: "",
			baseURL:     baseURL,
			response:    newMergeRequestListResponse(t, http.StatusNotFound, `{"message":"404 Project Not Found"}`),
			doRequest:   true,
			wantAPIKey:  true,
			wantErr:     `MR list "" returned status 404`,
			wantURLParts: []string{
				"/api/v4/projects//merge_requests",
				"state=opened",
			},
		},
		{
			name:        "API error",
			projectPath: "group/proj",
			baseURL:     baseURL,
			requestErr:  errors.New("connection refused"),
			doRequest:   true,
			wantAPIKey:  true,
			wantErr:     "connection refused",
		},
		{
			name:        "HTTP error status",
			projectPath: "group/proj",
			baseURL:     baseURL,
			response:    newMergeRequestListResponse(t, http.StatusInternalServerError, `{"message":"boom"}`),
			doRequest:   true,
			wantAPIKey:  true,
			wantErr:     "returned status 500",
		},
		{
			name:        "empty merge request list",
			projectPath: "group/proj",
			baseURL:     baseURL,
			response:    newMergeRequestListResponse(t, http.StatusOK, "[]"),
			doRequest:   true,
			wantAPIKey:  true,
			wantContent: []string{
				"# Merge Requests (Opened)",
				"Project: group/proj",
				"No merge requests found.",
			},
		},
		{
			name:        "malformed base URL",
			projectPath: "group/proj",
			baseURL:     "https://[::1",
			wantErr:     "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().String("gitlab.base_url").Return(tt.baseURL).Once()
			if tt.wantAPIKey {
				cfg.EXPECT().String("gitlab.api_key").Return("test-token").Once()
			}

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()
			if tt.doRequest {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						for _, part := range tt.wantURLParts {
							require.Contains(t, req.URL.String(), part)
						}
						return tt.response, tt.requestErr
					}).Once()
			}

			pluginInstance := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			got, err := pluginInstance.fetchGitLabMergeRequestsList(context.Background(), tt.projectPath)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			for _, content := range tt.wantContent {
				require.Contains(t, got, content)
			}
		})
	}
}

func TestFetchGitLabMergeRequest(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"
	const mrDetailBody = `{
		"id": 100,
		"iid": 42,
		"title": "Fix login flow",
		"description": "Resolves token expiry edge case",
		"state": "opened",
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-01-03T04:05:06Z",
		"web_url": "https://gitlab.example.com/group/proj/-/merge_requests/42",
		"source_branch": "fix-login",
		"target_branch": "main",
		"author": {"username": "alice", "name": "Alice"},
		"labels": ["bug", "security"],
		"milestone": {"title": "v1.0"},
		"changes_count": "1"
	}`
	const mrChangesBody = `{
		"changes": [
			{
				"old_path": "auth.go",
				"new_path": "auth.go",
				"new_file": false,
				"deleted_file": false,
				"renamed_file": false,
				"diff": "@@ -1,1 +1,1 @@\n-old\n+new"
			}
		]
	}`

	type mockCall struct {
		wantURLPart string
		response    *http.Response
		err         error
	}

	tests := []struct {
		name         string
		projectPath  string
		mrID         string
		baseURL      string
		useExpiredCx bool
		calls        []mockCall
		wantErr      string
		wantErrIs    error
		wantContent  []string
	}{
		{
			name:        "successful request returns formatted markdown",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusOK, mrDetailBody),
				},
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42/changes",
					response:    newMergeRequestListResponse(t, http.StatusOK, mrChangesBody),
				},
			},
			wantContent: []string{
				"Project: group/proj",
				"Title: Fix login flow",
				"Status: opened",
				"Author: alice",
				"Source Branch: fix-login",
				"Target Branch: main",
				"Milestone: v1.0",
				"Labels: bug, security",
				"Description:\nResolves token expiry edge case",
				"File Changes:",
				"1. Modified: auth.go",
			},
		},
		{
			name:        "invalid empty projectPath returns API error",
			projectPath: "",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects//merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusNotFound, `{"message":"404 Project Not Found"}`),
				},
			},
			wantErr: `MR ""/"42" returned status 404`,
		},
		{
			name:        "invalid empty mrID returns API error",
			projectPath: "group/proj",
			mrID:        "",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/",
					response:    newMergeRequestListResponse(t, http.StatusNotFound, `{"message":"404 Not Found"}`),
				},
			},
			wantErr: `MR "group/proj"/"" returned status 404`,
		},
		{
			name:        "HTTP 404 error",
			projectPath: "group/proj",
			mrID:        "999",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/999",
					response:    newMergeRequestListResponse(t, http.StatusNotFound, `{"message":"404 Not Found"}`),
				},
			},
			wantErr: `MR "group/proj"/"999" returned status 404`,
		},
		{
			name:        "HTTP 401 unauthorized error",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusUnauthorized, `{"message":"401 Unauthorized"}`),
				},
			},
			wantErr: `MR "group/proj"/"42" returned status 401`,
		},
		{
			name:        "HTTP 403 forbidden error",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusForbidden, `{"message":"403 Forbidden"}`),
				},
			},
			wantErr: `MR "group/proj"/"42" returned status 403`,
		},
		{
			name:        "HTTP 500 server error",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusInternalServerError, `{"message":"boom"}`),
				},
			},
			wantErr: `MR "group/proj"/"42" returned status 500`,
		},
		{
			name:         "context timeout error",
			projectPath:  "group/proj",
			mrID:         "42",
			baseURL:      baseURL,
			useExpiredCx: true,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					err:         context.DeadlineExceeded,
				},
			},
			wantErr:   "failed to fetch MR",
			wantErrIs: context.DeadlineExceeded,
		},
		{
			name:        "empty or invalid JSON API response",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusOK, "{invalid"),
				},
			},
			wantErr: "failed to parse response",
		},
		{
			name:        "changes endpoint HTTP 500 error",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusOK, mrDetailBody),
				},
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42/changes",
					response:    newMergeRequestListResponse(t, http.StatusInternalServerError, `{"message":"boom"}`),
				},
			},
			wantErr: `MR changes "group/proj"/"42" returned status 500`,
		},
		{
			name:        "changes endpoint invalid JSON response",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     baseURL,
			calls: []mockCall{
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42",
					response:    newMergeRequestListResponse(t, http.StatusOK, mrDetailBody),
				},
				{
					wantURLPart: "/api/v4/projects/group%2Fproj/merge_requests/42/changes",
					response:    newMergeRequestListResponse(t, http.StatusOK, ""),
				},
			},
			wantErr: "failed to parse changes response",
		},
		{
			name:        "malformed base URL",
			projectPath: "group/proj",
			mrID:        "42",
			baseURL:     "https://[::1",
			wantErr:     "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			baseURLCalls := max(1, len(tt.calls))
			cfg.EXPECT().String("gitlab.base_url").Return(tt.baseURL).Times(baseURLCalls)
			if len(tt.calls) > 0 {
				cfg.EXPECT().String("gitlab.api_key").Return("test-token").Times(len(tt.calls))
			}

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			callIdx := 0
			if len(tt.calls) > 0 {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						c := tt.calls[callIdx]
						callIdx++
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						require.Contains(t, req.URL.String(), c.wantURLPart)
						return c.response, c.err
					}).Times(len(tt.calls))
			}

			pluginInstance := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			ctx := context.Background()
			if tt.useExpiredCx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
			}

			got, err := pluginInstance.fetchGitLabMergeRequest(ctx, tt.projectPath, tt.mrID)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			for _, content := range tt.wantContent {
				require.Contains(t, got, content)
			}
		})
	}
}

func newGitLabFileResponse(t *testing.T, status int, body string) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchGitLabFile(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"

	tests := []struct {
		name         string
		projectPath  string
		branch       string
		fileName     string
		baseURL      string
		doRequest    bool
		wantAPIKey   bool
		wantURLParts []string
		response     *http.Response
		requestErr   error
		useExpiredCx bool
		wantResult   string
		wantErr      string
		wantErrIs    error
	}{
		{
			name:        "valid base64 encoded file fetch",
			projectPath: "group/subgroup/project",
			branch:      "main",
			fileName:    "src/main.go",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fsubgroup%2Fproject/repository/files/src%2Fmain.go",
				"ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"name": "main.go",
				"path": "src/main.go",
				"size": 26,
				"encoding": "base64",
				"content": "cGFja2FnZSBtYWluCgpmdW5jIG1haW4oKSB7fQo=",
				"ref": "main"
			}`),
			wantResult: "package main\n\nfunc main() {}\n",
		},
		{
			name:        "valid plain text encoded file fetch",
			projectPath: "namespace/repo",
			branch:      "feature-branch",
			fileName:    "README.md",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/namespace%2Frepo/repository/files/README.md",
				"ref=feature-branch",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"name": "README.md",
				"path": "README.md",
				"size": 12,
				"encoding": "text",
				"content": "# My Project",
				"ref": "feature-branch"
			}`),
			wantResult: "# My Project",
		},
		{
			name:        "empty branch defaults to main",
			projectPath: "user/repo",
			branch:      "",
			fileName:    "config.yml",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/user%2Frepo/repository/files/config.yml",
				"ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"name": "config.yml",
				"path": "config.yml",
				"size": 13,
				"encoding": "base64",
				"content": "dmVyc2lvbjogMS4wCg==",
				"ref": "main"
			}`),
			wantResult: "version: 1.0\n",
		},
		{
			name:        "special characters in branch and filename",
			projectPath: "group/repo",
			branch:      "feature/test#1",
			fileName:    "docs/path with spaces/file.txt",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Frepo/repository/files/docs%2Fpath%20with%20spaces%2Ffile.txt",
				"ref=feature/test#1",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"name": "file.txt",
				"path": "docs/path with spaces/file.txt",
				"size": 15,
				"encoding": "base64",
				"content": "c3BlY2lhbCBjb250ZW50",
				"ref": "feature/test#1"
			}`),
			wantResult: "special content",
		},
		{
			name:        "HTTP 404 file not found",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "missing.txt",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/missing.txt",
			},
			response: newGitLabFileResponse(t, http.StatusNotFound, `{"message":"404 File Not Found"}`),
			wantErr:  `file "missing.txt" in project "group/proj" returned status 404`,
		},
		{
			name:        "HTTP 401 unauthorized",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "secret.txt",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/secret.txt",
			},
			response: newGitLabFileResponse(t, http.StatusUnauthorized, `{"message":"401 Unauthorized"}`),
			wantErr:  `file "secret.txt" in project "group/proj" returned status 401`,
		},
		{
			name:        "HTTP 500 internal server error",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "app.go",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/app.go",
			},
			response: newGitLabFileResponse(t, http.StatusInternalServerError, `{"message":"Internal Server Error"}`),
			wantErr:  `file "app.go" in project "group/proj" returned status 500`,
		},
		{
			name:        "invalid JSON response",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "broken.json",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/broken.json",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{invalid json`),
			wantErr:  "failed to parse response",
		},
		{
			name:        "empty response body with no content",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "empty.txt",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/empty.txt",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{"name":"empty.txt","content":""}`),
			wantErr:  "file empty.txt has no content",
		},
		{
			name:        "invalid base64 content",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "corrupted.bin",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/corrupted.bin",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"name": "corrupted.bin",
				"encoding": "base64",
				"content": "not-valid-base64!!!"
			}`),
			wantErr: "failed to decode file",
		},
		{
			name:         "context timeout error",
			projectPath:  "group/proj",
			branch:       "main",
			fileName:     "timeout.txt",
			baseURL:      baseURL,
			doRequest:    true,
			wantAPIKey:   true,
			useExpiredCx: true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/files/timeout.txt",
			},
			requestErr: context.DeadlineExceeded,
			wantErr:    "failed to fetch file",
			wantErrIs:  context.DeadlineExceeded,
		},
		{
			name:        "malformed base URL",
			projectPath: "group/proj",
			branch:      "main",
			fileName:    "file.txt",
			baseURL:     "https://[::1",
			doRequest:   false,
			wantErr:     "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().String("gitlab.base_url").Return(tt.baseURL).Once()
			if tt.wantAPIKey {
				cfg.EXPECT().String("gitlab.api_key").Return("test-token").Once()
			}

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			if tt.doRequest {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						for _, part := range tt.wantURLParts {
							require.Contains(t, req.URL.String(), part)
						}
						return tt.response, tt.requestErr
					}).Once()
			}

			pluginInstance := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			ctx := context.Background()
			if tt.useExpiredCx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
			}

			got, err := pluginInstance.fetchGitLabFile(ctx, tt.projectPath, tt.branch, tt.fileName)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantResult, got)
		})
	}
}

func TestFetchGitLabTree(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"

	tests := []struct {
		name         string
		projectPath  string
		branch       string
		path         string
		baseURL      string
		doRequest    bool
		wantAPIKey   bool
		wantURLParts []string
		response     *http.Response
		requestErr   error
		wantContain  []string
		wantErr      string
	}{
		{
			name:        "successful request to root directory",
			projectPath: "group/subgroup/project",
			branch:      "main",
			path:        "",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fsubgroup%2Fproject/repository/tree?ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `[
				{"name":"cmd","path":"cmd","type":"tree","mode":"040000"},
				{"name":"README.md","path":"README.md","type":"blob","mode":"100644"}
			]`),
			wantContain: []string{"| cmd | tree | dir |", "README.md"},
		},
		{
			name:        "successful request to subdirectory",
			projectPath: "namespace/repo",
			branch:      "develop",
			path:        "internal/pkg",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/namespace%2Frepo/repository/tree?ref=develop",
				"path=internal%2Fpkg",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `[
				{"name":"service.go","path":"internal/pkg/service.go","type":"blob","mode":"100644"}
			]`),
			wantContain: []string{"internal/pkg", "service.go"},
		},
		{
			name:        "empty branch defaults to main",
			projectPath: "user/repo",
			branch:      "",
			path:        "",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/user%2Frepo/repository/tree?ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `[
				{"name":"go.mod","path":"go.mod","type":"blob","mode":"100644"}
			]`),
			wantContain: []string{"go.mod"},
		},
		{
			name:        "HTTP transport error",
			projectPath: "group/proj",
			branch:      "main",
			path:        "src",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/tree?ref=main",
			},
			requestErr: errors.New("connection refused"),
			wantErr:    "failed to fetch tree",
		},
		{
			name:        "HTTP 404 error",
			projectPath: "group/proj",
			branch:      "main",
			path:        "missing",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/tree?ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusNotFound, `{"message":"404 Tree Not Found"}`),
			wantErr:  "returned status 404",
		},
		{
			name:        "HTTP 500 error",
			projectPath: "group/proj",
			branch:      "main",
			path:        "",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/tree?ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusInternalServerError, `{"message":"Internal Server Error"}`),
			wantErr:  "returned status 500",
		},
		{
			name:        "invalid JSON response",
			projectPath: "group/proj",
			branch:      "main",
			path:        "",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj/repository/tree?ref=main",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{invalid json`),
			wantErr:  "failed to parse response",
		},
		{
			name:        "malformed base URL",
			projectPath: "group/proj",
			branch:      "main",
			path:        "",
			baseURL:     "https://[::1",
			doRequest:   false,
			wantErr:     "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().String("gitlab.base_url").Return(tt.baseURL).Once()
			if tt.wantAPIKey {
				cfg.EXPECT().String("gitlab.api_key").Return("test-token").Once()
			}

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			if tt.doRequest {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						for _, part := range tt.wantURLParts {
							require.Contains(t, req.URL.String(), part)
						}
						return tt.response, tt.requestErr
					}).Once()
			}

			pluginInstance := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			got, err := pluginInstance.fetchGitLabTree(context.Background(), tt.projectPath, tt.branch, tt.path)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			for _, want := range tt.wantContain {
				require.Contains(t, got, want)
			}
		})
	}
}

func TestFetchGitLabRepository(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"

	tests := []struct {
		name           string
		projectPath    string
		baseURL        string
		doRequest      bool
		wantAPIKey     bool
		wantURLParts   []string
		response       *http.Response
		requestErr     error
		wantContain    []string
		wantNotContain []string
		wantErr        string
	}{
		{
			name:        "successful request with full metadata",
			projectPath: "group/proj",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"id": 101,
				"name": "My Project",
				"path_with_namespace": "group/proj",
				"description": "Sample repository",
				"web_url": "https://gitlab.example.com/group/proj",
				"star_count": 42,
				"forks_count": 5,
				"open_issues_count": 3,
				"default_branch": "main",
				"visibility": "public",
				"ssh_url_to_repo": "git@gitlab.example.com:group/proj.git",
				"http_url_to_repo": "https://gitlab.example.com/group/proj.git",
				"last_activity_at": "2026-11-26T10:00:00Z"
			}`),
			wantContain: []string{
				"# My Project",
				"**Project Path:** group/proj",
				"**Default Branch:** main",
				"**Visibility:** public",
				"**Forks:** 5 | **Stars:** 42 | **Open Issues:** 3",
				"**Description:**\nSample repository",
				"**Web URL:** https://gitlab.example.com/group/proj",
				"**SSH URL:** git@gitlab.example.com:group/proj.git",
				"**HTTP URL:** https://gitlab.example.com/group/proj.git",
				"**Last Activity:** 2026-11-26T10:00:00Z",
			},
		},
		{
			name:        "successful request with subgroup and minimal fields",
			projectPath: "group/subgroup/proj",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fsubgroup%2Fproj",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{
				"id": 102,
				"name": "Sub Project",
				"path_with_namespace": "group/subgroup/proj",
				"default_branch": "develop",
				"visibility": "private",
				"web_url": "https://gitlab.example.com/group/subgroup/proj",
				"ssh_url_to_repo": "git@gitlab.example.com:group/subgroup/proj.git",
				"http_url_to_repo": "https://gitlab.example.com/group/subgroup/proj.git"
			}`),
			wantContain: []string{
				"# Sub Project",
				"**Project Path:** group/subgroup/proj",
				"**Default Branch:** develop",
				"**Visibility:** private",
			},
			wantNotContain: []string{
				"**Description:**",
				"**Last Activity:**",
			},
		},
		{
			name:        "HTTP transport error",
			projectPath: "group/proj",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj",
			},
			requestErr: errors.New("connection refused"),
			wantErr:    "failed to fetch project",
		},
		{
			name:        "HTTP 404 error",
			projectPath: "group/missing",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fmissing",
			},
			response: newGitLabFileResponse(t, http.StatusNotFound, `{"message":"404 Project Not Found"}`),
			wantErr:  "returned status 404",
		},
		{
			name:        "HTTP 500 error",
			projectPath: "group/proj",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj",
			},
			response: newGitLabFileResponse(t, http.StatusInternalServerError, `{"message":"Internal Server Error"}`),
			wantErr:  "returned status 500",
		},
		{
			name:        "invalid JSON response",
			projectPath: "group/proj",
			baseURL:     baseURL,
			doRequest:   true,
			wantAPIKey:  true,
			wantURLParts: []string{
				"/api/v4/projects/group%2Fproj",
			},
			response: newGitLabFileResponse(t, http.StatusOK, `{invalid json`),
			wantErr:  "failed to parse response",
		},
		{
			name:        "malformed base URL",
			projectPath: "group/proj",
			baseURL:     "https://[::1",
			doRequest:   false,
			wantAPIKey:  false,
			wantErr:     "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().String("gitlab.base_url").Return(tt.baseURL).Once()
			if tt.wantAPIKey {
				cfg.EXPECT().String("gitlab.api_key").Return("test-token").Once()
			}

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			if tt.doRequest {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						for _, part := range tt.wantURLParts {
							require.Contains(t, req.URL.String(), part)
						}
						return tt.response, tt.requestErr
					}).Once()
			}

			pluginInstance := &Plugin{
				state: &plugin.State{
					Config:           cfg,
					Printer:          printer.New(io.Discard, false),
					RequesterFactory: factory,
				},
			}

			got, err := pluginInstance.fetchGitLabRepository(context.Background(), tt.projectPath)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			for _, want := range tt.wantContain {
				require.Contains(t, got, want)
			}
			for _, unwanted := range tt.wantNotContain {
				require.NotContains(t, got, unwanted)
			}
		})
	}
}
