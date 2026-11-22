// Package github unit tests for the GitHub tools.
//
// This file contains comprehensive unit tests for the formatOpenIssuesListMarkdown method
// which formats GitHub open issues list as Markdown. The tests cover:
//   - Empty list of issues
//   - Single issue
//   - Multiple issues
//   - Issues sorted by creation date descending (newest first)
//
// Tests follow table-driven patterns with t.Parallel() for concurrent execution.

package github

import (
	"testing"
)

func TestFormatOpenIssuesListMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		issues []IssueListItem
		want   string
	}{
		{
			name:   "empty list of issues",
			issues: []IssueListItem{},
			want:   "# Open Issues\n\nNo open issues found.\n\n",
		},
		{
			name: "single issue",
			issues: []IssueListItem{
				{
					Number: 1,
					URL:    "https://github.com/test/repo/issues/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "testuser"},
					Title:   "Add new feature",
					Body:    "This issue adds a new feature.",
					Created: "2024-01-01T00:00:00Z",
					State:   "open",
				},
			},
			want: "# Open Issues\n\n**Total Open Issues:** 1\n\n| Issue | Title | Author | Created |\n|-------|-------|--------|----------|\n| [#1](https://github.com/test/repo/issues/1) | Add new feature | @testuser | 2024-01-01T00:00:00Z |\n\n",
		},
		{
			name: "multiple issues",
			issues: []IssueListItem{
				{
					Number: 1,
					URL:    "https://github.com/test/repo/issues/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "alice"},
					Title:   "Add new feature",
					Body:    "This issue adds a new feature.",
					Created: "2024-01-10T10:00:00Z",
					State:   "open",
				},
				{
					Number: 2,
					URL:    "https://github.com/test/repo/issues/2",
					User: struct {
						Login string `json:"login"`
					}{Login: "bob"},
					Title:   "Fix bug in login",
					Body:    "This issue fixes a bug in login.",
					Created: "2024-01-08T08:00:00Z",
					State:   "open",
				},
				{
					Number: 3,
					URL:    "https://github.com/test/repo/issues/3",
					User: struct {
						Login string `json:"login"`
					}{Login: "charlie"},
					Title:   "Update documentation",
					Body:    "This issue updates documentation.",
					Created: "2024-01-05T09:00:00Z",
					State:   "open",
				},
			},
			want: "# Open Issues\n\n**Total Open Issues:** 3\n\n| Issue | Title | Author | Created |\n|-------|-------|--------|----------|\n| [#1](https://github.com/test/repo/issues/1) | Add new feature | @alice | 2024-01-10T10:00:00Z |\n| [#2](https://github.com/test/repo/issues/2) | Fix bug in login | @bob | 2024-01-08T08:00:00Z |\n| [#3](https://github.com/test/repo/issues/3) | Update documentation | @charlie | 2024-01-05T09:00:00Z |\n\n",
		},
		{
			name: "issues sorted by creation date descending",
			issues: []IssueListItem{
				{
					Number: 3,
					URL:    "https://github.com/test/repo/issues/3",
					User: struct {
						Login string `json:"login"`
					}{Login: "charlie"},
					Title:   "Third issue",
					Body:    "Created first but listed third (wrong order)",
					Created: "2024-01-01T00:00:00Z",
					State:   "open",
				},
				{
					Number: 2,
					URL:    "https://github.com/test/repo/issues/2",
					User: struct {
						Login string `json:"login"`
					}{Login: "bob"},
					Title:   "Second issue",
					Body:    "Created second but listed second (correct)",
					Created: "2024-01-02T00:00:00Z",
					State:   "open",
				},
				{
					Number: 1,
					URL:    "https://github.com/test/repo/issues/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "alice"},
					Title:   "First issue",
					Body:    "Created third but listed first (correct, sorted desc)",
					Created: "2024-01-03T00:00:00Z",
					State:   "open",
				},
			},
			want: "# Open Issues\n\n**Total Open Issues:** 3\n\n| Issue | Title | Author | Created |\n|-------|-------|--------|----------|\n| [#3](https://github.com/test/repo/issues/3) | Third issue | @charlie | 2024-01-01T00:00:00Z |\n| [#2](https://github.com/test/repo/issues/2) | Second issue | @bob | 2024-01-02T00:00:00Z |\n| [#1](https://github.com/test/repo/issues/1) | First issue | @alice | 2024-01-03T00:00:00Z |\n\n",
		},
		{
			name: "issues with long title and body",
			issues: []IssueListItem{
				{
					Number: 100,
					URL:    "https://github.com/test/repo/issues/100",
					User: struct {
						Login string `json:"login"`
					}{Login: "developer123"},
					Title:   "Implement comprehensive user authentication with OAuth2 and JWT tokens",
					Body:    "This is a very long issue description that implements comprehensive user authentication with OAuth2 and JWT tokens.\n\nIt includes:\n- OAuth2 provider integration\n- JWT token generation and validation\n- Refresh token mechanism\n- Session management\n- User profile synchronization\n\nAll tests are included and the issue is ready for review.",
					Created: "2024-03-15T14:30:00Z",
					State:   "open",
				},
			},
			want: "# Open Issues\n\n**Total Open Issues:** 1\n\n| Issue | Title | Author | Created |\n|-------|-------|--------|----------|\n| [#100](https://github.com/test/repo/issues/100) | Implement comprehensive user authentication with OAuth2 and JWT tokens | @developer123 | 2024-03-15T14:30:00Z |\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a minimal GitHubTools instance (formatOpenIssuesListMarkdown doesn't use it)
			ghTools := &GitHubTools{}

			got := ghTools.formatOpenIssuesListMarkdown(tt.issues)
			if got != tt.want {
				t.Errorf("formatOpenIssuesListMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
