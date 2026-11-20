package github

import "testing"

type milestoneMarkdownUser = struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type milestoneMarkdownLabel = struct {
	Name string `json:"name"`
}

type milestoneMarkdownData = struct {
	URL          string                `json:"url"`
	HTMLURL      string                `json:"html_url"`
	LabelsURL    string                `json:"labels_url"`
	ID           int                   `json:"id"`
	Number       int                   `json:"number"`
	State        string                `json:"state"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Creator      milestoneMarkdownUser `json:"creator"`
	OpenIssues   int                   `json:"open_issues"`
	ClosedIssues int                   `json:"closed_issues"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
	DueOn        string                `json:"due_on"`
}

type milestoneMarkdownIssue = struct {
	URL       string                   `json:"url"`
	Number    int                      `json:"number"`
	Title     string                   `json:"title"`
	Body      string                   `json:"body"`
	State     string                   `json:"state"`
	Labels    []milestoneMarkdownLabel `json:"labels"`
	User      milestoneMarkdownUser    `json:"user"`
	CreatedAt string                   `json:"created_at"`
	UpdatedAt string                   `json:"updated_at"`
	ClosedAt  string                   `json:"closed_at"`
	HTMLURL   string                   `json:"html_url"`
}

func TestFormatMilestoneMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		milestone *milestoneMarkdownData
		issues    []milestoneMarkdownIssue
		want      string
	}{
		{
			name: "minimal milestone without issues",
			milestone: &milestoneMarkdownData{
				Number:    1,
				State:     "open",
				Title:     "First release",
				Creator:   milestoneMarkdownUser{Login: "octocat"},
				CreatedAt: "2026-01-02T03:04:05Z",
				UpdatedAt: "2026-01-03T04:05:06Z",
				HTMLURL:   "https://github.com/example/repo/milestone/1",
			},
			want: "# Milestone #1: First release\n\n" +
				"**Status:** open\n\n" +
				"**Creator:** @octocat\n\n" +
				"**Created:** 2026-01-02T03:04:05Z\n\n" +
				"**Updated:** 2026-01-03T04:05:06Z\n\n" +
				"**URL:** https://github.com/example/repo/milestone/1\n\n" +
				"**Issues:** 0 open, 0 closed\n\n",
		},
		{
			name: "milestone with description and due date",
			milestone: &milestoneMarkdownData{
				Number:       2,
				State:        "open",
				Title:        "Version 2.0",
				Description:  "Ship the next major version.",
				Creator:      milestoneMarkdownUser{Login: "maintainer"},
				OpenIssues:   3,
				ClosedIssues: 7,
				CreatedAt:    "2026-02-01T00:00:00Z",
				UpdatedAt:    "2026-02-10T12:00:00Z",
				DueOn:        "2026-03-01T00:00:00Z",
				HTMLURL:      "https://github.com/example/repo/milestone/2",
			},
			want: "# Milestone #2: Version 2.0\n\n" +
				"**Status:** open\n\n" +
				"**Creator:** @maintainer\n\n" +
				"**Created:** 2026-02-01T00:00:00Z\n\n" +
				"**Updated:** 2026-02-10T12:00:00Z\n\n" +
				"**Due on:** 2026-03-01T00:00:00Z\n\n" +
				"**URL:** https://github.com/example/repo/milestone/2\n\n" +
				"## Description\n\nShip the next major version.\n\n" +
				"**Issues:** 3 open, 7 closed\n\n",
		},
		{
			name: "single issue with description and labels",
			milestone: &milestoneMarkdownData{
				URL:       "https://api.github.com/repos/example/repo/milestones/3",
				HTMLURL:   "https://github.com/example/repo/milestone/3",
				LabelsURL: "https://api.github.com/repos/example/repo/milestones/3/labels",
				ID:        300,
				Number:    3,
				State:     "open",
				Title:     "Documentation",
				Creator: milestoneMarkdownUser{
					Login:     "writer",
					AvatarURL: "https://avatars.githubusercontent.com/u/300",
					HTMLURL:   "https://github.com/writer",
				},
				OpenIssues: 1,
				CreatedAt:  "2026-04-01T08:00:00Z",
				UpdatedAt:  "2026-04-02T09:00:00Z",
			},
			issues: []milestoneMarkdownIssue{
				{
					URL:    "https://api.github.com/repos/example/repo/issues/31",
					Number: 31,
					Title:  "Write migration guide",
					Body:   "Document the upgrade steps.",
					State:  "open",
					Labels: []milestoneMarkdownLabel{{Name: "documentation"}, {Name: "priority: high"}},
					User: milestoneMarkdownUser{
						Login:     "contributor",
						AvatarURL: "https://avatars.githubusercontent.com/u/301",
						HTMLURL:   "https://github.com/contributor",
					},
					CreatedAt: "2026-04-01T10:00:00Z",
					UpdatedAt: "2026-04-02T10:00:00Z",
					HTMLURL:   "https://github.com/example/repo/issues/31",
				},
			},
			want: "# Milestone #3: Documentation\n\n" +
				"**Status:** open\n\n" +
				"**Creator:** @writer\n\n" +
				"**Created:** 2026-04-01T08:00:00Z\n\n" +
				"**Updated:** 2026-04-02T09:00:00Z\n\n" +
				"**URL:** https://github.com/example/repo/milestone/3\n\n" +
				"**Issues:** 1 open, 0 closed\n\n" +
				"## Issues (1)\n\n" +
				"### [31] Write migration guide\n\n" +
				"**URL:** https://github.com/example/repo/issues/31\n\n" +
				"**Status:** open\n\n" +
				"**Description:**\nDocument the upgrade steps.\n\n" +
				"**Labels:** documentation, priority: high\n\n" +
				"**Created:** 2026-04-01T10:00:00Z\n" +
				"**Updated:** 2026-04-02T10:00:00Z\n\n---\n\n",
		},
		{
			name: "multiple issues include closed issue and empty fields",
			milestone: &milestoneMarkdownData{
				Number:       4,
				State:        "closed",
				Title:        "Legacy cleanup",
				Creator:      milestoneMarkdownUser{Login: "archivist"},
				ClosedIssues: 2,
				CreatedAt:    "2011-01-01T00:00:00Z",
				UpdatedAt:    "2012-02-02T00:00:00Z",
				HTMLURL:      "https://github.com/example/repo/milestone/4",
			},
			issues: []milestoneMarkdownIssue{
				{
					Number:    41,
					Title:     "Remove old command",
					Body:      "Delete the deprecated command.",
					State:     "closed",
					CreatedAt: "2011-03-01T00:00:00Z",
					UpdatedAt: "2011-03-02T00:00:00Z",
					ClosedAt:  "2011-03-03T00:00:00Z",
					HTMLURL:   "https://github.com/example/repo/issues/41",
				},
				{
					Number:    42,
					Title:     "Remove old flag",
					State:     "closed",
					CreatedAt: "2011-04-01T00:00:00Z",
					UpdatedAt: "2011-04-02T00:00:00Z",
					ClosedAt:  "2011-04-03T00:00:00Z",
					HTMLURL:   "https://github.com/example/repo/issues/42",
				},
			},
			want: "# Milestone #4: Legacy cleanup\n\n" +
				"**Status:** closed\n\n" +
				"**Creator:** @archivist\n\n" +
				"**Created:** 2011-01-01T00:00:00Z\n\n" +
				"**Updated:** 2012-02-02T00:00:00Z\n\n" +
				"**URL:** https://github.com/example/repo/milestone/4\n\n" +
				"**Issues:** 0 open, 2 closed\n\n" +
				"## Issues (2)\n\n" +
				"### [41] Remove old command\n\n" +
				"**URL:** https://github.com/example/repo/issues/41\n\n" +
				"**Status:** closed\n\n" +
				"**Description:**\nDelete the deprecated command.\n\n" +
				"**Created:** 2011-03-01T00:00:00Z\n" +
				"**Updated:** 2011-03-02T00:00:00Z\n" +
				"**Closed:** 2011-03-03T00:00:00Z\n\n---\n\n" +
				"### [42] Remove old flag\n\n" +
				"**URL:** https://github.com/example/repo/issues/42\n\n" +
				"**Status:** closed\n\n" +
				"**Created:** 2011-04-01T00:00:00Z\n" +
				"**Updated:** 2011-04-02T00:00:00Z\n" +
				"**Closed:** 2011-04-03T00:00:00Z\n\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			githubTools := &GitHubTools{}
			got := githubTools.formatMilestoneMarkdown(tt.milestone, tt.issues)
			if got != tt.want {
				t.Errorf("formatMilestoneMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
