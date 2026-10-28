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
