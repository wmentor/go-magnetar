package githubplug

import (
	"context"
	"errors"
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

func newGitHubPlugin(t *testing.T, baseURL string) *Plugin {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	body := fmt.Sprintf("version: \"1.0\"\ngithub:\n  enable: true\n  base_url: %s\n  api_key: test-key\n", baseURL)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return &Plugin{state: &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: requester.NewFactory(cfg),
	}}
}

func TestExecWebFetchRoutesGitHubURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		path string
		auth bool
	}{
		{name: "file", url: "https://github.com/acme/widgets/blob/main/src/app.go", path: "/repos/acme/widgets/contents/src/app.go", auth: true},
		{name: "pull request", url: "https://github.com/acme/widgets/pull/42", path: "/repos/acme/widgets/pulls/42", auth: true},
		{name: "pull requests list", url: "https://github.com/acme/widgets/pulls", path: "/repos/acme/widgets/pulls", auth: true},
		{name: "tree", url: "https://github.com/acme/widgets/tree/main/src", path: "/repos/acme/widgets/git/trees/main", auth: true},
		{name: "repository", url: "https://github.com/acme/widgets", path: "/repos/acme/widgets", auth: true},
		{name: "issue", url: "https://github.com/acme/widgets/issues/7", path: "/repos/acme/widgets/issues/7", auth: true},
		{name: "issues list", url: "https://github.com/acme/widgets/issues", path: "/repos/acme/widgets/issues", auth: true},
		{name: "milestone", url: "https://github.com/acme/widgets/milestone/3", path: "/repos/acme/widgets/milestones/3", auth: true},
		{name: "release", url: "https://github.com/acme/widgets/releases/tag/v1.2.3", path: "/repos/acme/widgets/releases/tags/v1.2.3", auth: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPaths []string
			var seenAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Some fetches issue several requests (PR files/reviews, repository readme).
				seenPaths = append(seenPaths, r.URL.Path)
				if seenAuth == "" {
					seenAuth = r.Header.Get("Authorization")
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			}))
			defer srv.Close()

			p := newGitHubPlugin(t, srv.URL)
			// The result may not parse `{}`; this test cares about routing.
			_, _ = p.execWebFetch(context.Background(), tt.url)

			assert.Contains(t, seenPaths, tt.path)
			if tt.auth {
				assert.Equal(t, "Bearer test-key", seenAuth)
			}
		})
	}
}

func TestExecWebFetchIgnoresNonGitHubURLs(t *testing.T) {
	t.Parallel()

	p := newGitHubPlugin(t, "http://127.0.0.1:1")
	_, err := p.execWebFetch(context.Background(), "https://example.com/readme")
	require.True(t, errors.Is(err, plugin.ErrNoWebProcessor), "got %v", err)
}

func TestExecWebFetchUsesConfiguredBaseURL(t *testing.T) {
	t.Parallel()

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	p := newGitHubPlugin(t, srv.URL+"/")
	_, _ = p.execWebFetch(context.Background(), "https://github.com/acme/widgets/issues/1")

	assert.True(t, called, "the request must go to the configured base_url")
}
