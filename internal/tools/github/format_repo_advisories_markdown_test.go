package github

import (
	"testing"
)

// Test formatRepoAdvisoriesMarkdown with empty advisories list
func TestFormatRepoAdvisoriesMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		advisories []SecurityAdvisory
		expect     []string
		notExpect  []string
	}{
		{
			name:       "empty list",
			advisories: []SecurityAdvisory{},
			expect: []string{
				"# Security Advisories",
				"No security advisories found",
			},
			notExpect: []string{
				"Total Advisories",
				"## Advisories",
				"| ID",
			},
		},
		{
			name: "single advisory",
			advisories: []SecurityAdvisory{
				{
					GHSAID:      "GHSA-xxxx-xxxx-xxxx",
					Summary:     "Test vulnerability",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-xxxx-xxxx-xxxx",
					Severity:    "high",
					PublishedAt: "2024-01-15T10:00:00Z",
				},
			},
			expect: []string{
				"# Security Advisories",
				"**Total Advisories:** 1",
				"## Advisories",
				"| ID | Title | Severity | Published |",
				"|----|-------|----------|----------|",
				"| [GHSA-xxxx-xxxx-xxxx]",
				"Test vulnerability",
				"high",
				"2024-01-15T10:00:00Z",
			},
			notExpect: []string{},
		},
		{
			name: "multiple advisories",
			advisories: []SecurityAdvisory{
				{
					GHSAID:      "GHSA-1111-1111-1111",
					Summary:     "Vulnerability one",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-1111-1111-1111",
					Severity:    "critical",
					PublishedAt: "2024-01-10T10:00:00Z",
				},
				{
					GHSAID:      "GHSA-2222-2222-2222",
					Summary:     "Vulnerability two",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-2222-2222-2222",
					Severity:    "medium",
					PublishedAt: "2024-01-12T10:00:00Z",
				},
				{
					GHSAID:      "GHSA-3333-3333-3333",
					Summary:     "Vulnerability three",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-3333-3333-3333",
					Severity:    "low",
					PublishedAt: "2024-01-14T10:00:00Z",
				},
			},
			expect: []string{
				"# Security Advisories",
				"**Total Advisories:** 3",
				"| [GHSA-1111-1111-1111]",
				"| [GHSA-2222-2222-2222]",
				"| [GHSA-3333-3333-3333]",
				"Vulnerability one",
				"Vulnerability two",
				"Vulnerability three",
				"critical",
				"medium",
				"low",
			},
			notExpect: []string{},
		},
		{
			name: "missing severity defaults to Unknown",
			advisories: []SecurityAdvisory{
				{
					GHSAID:      "GHSA-xxxx-xxxx-xxxx",
					Summary:     "Test vulnerability",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-xxxx-xxxx-xxxx",
					Severity:    "",
					PublishedAt: "2024-01-15T10:00:00Z",
				},
			},
			expect: []string{
				"# Security Advisories",
				"| Unknown |",
			},
			notExpect: []string{},
		},
		{
			name: "full advisory with all fields",
			advisories: []SecurityAdvisory{
				{
					GHSAID:      "GHSA-full-1234-5678",
					Summary:     "Complete vulnerability details",
					HTMLURL:     "https://github.com/test/repo/advisories/GHSA-full-1234-5678",
					Severity:    "critical",
					PublishedAt: "2024-01-15T10:00:00Z",
				},
			},
			expect: []string{
				"# Security Advisories",
				"**Total Advisories:** 1",
				"| ID | Title | Severity | Published |",
				"|----|-------|----------|----------|",
				"| [GHSA-full-1234-5678]",
				"Complete vulnerability details",
				"critical",
				"2024-01-15T10:00:00Z",
			},
			notExpect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Create a minimal valid config
			cfg := createTestConfig(t)
			ghTools := New(cfg, nil, nil, nil)

			result := ghTools.formatRepoAdvisoriesMarkdown(tt.advisories)

			// Check expected strings are present
			for _, expect := range tt.expect {
				if !containsString(result, expect) {
					t.Errorf("result missing expected string %q:\n%s", expect, result)
				}
			}

			// Check unexpected strings are absent
			for _, notExpect := range tt.notExpect {
				if containsString(result, notExpect) {
					t.Errorf("result unexpectedly contains %q:\n%s", notExpect, result)
				}
			}
		})
	}
}
