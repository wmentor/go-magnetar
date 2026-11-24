package gitlabplug

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func newGitLabPlugin(t *testing.T, baseURL string) *Plugin {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	body := fmt.Sprintf("version: \"1.0\"\ngitlab:\n  enable: true\n  base_url: %s\n  api_key: test-key\n", baseURL)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return &Plugin{state: &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: requester.NewFactory(cfg),
	}}
}

func TestFetchGitLabFile(t *testing.T) {
	t.Parallel()

	var gotPath, gotRef, gotAuth string
	content := base64.StdEncoding.EncodeToString([]byte("hello from gitlab"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotRef, gotAuth = r.URL.EscapedPath(), r.URL.Query().Get("ref"), r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"encoding":"base64","content":%q}`, content)
	}))
	defer srv.Close()

	p := newGitLabPlugin(t, srv.URL)
	got, err := p.fetchGitLabFile(context.Background(), "group/project", "feature/x", "dir/file name.txt")

	require.NoError(t, err)
	assert.Equal(t, "hello from gitlab", got)
	assert.Equal(t, "Bearer test-key", gotAuth)
	assert.Equal(t, "feature/x", gotRef)
	assert.Contains(t, gotPath, "/api/v4/projects/group%2Fproject/repository/files/")
	assert.Contains(t, gotPath, "dir%2Ffile%20name.txt")
}

func TestFetchGitLabFileDefaultsTheBranch(t *testing.T) {
	t.Parallel()

	var gotRef string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRef = r.URL.Query().Get("ref")
		fmt.Fprintf(w, `{"encoding":"base64","content":%q}`, base64.StdEncoding.EncodeToString([]byte("ok")))
	}))
	defer srv.Close()

	p := newGitLabPlugin(t, srv.URL)
	_, err := p.fetchGitLabFile(context.Background(), "group/project", "", "README.md")

	require.NoError(t, err)
	assert.Equal(t, "main", gotRef)
}

func TestFetchGitLabFileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "http 404", status: http.StatusNotFound, body: `{}`, wantErr: "status 404"},
		{name: "invalid json", status: http.StatusOK, body: `not json`, wantErr: "failed to parse response"},
		{name: "no content", status: http.StatusOK, body: `{"encoding":"base64"}`, wantErr: "has no content"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()

			p := newGitLabPlugin(t, srv.URL)
			_, err := p.fetchGitLabFile(context.Background(), "group/project", "main", "file.txt")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
