package web

import (
	"testing"
)

func TestIsGitHubPullRequestURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid PR URL",
			input: "https://github.com/prometheus-community/pgbouncer_exporter/pull/309",
			want:  true,
		},
		{
			name:  "valid PR URL with /files",
			input: "https://github.com/owner/repo/pull/123/files",
			want:  true,
		},
		{
			name:  "valid PR URL with /commits",
			input: "https://github.com/owner/repo/pull/123/commits",
			want:  true,
		},
		{
			name:  "PR URL with query params",
			input: "https://github.com/owner/repo/pull/123?expand=1",
			want:  true,
		},
		{
			name:  "not a GitHub URL",
			input: "https://gitlab.com/owner/repo/pull/123",
			want:  false,
		},
		{
			name:  "PR URL with /tree/ before /pull/",
			input: "https://github.com/owner/tree/repo/pull/123",
			want:  false,
		},
		{
			name:  "owner named 'pull'",
			input: "https://github.com/pull/repo/pull/123",
			want:  false,
		},
		{
			name:  "repo named 'pull'",
			input: "https://github.com/user/pull/pull/123",
			want:  false,
		},
		{
			name:  "just /pull/ without number",
			input: "https://github.com/owner/repo/pull",
			want:  false,
		},
		{
			name:  "issues URL is not PR",
			input: "https://github.com/owner/repo/issues/123",
			want:  false,
		},
		{
			name:  "blob URL is not PR",
			input: "https://github.com/owner/repo/blob/master/file.go",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsGitHubPullRequestURL(tt.input)
			if got != tt.want {
				t.Errorf("IsGitHubPullRequestURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsGitHubPullRequestsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid PRs list URL",
			input: "https://github.com/prometheus-community/pgbouncer_exporter/pulls",
			want:  true,
		},
		{
			name:  "valid PRs list URL with query",
			input: "https://github.com/owner/repo/pulls?q=is%3Aopen+is%3Apr",
			want:  true,
		},
		{
			name:  "not a GitHub URL",
			input: "https://gitlab.com/owner/repo/pulls",
			want:  false,
		},
		{
			name:  "URL with /pull/ (single PR)",
			input: "https://github.com/owner/repo/pull/123",
			want:  false,
		},
		{
			name:  "URL with /pull/ and number",
			input: "https://github.com/owner/repo/pull/123",
			want:  false,
		},
		{
			name:  "issues URL is not PRs list",
			input: "https://github.com/owner/repo/issues",
			want:  false,
		},
		{
			name:  "tree URL is not PRs list",
			input: "https://github.com/owner/repo/tree/main",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsGitHubPullRequestsURL(tt.input)
			if got != tt.want {
				t.Errorf("IsGitHubPullRequestsURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractGitHubPullRequestsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  struct {
			owner string
			repo  string
		}
		wantErr bool
	}{
		{
			name:  "valid PRs list URL",
			input: "https://github.com/prometheus-community/pgbouncer_exporter/pulls",
			want: struct {
				owner string
				repo  string
			}{
				owner: "prometheus-community",
				repo:  "pgbouncer_exporter",
			},
			wantErr: false,
		},
		{
			name:  "PRs list URL with query params",
			input: "https://github.com/owner/repo/pulls?q=is%3Aopen",
			want: struct {
				owner string
				repo  string
			}{
				owner: "owner",
				repo:  "repo",
			},
			wantErr: false,
		},
		{
			name:  "PRs list URL with fragment",
			input: "https://github.com/owner/repo/pulls#issues-123",
			want: struct {
				owner string
				repo  string
			}{
				owner: "owner",
				repo:  "repo",
			},
			wantErr: false,
		},
		{
			name:  "not a PRs list URL",
			input: "https://github.com/owner/repo/pull/123",
			want: struct {
				owner string
				repo  string
			}{
				owner: "",
				repo:  "",
			},
			wantErr: true,
		},
		{
			name:  "not a GitHub URL",
			input: "https://gitlab.com/owner/repo/pulls",
			want: struct {
				owner string
				repo  string
			}{
				owner: "",
				repo:  "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, repo, err := ExtractGitHubPullRequestsURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubPullRequestsURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.want.owner {
					t.Errorf("ExtractGitHubPullRequestsURL(%q) owner = %q, want %q", tt.input, owner, tt.want.owner)
				}
				if repo != tt.want.repo {
					t.Errorf("ExtractGitHubPullRequestsURL(%q) repo = %q, want %q", tt.input, repo, tt.want.repo)
				}
			}
		})
	}
}
