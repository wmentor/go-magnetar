package repo

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	content := "version: \"1.0\"\ngitlab:\n  base_url: \"https://gitlab.example.com\"\n  api_key: \"test-token\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newMockFactory(t *testing.T, client *testdata.Client) *testdata.ClientFactory {
	t.Helper()

	clientFactory := testdata.NewClientFactory(t)
	clientFactory.On("Get").Return(client)
	clientFactory.On("Put", client).Return()
	return clientFactory
}

func TestGitLabRepoNew(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)

	tool := New(cfg, p, clientFactory)
	if tool == nil {
		t.Fatal("New() returned nil")
	}

	if tool.Name() != "gitlab_repo" {
		t.Errorf("Name() = %s, want gitlab_repo", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestGitLabRepoParamsValidation(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)

	tool := New(cfg, p, clientFactory)

	result, err := tool.Exec(t.Context(), nil)
	if err == nil {
		t.Error("Exec() with nil params should return an error")
	}
	if result != nil {
		t.Log("Exec() returned non-nil result with nil params (expected)")
	}
}

func TestGitLabRepoExec_Success(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	client := testdata.NewClient(t)
	clientFactory := newMockFactory(t, client)

	repoJSON := `{
		"id": 1,
		"name": "test-project",
		"path_with_namespace": "namespace/test-project",
		"description": "A test project",
		"web_url": "https://gitlab.example.com/namespace/test-project",
		"avatar_url": "",
		"star_count": 42,
		"forks_count": 10,
		"open_issues_count": 5,
		"default_branch": "main",
		"visibility": "public",
		"ssh_url_to_repo": "git@gitlab.example.com:namespace/test-project.git",
		"http_url_to_repo": "https://gitlab.example.com/namespace/test-project.git",
		"last_activity_at": "2024-01-01T00:00:00Z"
	}`

	client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == http.MethodGet &&
			req.URL.String() == "https://gitlab.example.com/api/v4/projects/namespace%2Ftest-project" &&
			req.Header.Get("Authorization") == "Bearer test-token" &&
			req.Header.Get("Content-Type") == "application/json"
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(repoJSON)),
	}, nil)

	tool := New(cfg, printer.GetDefault(), clientFactory)
	res, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: "namespace/test-project"})
	if err != nil {
		t.Fatalf("Exec() unexpected error: %v", err)
	}

	resp, ok := res.(*tools.Response)
	if !ok {
		t.Fatalf("Exec() returned %T, want *tools.Response", res)
	}
	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}
	out := resp.Result
	for _, want := range []string{
		"# test-project",
		"**Project Path:** namespace/test-project",
		"**Default Branch:** main",
		"**Visibility:** public",
		"**Forks:** 10 | **Stars:** 42 | **Open Issues:** 5",
		"**Description:**\nA test project",
		"**Web URL:** https://gitlab.example.com/namespace/test-project",
		"**SSH URL:** git@gitlab.example.com:namespace/test-project.git",
		"**HTTP URL:** https://gitlab.example.com/namespace/test-project.git",
		"**Last Activity:** 2024-01-01T00:00:00Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestGitLabRepoExec_EmptyFields(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	client := testdata.NewClient(t)
	clientFactory := newMockFactory(t, client)

	repoJSON := `{
		"id": 2,
		"name": "minimal",
		"path_with_namespace": "org/minimal",
		"description": "",
		"web_url": "https://gitlab.example.com/org/minimal",
		"star_count": 0,
		"forks_count": 0,
		"open_issues_count": 0,
		"default_branch": "master",
		"visibility": "private",
		"ssh_url_to_repo": "git@gitlab.example.com:org/minimal.git",
		"http_url_to_repo": "https://gitlab.example.com/org/minimal.git",
		"last_activity_at": ""
	}`

	client.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(repoJSON)),
	}, nil)

	tool := New(cfg, printer.GetDefault(), clientFactory)
	res, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: "org/minimal"})
	if err != nil {
		t.Fatalf("Exec() unexpected error: %v", err)
	}

	out := res.(*tools.Response).Result
	if strings.Contains(out, "**Description:**") {
		t.Errorf("expected no Description section when description is empty, got:\n%s", out)
	}
	if strings.Contains(out, "**Last Activity:**") {
		t.Errorf("expected no Last Activity section when last_activity_at is empty, got:\n%s", out)
	}
	if !strings.Contains(out, "**Forks:** 0 | **Stars:** 0 | **Open Issues:** 0") {
		t.Errorf("expected zero counts in output, got:\n%s", out)
	}
}

func TestGitLabRepoExec_HTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "not found", statusCode: http.StatusNotFound},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "internal server error", statusCode: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			client := testdata.NewClient(t)
			clientFactory := newMockFactory(t, client)

			client.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: tc.statusCode,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil)

			tool := New(cfg, printer.GetDefault(), clientFactory)
			_, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: "org/repo"})
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tc.statusCode)
			}
		})
	}
}

func TestGitLabRepoExec_NetworkError(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	client := testdata.NewClient(t)
	clientFactory := newMockFactory(t, client)

	client.On("Do", mock.Anything).Return((*http.Response)(nil), errors.New("connection refused"))

	tool := New(cfg, printer.GetDefault(), clientFactory)
	_, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: "org/repo"})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitLabRepoExec_JSONError(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	client := testdata.NewClient(t)
	clientFactory := newMockFactory(t, client)

	client.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{invalid json")),
	}, nil)

	tool := New(cfg, printer.GetDefault(), clientFactory)
	_, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: "org/repo"})
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitLabRepoExec_URLEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		projectPath string
		wantEscaped string
	}{
		{
			name:        "nested subgroup",
			projectPath: "group/subgroup/project",
			wantEscaped: "group%2Fsubgroup%2Fproject",
		},
		{
			name:        "special characters",
			projectPath: "my group/my+project",
			wantEscaped: "my%20group%2Fmy+project",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			client := testdata.NewClient(t)
			clientFactory := newMockFactory(t, client)

			wantURL := "https://gitlab.example.com/api/v4/projects/" + tc.wantEscaped
			client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.String() == wantURL
			})).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name":"p"}`)),
			}, nil)

			tool := New(cfg, printer.GetDefault(), clientFactory)
			if _, err := tool.Exec(t.Context(), &GitLabRepoParams{ProjectPath: tc.projectPath}); err != nil {
				t.Fatalf("Exec() unexpected error: %v", err)
			}
		})
	}
}
