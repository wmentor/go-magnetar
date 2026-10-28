package web

import (
	"testing"
)

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
			owner, repo, num, err := ExtractGitHubIssueURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubIssueURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || num != tt.wantNum {
				t.Errorf("ExtractGitHubIssueURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
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
			owner, repo, num, err := ExtractGitHubMilestoneURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubMilestoneURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || num != tt.wantNum {
				t.Errorf("ExtractGitHubMilestoneURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, owner, repo, num, tt.wantOwner, tt.wantRepo, tt.wantNum)
			}
		})
	}
}
