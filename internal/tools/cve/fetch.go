package cve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const cveDefaultTimeout = 30 * time.Second

// CVETools provides CVE lookup via OSV API.
type CVETools struct {
	cfg *config.Config
}

// New creates a new CVETools instance.
func New(cfg *config.Config) *CVETools {
	return &CVETools{cfg: cfg}
}

// parseCVEID extracts the vulnerability identifier from various OSV database formats.
// Accepts all database prefixes defined in the OSV schema:
// CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, ALBA-, ALEA-, ALSA-, ASB-, PUB-, AZL-,
// BELL-, BIT-, BREW-, CGA-, CLEANSTART-, CURL-, DEBIAN-, DSA-, DLA-, DTSA-,
// ECHO-, EEF-, ELA-, GSD-, HSEC-, JLSEC-, KUBE-, LBSEC-, LSN-, MGASA-, MAL-,
// MINI-, OESA-, OSEC-, PHSA-, PSF-, PYSEC-, RHSA-/RHBA-/RHEA-, RLSA-/RXSA-,
// RSEC-, ROOT-, RUSTSEC-, SUSE-SU-/SUSE-RU-/SUSE-FU-/SUSE-OU-/openSUSE-SU-,
// UBUNTU-, USN-, V8-, VCPKG-, CLSA-
func parseCVEID(id string) (string, error) {
	id = strings.TrimSpace(id)

	prefixes := []string{
		"CVE-", "GO-", "GHSA-", "OSV-", "GSD-", "ALPINE-", "ALBA-", "ALEA-", "ALSA-",
		"ASB-", "PUB-", "AZL-", "BELL-", "BIT-", "BREW-", "CGA-", "CLEANSTART-",
		"CURL-", "DEBIAN-", "DSA-", "DLA-", "DTSA-", "ECHO-", "EEF-", "ELA-",
		"HSEC-", "JLSEC-", "KUBE-", "LBSEC-", "LSN-", "MGASA-", "MAL-", "MINI-",
		"OESA-", "OSEC-", "PHSA-", "PSF-", "PYSEC-", "RHSA-", "RHBA-", "RHEA-",
		"RLSA-", "RXSA-", "RSEC-", "ROOT-", "RUSTSEC-", "SUSE-SU-", "SUSE-RU-",
		"SUSE-FU-", "SUSE-OU-", "openSUSE-SU-", "UBUNTU-", "USN-", "V8-", "VCPKG-",
		"CLSA-",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(id, prefix) {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid identifier format: %s (must start with a valid OSV database prefix)", id)
}

// FetchVulnerability fetches vulnerability information from OSV API and returns Markdown content.
func (c *CVETools) FetchVulnerability(id string) (string, error) {
	parsedID, err := parseCVEID(id)
	if err != nil {
		return "", fmt.Errorf("cve: failed to parse identifier: %w", err)
	}

	printer.ToolCall(printer.IconSearch, "cve", "id", parsedID)

	apiURL := fmt.Sprintf("https://api.osv.dev/v1/vulns/%s", parsedID)

	client := &http.Client{Timeout: cveDefaultTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("cve: failed to fetch vulnerability %s: %w", parsedID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cve: vulnerability %s returned status %d: %s", parsedID, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cve: failed to read response body: %w", err)
	}

	var vulnData Vulnerability
	if err := json.Unmarshal(body, &vulnData); err != nil {
		return "", fmt.Errorf("cve: failed to parse response: %w", err)
	}

	return c.formatVulnerabilityMarkdown(&vulnData), nil
}

type Vulnerability struct {
	ID            string            `json:"id"`
	Summary       string            `json:"summary,omitempty"`
	Details       string            `json:"details,omitempty"`
	Modified      string            `json:"modified,omitempty"`
	Published     string            `json:"published,omitempty"`
	Removed       string            `json:"removed,omitempty"`
	References    []Reference       `json:"references,omitempty"`
	Affected      []AffectedPackage `json:"affected,omitempty"`
	SchemaVersion string            `json:"schema_version,omitempty"`
}

type Reference struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url"`
}

type AffectedPackage struct {
	Package           Package `json:"package"`
	Ranges            []Range `json:"ranges,omitempty"`
	EcosystemSpecific any     `json:"ecosystem_specific,omitempty"`
	DatabaseSpecific  any     `json:"database_specific,omitempty"`
}

type Package struct {
	Name      string `json:"name,omitempty"`
	Ecosystem string `json:"ecosystem,omitempty"`
	PURL      string `json:"purl,omitempty"`
}

type Range struct {
	Type   string       `json:"type,omitempty"`
	Repo   string       `json:"repo,omitempty"`
	Events []RangeEvent `json:"events,omitempty"`
}

type RangeEvent struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
	Limit      string `json:"limit,omitempty"`
}

