package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubRepoURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    [2]string // owner, repo
		wantErr bool
	}{
		{
			name:  "full https URL",
			input: "https://github.com/owner/repo",
			want:  [2]string{"owner", "repo"},
		},
		{
			name:  "full http URL",
			input: "http://github.com/owner/repo",
			want:  [2]string{"owner", "repo"},
		},
		{
			name:  "URL without protocol",
			input: "github.com/owner/repo",
			want:  [2]string{"owner", "repo"},
		},
		{
			name:  "with .git suffix",
			input: "https://github.com/owner/repo.git",
			want:  [2]string{"owner", "repo"},
		},
		{
			name:  "with trailing slash",
			input: "https://github.com/owner/repo/",
			want:  [2]string{"owner", "repo"},
		},
		{
			name:  "owner with dashes",
			input: "https://github.com/my-owner/my-repo",
			want:  [2]string{"my-owner", "my-repo"},
		},
		{
			name:  "owner with underscores",
			input: "https://github.com/my_owner/my_repo",
			want:  [2]string{"my_owner", "my_repo"},
		},
		{
			name:  "owner with numbers",
			input: "https://github.com/owner123/repo456",
			want:  [2]string{"owner123", "repo456"},
		},
		{
			name:  "capital letters in repo",
			input: "https://github.com/owner/MyRepo",
			want:  [2]string{"owner", "MyRepo"},
		},
		{
			name:    "invalid URL - no repo",
			input:   "https://github.com/owner",
			want:    [2]string{"", ""},
			wantErr: true,
		},
		{
			name:    "invalid URL - only protocol",
			input:   "https://github.com",
			want:    [2]string{"https:", ""},
			wantErr: false,
		},
		{
			name:    "invalid URL - empty string",
			input:   "",
			want:    [2]string{"", ""},
			wantErr: true,
		},
		{
			name:    "invalid URL - single path",
			input:   "github.com/owner",
			want:    [2]string{"", ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			owner, repo, err := parseGitHubRepoURL(tt.input)
			if tt.wantErr {
				require.Error(t, err, "parseGitHubRepoURL() should return error")
				return
			}

			require.NoError(t, err, "parseGitHubRepoURL() should not return error")
			assert.Equal(t, tt.want[0], owner, "owner mismatch")
			assert.Equal(t, tt.want[1], repo, "repo mismatch")
		})
	}
}
