// Package github unit tests for the GitHub tools.
//
// This file contains unit tests for the formatAdvisoryMarkdown method, which
// formats a single GitHub Security Advisory as Markdown. The tests cover:
//   - Basic advisory with minimal fields
//   - Advisory with CVE ID
//   - Advisory with description
//   - Advisory with references
//   - Advisory with a vulnerability and its version range
//   - Advisory with patched versions
//   - Advisory with withdrawn date
//   - Advisory with empty fields
//   - Multiple vulnerabilities in a single advisory
//   - Advisory with CVSS and CWE data (not rendered today)
//   - Advisory with all fields populated
//
// Tests follow table-driven patterns with t.Parallel() for concurrent execution.

package github

import (
	"slices"
	"testing"
)

// advisoryVuln describes one entry of SecurityAdvisory.Vulnerabilities.
type advisoryVuln struct {
	name, ecosystem, vulnerableRange, firstPatched string
}

// newTestAdvisory returns a copy of base with the given vulnerabilities appended.
func newTestAdvisory(t *testing.T, base SecurityAdvisory, vulns ...advisoryVuln) *SecurityAdvisory {
	t.Helper()

	advisory := base
	for _, v := range vulns {
		n := len(advisory.Vulnerabilities)
		advisory.Vulnerabilities = slices.Grow(advisory.Vulnerabilities, 1)[:n+1]
		entry := &advisory.Vulnerabilities[n]
		entry.Package.Name = v.name
		entry.Package.Ecosystem = v.ecosystem
		entry.VulnerableVersionRange = v.vulnerableRange
		entry.FirstPatchedVersion = v.firstPatched
	}

	return &advisory
}

