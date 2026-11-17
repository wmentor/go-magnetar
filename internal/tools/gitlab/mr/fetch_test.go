package mr

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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const testBaseURL = "https://gitlab.example.com"

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read([]byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}

func newTestConfig(t *testing.T, baseURL string) *config.Config {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	content := "version: \"1.0\"\ngitlab:\n  base_url: \"" + baseURL + "\"\n  api_key: \"test-api-key\"\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	return cfg
}

func newMockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// isMRRequest matches the request for the merge request itself.
func isMRRequest(req *http.Request) bool {
	return strings.HasSuffix(req.URL.Path, "/merge_requests/7")
}

// isChangesRequest matches the request for the merge request changes.
func isChangesRequest(req *http.Request) bool {
	return strings.HasSuffix(req.URL.Path, "/merge_requests/7/changes")
}

const fullMRBody = `{
  "id": 10,
  "iid": 7,
  "title": "Add feature",
  "description": "Does things",
  "state": "opened",
  "created_at": "2024-01-02T03:04:05Z",
  "updated_at": "2024-01-03T03:04:05Z",
  "web_url": "https://gitlab.example.com/ns/proj/-/merge_requests/7",
  "source_branch": "feature",
  "target_branch": "main",
  "work_in_progress": true,
  "draft": true,
  "changes_count": "4",
  "author": {"username": "alice", "name": "Alice"},
  "labels": ["bug", "ui"],
  "milestone": {"title": "v1.0"}
}`

const fullChangesBody = `{
  "changes": [
    {"old_path": "a.go", "new_path": "a.go", "diff": "@@ -1 +1 @@\n-old\n+new\n"},
    {"old_path": "b.go", "new_path": "b.go", "new_file": true, "diff": "+added"},
    {"old_path": "c.go", "new_path": "c.go", "deleted_file": true, "diff": "-removed"},
    {"old_path": "d.go", "new_path": "e.go", "renamed_file": true, "diff": ""}
  ]
}`

func TestGitLabMRNew(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)

	tool := New(cfg, p, clientFactory)
	if tool == nil {
		t.Fatal("New() returned nil")
	}

	if tool.Name() != "gitlab_fetch_mr" {
		t.Errorf("Name() = %s, want gitlab_fetch_mr", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestGitLabMRParamsValidation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)

	tool := New(cfg, p, clientFactory)

	// Test with nil params
	result, err := tool.Exec(t.Context(), nil)
	if err == nil {
		t.Error("Exec() with nil params should return an error")
	}
	if result != nil {
		t.Log("Exec() returned non-nil result with nil params (expected)")
	}
}

func TestGitLabMRExec_NilParams(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	// No Get/Put expectations: a nil params error must come before any client use.
	factory := testdata.NewClientFactory(t)
	tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)

	res, err := tool.Exec(context.Background(), nil)

	require.EqualError(t, err, "invalid params")
	require.Nil(t, res)
}

func TestGitLabMRExec_Success(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()

	client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, fullMRBody), nil).Once()
	client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
		Return(newMockHTTPResponse(http.StatusOK, fullChangesBody), nil).Once()

	tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
	res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

	require.NoError(t, err)
	resp, ok := res.(*tools.Response)
	require.True(t, ok, "expected *tools.Response, got %T", res)
	require.Equal(t, tools.StatusOK, resp.Status)

	want := "Project: ns/proj\n" +
		"Title: Add feature\n" +
		"Status: opened\n" +
		"Author: alice\n" +
		"Source Branch: feature\n" +
		"Target Branch: main\n" +
		"Milestone: v1.0\n" +
		"Labels: bug, ui\n" +
		"Draft: true\n" +
		"WIP: true\n" +
		"Changes: 4\n" +
		"Created: 2024-01-02T03:04:05Z\n" +
		"Updated: 2024-01-03T03:04:05Z\n" +
		"Web URL: https://gitlab.example.com/ns/proj/-/merge_requests/7\n" +
		"\n" +
		"Description:\n" +
		"Does things\n" +
		"\n" +
		"File Changes:\n" +
		"1. Modified: a.go\n" +
		"Diff:\n" +
		"@@ -1 +1 @@\n-old\n+new\n\n\n" +
		"2. Created: b.go\n" +
		"Diff:\n" +
		"+added\n\n" +
		"3. Deleted: c.go\n" +
		"Diff:\n" +
		"-removed\n\n" +
		"4. Renamed: d.go -> e.go\n" +
		"Diff:\n" +
		"\n\n"
	require.Equal(t, want, resp.Result)
}

