package tree

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

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

func TestGitLabTree_NameAndDescription(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, "https://gitlab.example.com")
	var buf strings.Builder
	p := printer.New(&buf, false)

	factory := testdata.NewClientFactory(t)
	tool := New(cfg, p, factory)

	require.Equal(t, "gitlab_tree", tool.Name())
	require.Contains(t, tool.Description(), "repository")
	require.Contains(t, tool.Description(), "root")
}

func TestGitLabTree_ParamsValidation(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, "https://gitlab.example.com")
	var buf strings.Builder
	p := printer.New(&buf, false)

	factory := testdata.NewClientFactory(t)
	tool := New(cfg, p, factory)

	_, err := tool.Exec(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")
}

func TestGitLabTree_Exec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     *GitLabTreeParams
		baseURL    string
		setupMocks func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory)
		wantErr    string
		check      func(t *testing.T, result string)
	}{
		{
			name: "list root directory with files and subdirectories",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				payload := `[
					{"name":"README.md","path":"README.md","type":"blob","mode":"100644"},
					{"name":"internal","path":"internal","type":"tree","mode":"040000"},
					{"name":"main.go","path":"main.go","type":"blob","mode":"100644"},
					{"name":"docs","path":"docs","type":"tree","mode":"040000"}
				]`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://gitlab.example.com/api/v4/projects/namespace%2Fproject/repository/tree?ref=main", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						return newMockHTTPResponse(http.StatusOK, payload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "**Repository root contents**")
				require.Contains(t, result, "| README.md | blob | file |")
				require.Contains(t, result, "| internal | tree | dir |")
				require.Contains(t, result, "| main.go | blob | file |")
				require.Contains(t, result, "| docs | tree | dir |")
			},
		},
		{
			name: "list specific subdirectory",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "develop",
				Path:        "internal/tools",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				payload := `[
					{"name":"tool1.go","path":"internal/tools/tool1.go","type":"blob","mode":"100644"},
					{"name":"tool2.go","path":"internal/tools/tool2.go","type":"blob","mode":"100644"}
				]`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://gitlab.example.com/api/v4/projects/namespace%2Fproject/repository/tree?ref=develop&path=internal%2Ftools", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						return newMockHTTPResponse(http.StatusOK, payload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "**Contents of internal/tools**")
				require.Contains(t, result, "| tool1.go | blob | file |")
				require.Contains(t, result, "| tool2.go | blob | file |")
			},
		},
		{
			name: "default branch to main when empty",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				payload := `[{"name":"main.go","path":"main.go","type":"blob","mode":"100644"}]`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Contains(t, req.URL.String(), "ref=main")
						return newMockHTTPResponse(http.StatusOK, payload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "| main.go | blob | file |")
			},
		},
		{
			name: "list directory with special characters in path",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "feature",
				Path:        "path/with spaces",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				payload := `[{"name":"file.txt","path":"path/with spaces/file.txt","type":"blob","mode":"100644"}]`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Contains(t, req.URL.String(), "path=path%2Fwith%20spaces")
						return newMockHTTPResponse(http.StatusOK, payload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "**Contents of path/with spaces**")
			},
		},
		{
			name: "HTTP transport error",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(nil, errors.New("connection refused")).Once()
			},
			wantErr: "failed to fetch tree",
		},
		{
			name: "non-200 HTTP status 404",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusNotFound, `{"message":"404 Project Not Found"}`), nil).Once()
			},
			wantErr: "returned status 404",
		},
		{
			name: "invalid JSON response",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, `{not-valid-json`), nil).Once()
			},
			wantErr: "failed to parse response",
		},
		{
			name: "empty response body",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, ""), nil).Once()
			},
			wantErr: "unexpected end of JSON",
		},
		{
			name: "response body read error",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "https://gitlab.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(&http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body: struct {
							io.Reader
							io.Closer
						}{
							Reader: io.MultiReader(strings.NewReader("[{}]"), errReadCloser{err: errors.New("stream broken")}),
							Closer: io.NopCloser(nil),
						},
					}, nil).Once()
			},
			wantErr: "failed to read response body",
		},
		{
			name: "malformed base URL returns request creation error",
			params: &GitLabTreeParams{
				ProjectPath: "namespace/project",
				Branch:      "main",
				Path:        "",
			},
			baseURL: "://invalid-url",
			setupMocks: func(t *testing.T, _ *testdata.Client, factory *testdata.ClientFactory) {
				t.Helper()
			},
			wantErr: "failed to create request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, tt.baseURL)
			var buf strings.Builder
			p := printer.New(&buf, false)

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()
			if tt.setupMocks != nil {
				tt.setupMocks(t, client, factory)
			}

			tool := New(cfg, p, factory)
			res, err := tool.Exec(context.Background(), tt.params)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			resp, ok := res.(*tools.Response)
			require.True(t, ok, "expected *tools.Response, got %T", res)
			require.Equal(t, tools.StatusOK, resp.Status)
			if tt.check != nil {
				tt.check(t, resp.Result)
			}
		})
	}
}

func TestFormatTreeMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		treeItems []struct {
			Name   string `json:"name"`
			Path   string `json:"path"`
			Type   string `json:"type"`
			Mode   string `json:"mode"`
			Target string `json:"target"`
		}
		path     string
		expected string
	}{
		{
			name: "root directory with mixed content",
			treeItems: []struct {
				Name   string `json:"name"`
				Path   string `json:"path"`
				Type   string `json:"type"`
				Mode   string `json:"mode"`
				Target string `json:"target"`
			}{
				{"README.md", "README.md", "blob", "100644", ""},
				{"internal", "internal", "tree", "040000", ""},
				{"main.go", "main.go", "blob", "100644", ""},
			},
			path: "",
			expected: "**Repository root contents**\n\n" +
				"| Name | Type | Size |\n" +
				"|------|------|------|\n" +
				"| README.md | blob | file |\n" +
				"| internal | tree | dir |\n" +
				"| main.go | blob | file |\n\n",
		},
		{
			name: "subdirectory with files",
			treeItems: []struct {
				Name   string `json:"name"`
				Path   string `json:"path"`
				Type   string `json:"type"`
				Mode   string `json:"mode"`
				Target string `json:"target"`
			}{
				{"tool1.go", "internal/tools/tool1.go", "blob", "100644", ""},
				{"tool2.go", "internal/tools/tool2.go", "blob", "100644", ""},
			},
			path: "internal/tools",
			expected: "**Contents of internal/tools**\n\n" +
				"| Name | Type | Size |\n" +
				"|------|------|------|\n" +
				"| tool1.go | blob | file |\n" +
				"| tool2.go | blob | file |\n\n",
		},
		{
			name: "empty directory",
			treeItems: []struct {
				Name   string `json:"name"`
				Path   string `json:"path"`
				Type   string `json:"type"`
				Mode   string `json:"mode"`
				Target string `json:"target"`
			}{},
			path: "",
			expected: "**Repository root contents**\n\n" +
				"| Name | Type | Size |\n" +
				"|------|------|------|\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := formatTreeMarkdown(tt.treeItems, tt.path)
			require.Equal(t, tt.expected, result)
		})
	}
}