func (c *CVETools) formatVulnerabilityMarkdown(vuln *Vulnerability) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", vuln.ID)

	if vuln.Summary != "" {
		fmt.Fprintf(&sb, "## Summary\n\n%s\n\n", vuln.Summary)
	}

	fmt.Fprintf(&sb, "**Published:** %s\n\n", vuln.Published)
	fmt.Fprintf(&sb, "**Modified:** %s\n\n", vuln.Modified)
	if vuln.Removed != "" {
		fmt.Fprintf(&sb, "**Removed:** %s\n\n", vuln.Removed)
	}

	if vuln.Details != "" {
		fmt.Fprintf(&sb, "## Details\n\n%s\n\n", vuln.Details)
	}

	if len(vuln.References) > 0 {
		fmt.Fprintf(&sb, "## References\n\n")
		for _, ref := range vuln.References {
			fmt.Fprintf(&sb, "- [%s](%s)\n", ref.URL, ref.URL)
		}
		sb.WriteString("\n")
	}

	if len(vuln.Affected) > 0 {
		fmt.Fprintf(&sb, "## Affected Packages\n\n")
		for _, pkg := range vuln.Affected {
			c.formatAffectedPackage(&sb, pkg)
		}
	}

	return sb.String()
}

func (c *CVETools) formatAffectedPackage(sb *strings.Builder, pkg AffectedPackage) {
	if pkg.Package.Name != "" {
		fmt.Fprintf(sb, "### %s (%s)\n\n", pkg.Package.Name, pkg.Package.Ecosystem)
	} else {
		sb.WriteString("### Package\n\n")
	}

	if pkg.Package.PURL != "" {
		fmt.Fprintf(sb, "**PURL:** %s\n\n", pkg.Package.PURL)
	}

	if len(pkg.Ranges) > 0 {
		sb.WriteString("## Version Ranges\n\n")
		for _, r := range pkg.Ranges {
			c.formatRange(sb, r)
		}
		sb.WriteString("\n")
	}
}

func (c *CVETools) formatRange(sb *strings.Builder, r Range) {
	fmt.Fprintf(sb, "- **Type:** %s\n", r.Type)
	if r.Repo != "" {
		fmt.Fprintf(sb, "  - **Repo:** %s\n", r.Repo)
	}

	if len(r.Events) > 0 {
		sb.WriteString("  - **Events:**\n")
		for _, e := range r.Events {
			if e.Introduced != "" {
				fmt.Fprintf(sb, "    - Introduced: %s\n", e.Introduced)
			}
			if e.Fixed != "" {
				fmt.Fprintf(sb, "    - Fixed: %s\n", e.Fixed)
			}
			if e.Limit != "" {
				fmt.Fprintf(sb, "    - Limit: %s\n", e.Limit)
			}
		}
	}
	sb.WriteString("\n")
}

func (c *CVETools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "cve",
			Description: "Lookup vulnerability information from OSV database using any valid OSV database identifier (CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, and many more)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Vulnerability identifier with any valid OSV database prefix (e.g., CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, etc.)",
					},
				},
				"required": []string{"id"},
			},
		},
	}
}

func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "cve",
			Description: "Lookup vulnerability information from OSV database using any valid OSV database identifier (CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, and many more)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Vulnerability identifier with any valid OSV database prefix (e.g., CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, etc.)",
					},
				},
				"required": []string{"id"},
			},
		},
	}
}

func (c *CVETools) Dispatch(name string, args string) string {
	switch name {
	case "cve":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := c.FetchVulnerability(params.ID)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	default:
		return "error: unknown tool " + name
	}
}