func TestGitLabMRExec_EmptyResponseFields(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()

	client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
	client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
		Return(newMockHTTPResponse(http.StatusOK, `{"changes": null}`), nil).Once()

	tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
	res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

	require.NoError(t, err)
	resp, ok := res.(*tools.Response)
	require.True(t, ok, "expected *tools.Response, got %T", res)
	require.Equal(t, tools.StatusOK, resp.Status)

	// Optional lines (milestone, labels, draft, WIP, changes count, file changes) are left out.
	want := "Project: ns/proj\n" +
		"Title: \n" +
		"Status: \n" +
		"Author: \n" +
		"Source Branch: \n" +
		"Target Branch: \n" +
		"Created: \n" +
		"Updated: \n" +
		"Web URL: \n" +
		"\n" +
		"Description:\n"
	require.Equal(t, want, resp.Result)
}

func TestGitLabMRExec_RequestShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		projectPath string
		wantMRPath  string
	}{
		{
			name:        "simple project path",
			projectPath: "namespace/project",
			wantMRPath:  "/api/v4/projects/namespace%2Fproject/merge_requests/7",
		},
		{
			name:        "subgroup path",
			projectPath: "group/subgroup/project",
			wantMRPath:  "/api/v4/projects/group%2Fsubgroup%2Fproject/merge_requests/7",
		},
		{
			name:        "spaces are escaped",
			projectPath: "my group/my project",
			wantMRPath:  "/api/v4/projects/my%20group%2Fmy%20project/merge_requests/7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()

			var seen []*http.Request
			client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
				RunAndReturn(func(req *http.Request) (*http.Response, error) {
					seen = append(seen, req)
					return newMockHTTPResponse(http.StatusOK, `{}`), nil
				}).Twice()

			tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
			_, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: tt.projectPath, MRID: "7"})
			require.NoError(t, err)

			require.Len(t, seen, 2)
			wantPaths := []string{tt.wantMRPath, tt.wantMRPath + "/changes"}
			for i, req := range seen {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "gitlab.example.com", req.URL.Host)
				require.Equal(t, wantPaths[i], req.URL.EscapedPath())
				require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			}
		})
	}
}

func TestGitLabMRExec_RequestCreationError(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	// The client is taken (and returned) before the request is built.
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()

	tool := New(newTestConfig(t, "://invalid-url"), printer.New(&buf, false), factory)
	res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create request")
	require.Nil(t, res)
}

func TestGitLabMRExec_HTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(client *testdata.Client)
		wantErr    string
	}{
		{
			name: "MR not found",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusNotFound, `{"message":"404 Not found"}`), nil).Once()
			},
			wantErr: `MR "ns/proj"/"7" returned status 404`,
		},
		{
			name: "MR forbidden",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusForbidden, `{"message":"403 Forbidden"}`), nil).Once()
			},
			wantErr: `MR "ns/proj"/"7" returned status 403`,
		},
		{
			name: "MR server error",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusInternalServerError, ``), nil).Once()
			},
			wantErr: `MR "ns/proj"/"7" returned status 500`,
		},
		{
			name: "changes not found",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
				client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
					Return(newMockHTTPResponse(http.StatusNotFound, ``), nil).Once()
			},
			wantErr: `MR changes "ns/proj"/"7" returned status 404`,
		},
		{
			name: "changes forbidden",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
				client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
					Return(newMockHTTPResponse(http.StatusForbidden, ``), nil).Once()
			},
			wantErr: `MR changes "ns/proj"/"7" returned status 403`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()
			tt.setupMocks(client)

			tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
			res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.Nil(t, res)
		})
	}
}

