package web

import (
	"testing"
)

func TestExtractGitHubReleaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantTag   string
		wantErr   bool
	}{
		{
			name:      "basic release URL",
			input:     "https://github.com/wmentor/go-magnetar/releases/tag/v1.6.0",
			wantOwner: "wmentor",
			wantRepo:  "go-magnetar",
			wantTag:   "v1.6.0",
			wantErr:   false,
		},
		{
			name:      "release URL with trailing slash",
			input:     "https://github.com/wmentor/go-magnetar/releases/tag/v1.6.0/",
			wantOwner: "wmentor",
			wantRepo:  "go-magnetar",
			wantTag:   "v1.6.0",
			wantErr:   false,
		},
		{
			name:      "release URL with query parameters",
			input:     "https://github.com/wmentor/go-magnetar/releases/tag/v1.6.0?foo=bar",
			wantOwner: "wmentor",
			wantRepo:  "go-magnetar",
			wantTag:   "v1.6.0",
			wantErr:   false,
		},
		{
			name:      "release URL with fragment",
			input:     "https://github.com/wmentor/go-magnetar/releases/tag/v1.6.0#release-assets",
			wantOwner: "wmentor",
			wantRepo:  "go-magnetar",
			wantTag:   "v1.6.0",
			wantErr:   false,
		},
		{
			name:    "release URL without tag",
			input:   "https://github.com/wmentor/go-magnetar/releases/tag",
			wantErr: true,
		},
		{
			name:    "releases list URL is not matched",
			input:   "https://github.com/wmentor/go-magnetar/releases",
			wantErr: true,
		},
		{
			name:    "issue URL is not a release URL",
			input:   "https://github.com/wmentor/go-magnetar/issues/123",
			wantErr: true,
		},
		{
			name:      "release URL without scheme",
			input:     "github.com/wmentor/go-magnetar/releases/tag/v1.6.0",
			wantOwner: "wmentor",
			wantRepo:  "go-magnetar",
			wantTag:   "v1.6.0",
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/wmentor/go-magnetar/releases/tag/v1.6.0",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/wmentor/go-magnetar/releases/tag/v1.6.0",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, repo, tag, err := ExtractGitHubReleaseURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubReleaseURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || tag != tt.wantTag {
				t.Errorf("ExtractGitHubReleaseURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, owner, repo, tag, tt.wantOwner, tt.wantRepo, tt.wantTag)
			}
		})
	}
}