func TestFormatAdvisoryMarkdown(t *testing.T) {
	t.Parallel()

	withdrawn := "2024-03-01T00:00:00Z"
	vector := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"

	minimal := SecurityAdvisory{
		GHSAID:      "GHSA-aaaa-bbbb-cccc",
		Summary:     "Prototype pollution in example-lib",
		Severity:    "high",
		PublishedAt: "2024-01-15T10:00:00Z",
		UpdatedAt:   "2024-01-20T12:00:00Z",
	}
	minimalHeader := "# Prototype pollution in example-lib\n\n" +
		"**GHSA ID:** GHSA-aaaa-bbbb-cccc\n\n" +
		"**Severity:** high\n\n" +
		"**Published:** 2024-01-15T10:00:00Z\n\n" +
		"**Updated:** 2024-01-20T12:00:00Z\n\n"

	withCVE := minimal
	withCVE.CveID = "CVE-2024-12345"

	withDescription := minimal
	withDescription.Description = "Crafted input can modify Object.prototype."

	withReferences := minimal
	withReferences.References = []string{
		"https://nvd.nist.gov/vuln/detail/CVE-2024-12345",
		"https://github.com/example/example-lib/commit/abc123",
	}

	withWithdrawn := minimal
	withWithdrawn.WithdrawnAt = &withdrawn

	withCVSS := minimal
	withCVSS.CvssSeverities.CvssV3.VectorString = &vector
	withCVSS.CvssSeverities.CvssV3.Score = 9.8
	withCVSS.Cwes = slices.Grow(withCVSS.Cwes, 1)[:1]
	withCVSS.Cwes[0].CweID = "CWE-1321"
	withCVSS.Cwes[0].Name = "Prototype Pollution"

	full := minimal
	full.CveID = "CVE-2024-12345"
	full.Description = "Crafted input can modify Object.prototype."
	full.References = []string{"https://nvd.nist.gov/vuln/detail/CVE-2024-12345"}
	full.WithdrawnAt = &withdrawn

	tests := []struct {
		name     string
		advisory *SecurityAdvisory
		want     string
	}{
		{
			name:     "basic advisory with minimal fields",
			advisory: newTestAdvisory(t, minimal),
			want:     minimalHeader,
		},
		{
			name:     "advisory with CVE ID",
			advisory: newTestAdvisory(t, withCVE),
			want: "# Prototype pollution in example-lib\n\n" +
				"**GHSA ID:** GHSA-aaaa-bbbb-cccc\n\n" +
				"**CVE ID:** CVE-2024-12345\n\n" +
				"**Severity:** high\n\n" +
				"**Published:** 2024-01-15T10:00:00Z\n\n" +
				"**Updated:** 2024-01-20T12:00:00Z\n\n",
		},
		{
			name:     "advisory with description",
			advisory: newTestAdvisory(t, withDescription),
			want: minimalHeader +
				"## Description\n\nCrafted input can modify Object.prototype.\n\n",
		},
		{
			name:     "advisory with references",
			advisory: newTestAdvisory(t, withReferences),
			want: minimalHeader +
				"## References\n\n" +
				"- https://nvd.nist.gov/vuln/detail/CVE-2024-12345\n" +
				"- https://github.com/example/example-lib/commit/abc123\n" +
				"\n",
		},
		{
			name: "advisory with a vulnerability and version range",
			advisory: newTestAdvisory(t, minimal, advisoryVuln{
				name: "example-lib", ecosystem: "npm", vulnerableRange: "< 1.2.3",
			}),
			want: minimalHeader +
				"## Vulnerable Package\n\n" +
				"**Package:** example-lib (npm)\n\n" +
				"**Vulnerable Versions:** < 1.2.3\n\n",
		},
		{
			name: "advisory with patched version",
			advisory: newTestAdvisory(t, minimal, advisoryVuln{
				name: "example-lib", ecosystem: "npm", vulnerableRange: "< 1.2.3", firstPatched: "1.2.3",
			}),
			want: minimalHeader +
				"## Vulnerable Package\n\n" +
				"**Package:** example-lib (npm)\n\n" +
				"**Patched Versions:** 1.2.3\n\n" +
				"**Vulnerable Versions:** < 1.2.3\n\n",
		},
		{
			name:     "advisory with withdrawn date",
			advisory: newTestAdvisory(t, withWithdrawn),
			want: minimalHeader +
				"**Withdrawn:** 2024-03-01T00:00:00Z\n\n",
		},
		{
			name: "advisory with empty fields",
			advisory: newTestAdvisory(t, SecurityAdvisory{
				References: []string{},
			}),
			want: "# \n\n" +
				"**GHSA ID:** \n\n" +
				"**Severity:** \n\n" +
				"**Published:** \n\n" +
				"**Updated:** \n\n",
		},
		{
			name: "vulnerability without range or patched version",
			advisory: newTestAdvisory(t, minimal, advisoryVuln{
				name: "example-lib", ecosystem: "npm",
			}),
			want: minimalHeader +
				"## Vulnerable Package\n\n" +
				"**Package:** example-lib (npm)\n\n",
		},
		{
			name: "multiple vulnerabilities show the first package and join versions",
			advisory: newTestAdvisory(t, minimal,
				advisoryVuln{name: "example-lib", ecosystem: "npm", vulnerableRange: ">= 1.0.0, < 1.2.3", firstPatched: "1.2.3"},
				advisoryVuln{name: "example-lib", ecosystem: "npm", vulnerableRange: ">= 2.0.0, < 2.0.5", firstPatched: "2.0.5"},
				advisoryVuln{name: "example-lib-extra", ecosystem: "npm", vulnerableRange: "< 0.4.0"},
			),
			want: minimalHeader +
				"## Vulnerable Package\n\n" +
				"**Package:** example-lib (npm)\n\n" +
				"**Patched Versions:** 1.2.3, 2.0.5\n\n" +
				"**Vulnerable Versions:** >= 1.0.0, < 1.2.3, >= 2.0.0, < 2.0.5, < 0.4.0\n\n",
		},
		{
			name:     "CVSS and CWE data are not rendered",
			advisory: newTestAdvisory(t, withCVSS),
			want:     minimalHeader,
		},
		{
			name: "advisory with all fields populated",
			advisory: newTestAdvisory(t, full, advisoryVuln{
				name: "example-lib", ecosystem: "npm", vulnerableRange: "< 1.2.3", firstPatched: "1.2.3",
			}),
			want: "# Prototype pollution in example-lib\n\n" +
				"**GHSA ID:** GHSA-aaaa-bbbb-cccc\n\n" +
				"**CVE ID:** CVE-2024-12345\n\n" +
				"**Severity:** high\n\n" +
				"**Published:** 2024-01-15T10:00:00Z\n\n" +
				"**Updated:** 2024-01-20T12:00:00Z\n\n" +
				"**Withdrawn:** 2024-03-01T00:00:00Z\n\n" +
				"## Description\n\nCrafted input can modify Object.prototype.\n\n" +
				"## References\n\n" +
				"- https://nvd.nist.gov/vuln/detail/CVE-2024-12345\n" +
				"\n" +
				"## Vulnerable Package\n\n" +
				"**Package:** example-lib (npm)\n\n" +
				"**Patched Versions:** 1.2.3\n\n" +
				"**Vulnerable Versions:** < 1.2.3\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := createTestConfig(t)
			ghTools := New(cfg, nil, nil, nil)

			got := ghTools.formatAdvisoryMarkdown(tt.advisory)
			if got != tt.want {
				t.Errorf("formatAdvisoryMarkdown() mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