func TestGitLabMRExec_NetworkError(t *testing.T) {
	t.Parallel()

	netErr := errors.New("connection refused")

	tests := []struct {
		name       string
		setupMocks func(client *testdata.Client)
		wantErr    string
	}{
		{
			name: "error while fetching the MR",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).Return(nil, netErr).Once()
			},
			wantErr: `failed to fetch MR "ns/proj"/"7"`,
		},
		{
			name: "error while fetching the changes",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
				client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).Return(nil, netErr).Once()
			},
			wantErr: `failed to fetch MR changes "ns/proj"/"7"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()
			tt.setupMocks(client)

			tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
			res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

			require.Error(t, err)
			require.ErrorIs(t, err, netErr)
			require.Contains(t, err.Error(), tt.wantErr)
			require.Nil(t, res)
		})
	}
}

func TestGitLabMRExec_JSONError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("stream broken")

	tests := []struct {
		name       string
		setupMocks func(client *testdata.Client)
		wantErr    string
	}{
		{
			name: "invalid JSON for the MR",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, `not json`), nil).Once()
			},
			wantErr: "failed to parse response",
		},
		{
			name: "empty body for the MR",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, ``), nil).Once()
			},
			wantErr: "failed to parse response",
		},
		{
			name: "invalid JSON for the changes",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
				client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
					Return(newMockHTTPResponse(http.StatusOK, `[1, 2`), nil).Once()
			},
			wantErr: "failed to parse changes response",
		},
		{
			name: "read error for the MR body",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(&http.Response{StatusCode: http.StatusOK, Body: errReadCloser{err: readErr}}, nil).Once()
			},
			wantErr: "failed to read response body",
		},
		{
			name: "read error for the changes body",
			setupMocks: func(client *testdata.Client) {
				client.EXPECT().Do(mock.MatchedBy(isMRRequest)).Return(newMockHTTPResponse(http.StatusOK, `{}`), nil).Once()
				client.EXPECT().Do(mock.MatchedBy(isChangesRequest)).
					Return(&http.Response{StatusCode: http.StatusOK, Body: errReadCloser{err: readErr}}, nil).Once()
			},
			wantErr: "failed to read changes response body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()
			tt.setupMocks(client)

			tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
			res, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.Nil(t, res)
		})
	}
}

func TestGitLabMRExec_Timeout(t *testing.T) {
	t.Parallel()

	t.Run("requests carry a deadline of at most one minute", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		factory := testdata.NewClientFactory(t)
		client := testdata.NewClient(t)
		factory.EXPECT().Get().Return(client).Once()
		factory.EXPECT().Put(client).Return().Once()

		client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
			RunAndReturn(func(req *http.Request) (*http.Response, error) {
				deadline, ok := req.Context().Deadline()
				require.True(t, ok, "request context has no deadline")
				require.LessOrEqual(t, time.Until(deadline), gitlabDefaultTimeout)
				return newMockHTTPResponse(http.StatusOK, `{}`), nil
			}).Twice()

		tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
		_, err := tool.Exec(context.Background(), &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})
		require.NoError(t, err)
	})

	t.Run("a cancelled context stops the fetch", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		factory := testdata.NewClientFactory(t)
		client := testdata.NewClient(t)
		factory.EXPECT().Get().Return(client).Once()
		factory.EXPECT().Put(client).Return().Once()

		client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
			RunAndReturn(func(req *http.Request) (*http.Response, error) {
				return nil, req.Context().Err()
			}).Once()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		tool := New(newTestConfig(t, testBaseURL), printer.New(&buf, false), factory)
		res, err := tool.Exec(ctx, &GitLabMRParams{ProjectPath: "ns/proj", MRID: "7"})

		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, res)
	})
}
