// Package github unit tests for the GitHub tools.
//
// This file contains comprehensive unit tests for the formatIssueMarkdown method
// which formats GitHub issue data as Markdown. The tests cover:
//   - Basic issue with minimal fields
//   - Issue with description
//   - Issue with labels
//   - Issue with assignees
//   - Issue with comments
//   - Locked issue
//   - Closed issue
//   - Issue with empty fields
//   - Issue with all fields populated
//
// Tests follow table-driven patterns with t.Parallel() for concurrent execution.

package github

import (
	"testing"
)

func TestFormatIssueMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		issue *struct {
			ID     int    `json:"id"`
			Number int    `json:"number"`
			Title  string `json:"title"`
			State  string `json:"state"`
			Locked bool   `json:"locked"`
			Body   string `json:"body"`
			User   struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
				HTMLURL   string `json:"html_url"`
			} `json:"user"`
			Labels []struct {
				Name        string `json:"name"`
				Color       string `json:"color"`
				Description string `json:"description"`
			} `json:"labels"`
			Assignees []struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
				HTMLURL   string `json:"html_url"`
			} `json:"assignees"`
			CommentCount int    `json:"comments"`
			CreatedAt    string `json:"created_at"`
			UpdatedAt    string `json:"updated_at"`
			ClosedAt     string `json:"closed_at"`
			HTMLURL      string `json:"html_url"`
		}
		comments []string
		want     string
	}{
		{
			name: "basic issue with minimal fields",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     1,
				Number: 123,
				Title:  "Test Issue",
				State:  "open",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "testuser",
					AvatarURL: "https://avatars.githubusercontent.com/u/1",
					HTMLURL:   "https://github.com/testuser",
				},
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-01T00:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/123",
			},
			comments: []string{},
			want:     "# Issue #123: Test Issue\n\n**Status:** open\n\n**Author:** @testuser\n\n**Created:** 2024-01-01T00:00:00Z\n\n**Updated:** 2024-01-01T00:00:00Z\n\n**URL:** https://github.com/test/repo/issues/123\n\n",
		},
		{
			name: "issue with description",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     2,
				Number: 456,
				Title:  "Bug Report",
				State:  "open",
				Body:   "This is a bug description.\n\nIt has multiple lines.",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "bughunter",
					AvatarURL: "https://avatars.githubusercontent.com/u/2",
					HTMLURL:   "https://github.com/bughunter",
				},
				CreatedAt: "2024-01-15T10:30:00Z",
				UpdatedAt: "2024-01-15T10:30:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/456",
			},
			comments: []string{},
			want:     "# Issue #456: Bug Report\n\n**Status:** open\n\n**Author:** @bughunter\n\n**Created:** 2024-01-15T10:30:00Z\n\n**Updated:** 2024-01-15T10:30:00Z\n\n**URL:** https://github.com/test/repo/issues/456\n\n## Description\n\nThis is a bug description.\n\nIt has multiple lines.\n\n",
		},
		{
			name: "issue with labels",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     3,
				Number: 789,
				Title:  "Feature Request",
				State:  "open",
				Labels: []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				}{
					{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
					{Name: "documentation", Color: "0075ca", Description: "Improvements or additions to documentation"},
				},
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "featurecreator",
					AvatarURL: "https://avatars.githubusercontent.com/u/3",
					HTMLURL:   "https://github.com/featurecreator",
				},
				CreatedAt: "2024-02-01T08:00:00Z",
				UpdatedAt: "2024-02-01T08:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/789",
			},
			comments: []string{},
			want:     "# Issue #789: Feature Request\n\n**Status:** open\n\n**Author:** @featurecreator\n\n**Created:** 2024-02-01T08:00:00Z\n\n**Updated:** 2024-02-01T08:00:00Z\n\n**URL:** https://github.com/test/repo/issues/789\n\n**Labels:** enhancement, documentation\n\n",
		},
		{
			name: "issue with assignees",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     4,
				Number: 101,
				Title:  "Assignee Test",
				State:  "open",
				Assignees: []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					{Login: "alice", AvatarURL: "https://avatars.githubusercontent.com/u/10", HTMLURL: "https://github.com/alice"},
					{Login: "bob", AvatarURL: "https://avatars.githubusercontent.com/u/20", HTMLURL: "https://github.com/bob"},
				},
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "requester",
					AvatarURL: "https://avatars.githubusercontent.com/u/4",
					HTMLURL:   "https://github.com/requester",
				},
				CreatedAt: "2024-02-15T12:00:00Z",
				UpdatedAt: "2024-02-15T12:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/101",
			},
			comments: []string{},
			want:     "# Issue #101: Assignee Test\n\n**Status:** open\n\n**Author:** @requester\n\n**Created:** 2024-02-15T12:00:00Z\n\n**Updated:** 2024-02-15T12:00:00Z\n\n**URL:** https://github.com/test/repo/issues/101\n\n**Assignees:** alice, bob\n\n",
		},
		{
			name: "issue with comments",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     5,
				Number: 202,
				Title:  "Issue with Comments",
				State:  "open",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "opener",
					AvatarURL: "https://avatars.githubusercontent.com/u/5",
					HTMLURL:   "https://github.com/opener",
				},
				CreatedAt: "2024-03-01T09:00:00Z",
				UpdatedAt: "2024-03-05T14:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/202",
			},
			comments: []string{
				"### Comment by @reviewer1\n\nFirst comment.\n\n**Created:** 2024-03-02T10:00:00Z\n**Updated:** 2024-03-02T10:00:00Z\n",
				"### Comment by @reviewer2\n\nSecond comment.\n\n**Created:** 2024-03-03T11:00:00Z\n**Updated:** 2024-03-03T11:00:00Z\n",
			},
			want: "# Issue #202: Issue with Comments\n\n**Status:** open\n\n**Author:** @opener\n\n**Created:** 2024-03-01T09:00:00Z\n\n**Updated:** 2024-03-05T14:00:00Z\n\n**URL:** https://github.com/test/repo/issues/202\n\n## Comments (2)\n\n### Comment by @reviewer1\n\nFirst comment.\n\n**Created:** 2024-03-02T10:00:00Z\n**Updated:** 2024-03-02T10:00:00Z\n\n---\n\n### Comment by @reviewer2\n\nSecond comment.\n\n**Created:** 2024-03-03T11:00:00Z\n**Updated:** 2024-03-03T11:00:00Z\n\n---\n\n",
		},
		{
			name: "locked issue",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     6,
				Number: 303,
				Title:  "Locked Issue",
				State:  "open",
				Locked: true,
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "opener",
					AvatarURL: "https://avatars.githubusercontent.com/u/6",
					HTMLURL:   "https://github.com/opener",
				},
				CreatedAt: "2024-03-10T08:00:00Z",
				UpdatedAt: "2024-03-10T08:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/303",
			},
			comments: []string{},
			want:     "# Issue #303: Locked Issue\n\n**Status:** open\n\n**⚠️ This issue is locked**\n\n**Author:** @opener\n\n**Created:** 2024-03-10T08:00:00Z\n\n**Updated:** 2024-03-10T08:00:00Z\n\n**URL:** https://github.com/test/repo/issues/303\n\n",
		},
		{
			name: "closed issue",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     7,
				Number: 404,
				Title:  "Closed Issue",
				State:  "closed",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "opener",
					AvatarURL: "https://avatars.githubusercontent.com/u/7",
					HTMLURL:   "https://github.com/opener",
				},
				CreatedAt: "2024-03-01T10:00:00Z",
				UpdatedAt: "2024-03-15T12:00:00Z",
				ClosedAt:  "2024-03-15T12:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/404",
			},
			comments: []string{},
			want:     "# Issue #404: Closed Issue\n\n**Status:** closed\n\n**Author:** @opener\n\n**Created:** 2024-03-01T10:00:00Z\n\n**Updated:** 2024-03-15T12:00:00Z\n\n**Closed:** 2024-03-15T12:00:00Z\n\n**URL:** https://github.com/test/repo/issues/404\n\n",
		},
		{
			name: "issue with all fields populated",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     8,
				Number: 505,
				Title:  "Complete Issue Test",
				State:  "closed",
				Locked: true,
				Body:   "This is a complete test issue.\n\nWith multiple lines.",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "testuser",
					AvatarURL: "https://avatars.githubusercontent.com/u/8",
					HTMLURL:   "https://github.com/testuser",
				},
				Labels: []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				}{
					{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
					{Name: "priority-high", Color: "ff0000", Description: "High priority"},
				},
				Assignees: []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					{Login: "developer", AvatarURL: "https://avatars.githubusercontent.com/u/9", HTMLURL: "https://github.com/developer"},
				},
				CommentCount: 3,
				CreatedAt:    "2024-04-01T09:00:00Z",
				UpdatedAt:    "2024-04-05T15:00:00Z",
				ClosedAt:     "2024-04-05T15:00:00Z",
				HTMLURL:      "https://github.com/test/repo/issues/505",
			},
			comments: []string{},
			want:     "# Issue #505: Complete Issue Test\n\n**Status:** closed\n\n**⚠️ This issue is locked**\n\n**Author:** @testuser\n\n**Created:** 2024-04-01T09:00:00Z\n\n**Updated:** 2024-04-05T15:00:00Z\n\n**Closed:** 2024-04-05T15:00:00Z\n\n**URL:** https://github.com/test/repo/issues/505\n\n**Labels:** bug, priority-high\n\n**Assignees:** developer\n\n## Description\n\nThis is a complete test issue.\n\nWith multiple lines.\n\n",
		},
		{
			name: "issue with empty body",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     9,
				Number: 606,
				Title:  "Empty Body Issue",
				State:  "open",
				Body:   "",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "opener",
					AvatarURL: "https://avatars.githubusercontent.com/u/9",
					HTMLURL:   "https://github.com/opener",
				},
				CreatedAt: "2024-04-10T10:00:00Z",
				UpdatedAt: "2024-04-10T10:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/606",
			},
			comments: []string{},
			want:     "# Issue #606: Empty Body Issue\n\n**Status:** open\n\n**Author:** @opener\n\n**Created:** 2024-04-10T10:00:00Z\n\n**Updated:** 2024-04-10T10:00:00Z\n\n**URL:** https://github.com/test/repo/issues/606\n\n",
		},
		{
			name: "issue with no labels and no assignees",
			issue: &struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
				Locked bool   `json:"locked"`
				Body   string `json:"body"`
				User   struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"user"`
				Labels []struct {
					Name        string `json:"name"`
					Color       string `json:"color"`
					Description string `json:"description"`
				} `json:"labels"`
				Assignees []struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				} `json:"assignees"`
				CommentCount int    `json:"comments"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				ClosedAt     string `json:"closed_at"`
				HTMLURL      string `json:"html_url"`
			}{
				ID:     10,
				Number: 707,
				Title:  "No Labels or Assignees",
				State:  "open",
				User: struct {
					Login     string `json:"login"`
					AvatarURL string `json:"avatar_url"`
					HTMLURL   string `json:"html_url"`
				}{
					Login:     "opener",
					AvatarURL: "https://avatars.githubusercontent.com/u/10",
					HTMLURL:   "https://github.com/opener",
				},
				CreatedAt: "2024-04-15T11:00:00Z",
				UpdatedAt: "2024-04-15T11:00:00Z",
				HTMLURL:   "https://github.com/test/repo/issues/707",
			},
			comments: []string{},
			want:     "# Issue #707: No Labels or Assignees\n\n**Status:** open\n\n**Author:** @opener\n\n**Created:** 2024-04-15T11:00:00Z\n\n**Updated:** 2024-04-15T11:00:00Z\n\n**URL:** https://github.com/test/repo/issues/707\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a minimal GitHubTools instance (formatIssueMarkdown doesn't use it)
			ghTools := &GitHubTools{}

			got := ghTools.formatIssueMarkdown(tt.issue, tt.comments)
			if got != tt.want {
				t.Errorf("formatIssueMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
