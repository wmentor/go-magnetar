// Package github unit tests for the GitHub tools.
//
// This file contains comprehensive unit tests for the formatPullRequestsListMarkdown method
// which formats GitHub pull requests list as Markdown. The tests cover:
//   - Empty list of PRs
//   - Single PR
//   - Multiple PRs
//   - PRs with various details
//
// Tests follow table-driven patterns with t.Parallel() for concurrent execution.

package github

import (
	"testing"
)

func TestFormatPullRequestsListMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prs  []PullRequestListItem
		want string
	}{
		{
			name: "empty list of PRs",
			prs:  []PullRequestListItem{},
			want: "# Pull Requests\n\nNo open pull requests found.\n\n",
		},
		{
			name: "single PR",
			prs: []PullRequestListItem{
				{
					Number:  1,
					URL:     "https://github.com/test/repo/pull/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "testuser"},
					Title:   "Add new feature",
					Body:    "This PR adds a new feature.",
					Created: "2024-01-01T00:00:00Z",
				},
			},
			want: "# Pull Requests\n\n**Total Open PRs:** 1\n\n| PR | Title | Author | Created |\n|----|-------|--------|----------|\n| [#1](https://github.com/test/repo/pull/1) | Add new feature | @testuser | 2024-01-01T00:00:00Z |\n\n",
		},
		{
			name: "multiple PRs",
			prs: []PullRequestListItem{
				{
					Number:  1,
					URL:     "https://github.com/test/repo/pull/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "alice"},
					Title:   "Add new feature",
					Body:    "This PR adds a new feature.",
					Created: "2024-01-10T10:00:00Z",
				},
				{
					Number:  2,
					URL:     "https://github.com/test/repo/pull/2",
					User: struct {
						Login string `json:"login"`
					}{Login: "bob"},
					Title:   "Fix bug in login",
					Body:    "This PR fixes a bug in login.",
					Created: "2024-01-08T08:00:00Z",
				},
				{
					Number:  3,
					URL:     "https://github.com/test/repo/pull/3",
					User: struct {
						Login string `json:"login"`
					}{Login: "charlie"},
					Title:   "Update documentation",
					Body:    "This PR updates documentation.",
					Created: "2024-01-05T09:00:00Z",
				},
			},
			want: "# Pull Requests\n\n**Total Open PRs:** 3\n\n| PR | Title | Author | Created |\n|----|-------|--------|----------|\n| [#1](https://github.com/test/repo/pull/1) | Add new feature | @alice | 2024-01-10T10:00:00Z |\n| [#2](https://github.com/test/repo/pull/2) | Fix bug in login | @bob | 2024-01-08T08:00:00Z |\n| [#3](https://github.com/test/repo/pull/3) | Update documentation | @charlie | 2024-01-05T09:00:00Z |\n\n",
		},
		{
			name: "PRs with long title and body",
			prs: []PullRequestListItem{
				{
					Number:  100,
					URL:     "https://github.com/test/repo/pull/100",
					User: struct {
						Login string `json:"login"`
					}{Login: "developer123"},
					Title:   "Implement comprehensive user authentication with OAuth2 and JWT tokens",
					Body:    "This is a very long PR description that implements comprehensive user authentication with OAuth2 and JWT tokens.\n\nIt includes:\n- OAuth2 provider integration\n- JWT token generation and validation\n- Refresh token mechanism\n- Session management\n- User profile synchronization\n\nAll tests are included and the PR is ready for review.",
					Created: "2024-03-15T14:30:00Z",
				},
			},
			want: "# Pull Requests\n\n**Total Open PRs:** 1\n\n| PR | Title | Author | Created |\n|----|-------|--------|----------|\n| [#100](https://github.com/test/repo/pull/100) | Implement comprehensive user authentication with OAuth2 and JWT tokens | @developer123 | 2024-03-15T14:30:00Z |\n\n",
		},
		{
			name: "PRs with special characters in title",
			prs: []PullRequestListItem{
				{
					Number:  42,
					URL:     "https://github.com/test/repo/pull/42",
					User: struct {
						Login string `json:"login"`
					}{Login: "dev_user"},
					Title:   "Fix: Handle `null` values in JSON parsing",
					Body:    "Resolves issue #41.",
					Created: "2024-02-20T11:15:00Z",
				},
				{
					Number:  43,
					URL:     "https://github.com/test/repo/pull/43",
					User: struct {
						Login string `json:"login"`
					}{Login: "qa_tester"},
					Title:   "Update README: Add installation instructions for macOS",
					Body:    "Closes issue #40.",
					Created: "2024-02-21T16:45:00Z",
				},
			},
			want: "# Pull Requests\n\n**Total Open PRs:** 2\n\n| PR | Title | Author | Created |\n|----|-------|--------|----------|\n| [#42](https://github.com/test/repo/pull/42) | Fix: Handle `null` values in JSON parsing | @dev_user | 2024-02-20T11:15:00Z |\n| [#43](https://github.com/test/repo/pull/43) | Update README: Add installation instructions for macOS | @qa_tester | 2024-02-21T16:45:00Z |\n\n",
		},
		{
			name: "PRs sorted by creation date descending",
			prs: []PullRequestListItem{
				{
					Number:  3,
					URL:     "https://github.com/test/repo/pull/3",
					User: struct {
						Login string `json:"login"`
					}{Login: "charlie"},
					Title:   "Third PR",
					Body:    "Created first but listed third (wrong order)",
					Created: "2024-01-01T00:00:00Z",
				},
				{
					Number:  2,
					URL:     "https://github.com/test/repo/pull/2",
					User: struct {
						Login string `json:"login"`
					}{Login: "bob"},
					Title:   "Second PR",
					Body:    "Created second but listed second (correct)",
					Created: "2024-01-02T00:00:00Z",
				},
				{
					Number:  1,
					URL:     "https://github.com/test/repo/pull/1",
					User: struct {
						Login string `json:"login"`
					}{Login: "alice"},
					Title:   "First PR",
					Body:    "Created third but listed first (correct, sorted desc)",
					Created: "2024-01-03T00:00:00Z",
				},
			},
			want: "# Pull Requests\n\n**Total Open PRs:** 3\n\n| PR | Title | Author | Created |\n|----|-------|--------|----------|\n| [#3](https://github.com/test/repo/pull/3) | Third PR | @charlie | 2024-01-01T00:00:00Z |\n| [#2](https://github.com/test/repo/pull/2) | Second PR | @bob | 2024-01-02T00:00:00Z |\n| [#1](https://github.com/test/repo/pull/1) | First PR | @alice | 2024-01-03T00:00:00Z |\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a minimal GitHubTools instance (formatPullRequestsListMarkdown doesn't use it)
			ghTools := &GitHubTools{}

			got := ghTools.formatPullRequestsListMarkdown(tt.prs)
			if got != tt.want {
				t.Errorf("formatPullRequestsListMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
