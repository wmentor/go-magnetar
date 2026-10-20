package web

import (
	"testing"
)

func TestExtractGitHubTreeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantBranch string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "basic tree URL",
			input:      "https://github.com/golang/go/tree/master",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "tree URL with path",
			input:      "https://github.com/golang/go/tree/master/src/cmd",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src/cmd",
			wantErr:    false,
		},
		{
			name:       "tree URL with trailing slash",
			input:      "https://github.com/golang/go/tree/master/",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "tree URL with query parameters on path",
			input:      "https://github.com/golang/go/tree/master/src?foo=bar",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
			wantErr:    false,
		},
		{
			name:       "tree URL with query on branch (no path)",
			input:      "https://github.com/golang/go/tree/master?foo=bar",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "commits URL without path",
			input:      "https://github.com/golang/go/commits/master",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "commits URL with path",
			input:      "https://github.com/golang/go/commits/master/src",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
			wantErr:    false,
		},
		{
			name:    "blob URL is not a tree URL",
			input:   "https://github.com/golang/go/blob/master/README.md",
			wantErr: true,
		},
		{
			name:    "issue URL is not a tree URL",
			input:   "https://github.com/golang/go/issues/1234",
			wantErr: true,
		},
		{
			name:       "tree URL with fragment",
			input:      "https://github.com/golang/go/tree/master/src#readme",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
		},
		{
			name:       "tree URL on www.github.com",
			input:      "https://www.github.com/golang/go/tree/master/src",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/tree/master",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/tree/master",
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
			owner, repo, branch, path, err := extractGitHubTreeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubTreeURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || branch != tt.wantBranch || path != tt.wantPath {
				t.Errorf("extractGitHubTreeURL(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tt.input, owner, repo, branch, path, tt.wantOwner, tt.wantRepo, tt.wantBranch, tt.wantPath)
			}
		})
	}
}

func TestExtractGitHubFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantBranch string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "basic blob URL",
			input:      "https://github.com/golang/go/blob/master/README.md",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with nested path",
			input:      "https://github.com/golang/go/blob/master/src/cmd/go/main.go",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "src/cmd/go/main.go",
			wantErr:    false,
		},
		{
			name:       "blob URL with trailing slash",
			input:      "https://github.com/golang/go/blob/master/README.md/",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with query parameters",
			input:      "https://github.com/golang/go/blob/master/README.md?raw=1",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with fragment",
			input:      "https://github.com/golang/go/blob/master/README.md#L10",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:    "blob URL without file path",
			input:   "https://github.com/golang/go/blob/master",
			wantErr: true,
		},
		{
			name:    "tree URL is not a blob URL",
			input:   "https://github.com/golang/go/tree/master/src",
			wantErr: true,
		},
		{
			name:       "blob URL with line range fragment",
			input:      "https://github.com/golang/go/blob/master/src/cmd/go/main.go#L10-L20",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "src/cmd/go/main.go",
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/blob/master/README.md",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/blob/master/README.md",
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
			owner, repo, branch, file, err := extractGitHubFileURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubFileURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || branch != tt.wantBranch || file != tt.wantFile {
				t.Errorf("extractGitHubFileURL(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tt.input, owner, repo, branch, file, tt.wantOwner, tt.wantRepo, tt.wantBranch, tt.wantFile)
			}
		})
	}
}

func TestExtractGitHubIssueURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantNum   string
		wantErr   bool
	}{
		{
			name:      "basic issue URL",
			input:     "https://github.com/golang/go/issues/1234",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "1234",
			wantErr:   false,
		},
		{
			name:      "issue URL with trailing slash",
			input:     "https://github.com/golang/go/issues/1234/",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "1234",
			wantErr:   false,
		},
		{
			name:      "issue URL with query parameters",
			input:     "https://github.com/golang/go/issues/1234?foo=bar",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "1234",
			wantErr:   false,
		},
		{
			name:      "issue URL with fragment",
			input:     "https://github.com/golang/go/issues/1234#issuecomment-1",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "1234",
			wantErr:   false,
		},
		{
			name:    "pull request URL is not an issue URL",
			input:   "https://github.com/golang/go/pull/1234",
			wantErr: true,
		},
		{
			name:    "issues list without number",
			input:   "https://github.com/golang/go/issues",
			wantErr: true,
		},
		{
			name:      "issue URL without scheme",
			input:     "github.com/golang/go/issues/1234",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "1234",
		},
		{
			name:    "issue number followed by other characters",
			input:   "https://github.com/golang/go/issues/1234abc",
			wantErr: true,
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/issues/1234",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/issues/1234",
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
			owner, repo, num, err := extractGitHubIssueURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubIssueURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || num != tt.wantNum {
				t.Errorf("extractGitHubIssueURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, owner, repo, num, tt.wantOwner, tt.wantRepo, tt.wantNum)
			}
		})
	}
}

func TestExtractGitHubMilestoneURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantNum   string
		wantErr   bool
	}{
		{
			name:      "basic milestone URL",
			input:     "https://github.com/golang/go/milestone/3",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "3",
			wantErr:   false,
		},
		{
			name:      "milestone URL with trailing slash",
			input:     "https://github.com/golang/go/milestone/3/",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "3",
			wantErr:   false,
		},
		{
			name:      "milestone URL with query parameters",
			input:     "https://github.com/golang/go/milestone/3?closed=1",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "3",
			wantErr:   false,
		},
		{
			name:      "milestone URL with fragment",
			input:     "https://github.com/golang/go/milestone/3#partial-issues",
			wantOwner: "golang",
			wantRepo:  "go",
			wantNum:   "3",
			wantErr:   false,
		},
		{
			name:    "milestones list path (plural) is not matched",
			input:   "https://github.com/golang/go/milestones/3",
			wantErr: true,
		},
		{
			name:    "milestone without number",
			input:   "https://github.com/golang/go/milestone",
			wantErr: true,
		},
		{
			name:    "issue URL is not a milestone URL",
			input:   "https://github.com/golang/go/issues/3",
			wantErr: true,
		},
		{
			name:    "milestone number followed by other characters",
			input:   "https://github.com/golang/go/milestone/3abc",
			wantErr: true,
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/milestone/3",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/milestone/3",
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
			owner, repo, num, err := extractGitHubMilestoneURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubMilestoneURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || num != tt.wantNum {
				t.Errorf("extractGitHubMilestoneURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, owner, repo, num, tt.wantOwner, tt.wantRepo, tt.wantNum)
			}
		})
	}
}

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
			got := isGitHubPullRequestURL(tt.input)
			if got != tt.want {
				t.Errorf("isGitHubPullRequestURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

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
			owner, repo, tag, err := extractGitHubReleaseURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubReleaseURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || tag != tt.wantTag {
				t.Errorf("extractGitHubReleaseURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, owner, repo, tag, tt.wantOwner, tt.wantRepo, tt.wantTag)
			}
		})
	}
}
