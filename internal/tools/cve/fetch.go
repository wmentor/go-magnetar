package cve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wmentor/go-magnetar/internal/adapter/nist"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const cveDefaultTimeout = 2 * time.Minute

type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

func NewTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

type Params struct {
	ID string `json:"id" jsonschema_description:"Vulnerability identifier with any valid OSV database prefix (e.g., CVE-, GO-, etc.)"`
}

func (t *Tool) Name() string {
	return "cve"
}

func (t *Tool) Description() string {
	return "Lookup vulnerability information from OSV database using any valid OSV database identifier (CVE-, " +
		"GO-, GHSA-, OSV-, GSD-, ALPINE-, and many more)"
}

func (t *Tool) FetchVulnerability(ctx context.Context, params *Params) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, cveDefaultTimeout)
	defer cancel()

	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "cve", "id", params.ID)

	parsedID, err := parseCVEID(params.ID)
	if err != nil {
		return nil, fmt.Errorf("cve: failed to parse identifier: %w", err)
	}

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	if strings.HasPrefix(params.ID, "CVE-") {
		// Use NIST adapter for CVE identifiers
		nistAdapter := nist.NewAdapter(client)
		cveData, err := nistAdapter.Get(ctx, parsedID)
		if err != nil {
			return nil, fmt.Errorf("cve: failed to fetch from NIST: %w", err)
		}

		// Map NIST CVE data to Vulnerability struct
		t.printer.ToolCall(printer.IconDone, "information about the vulnerability was received", "vuln", params.ID)
		vulnData := mapNISTToVulnerability(cveData)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: formatVulnerabilityMarkdown(&vulnData),
		}, nil
	}

	// Use OSV API for non-CVE identifiers
	apiURL := fmt.Sprintf("https://api.osv.dev/v1/vulns/%s", parsedID)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("cve: create http request error: %w", err)
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cve: failed to fetch vulnerability %s: %w", parsedID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cve: vulnerability %s returned status %d: %s", parsedID, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cve: failed to read response body: %w", err)
	}

	var vulnData Vulnerability
	if err := json.Unmarshal(body, &vulnData); err != nil {
		return nil, fmt.Errorf("cve: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconDone, "information about the vulnerability was received", "vuln", params.ID)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: formatVulnerabilityMarkdown(&vulnData),
	}, nil
}

// parseCVEID extracts the vulnerability identifier from various OSV database formats.
// Accepts all database prefixes defined in the OSV schema:
// CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, ALBA-, ALEA-, ALSA-, ASB-, PUB-, AZL-,
// BELL-, BIT-, BREW-, CGA-, CLEANSTART-, CURL-, DEBIAN-, DSA-, DLA-, DTSA-,
// ECHO-, EEF-, ELA-, GSD-, HSEC-, JLSEC-, KUBE-, LBSEC-, LSN-, MGASA-, MAL-,
// MINI-, OESA-, OSEC-, PHSA-, PSF-, PYSEC-, RHSA-/RHBA-/RHEA-, RLSA-/RXSA-,
// RSEC-, ROOT-, RUSTSEC-, SUSE-SU-/SUSE-RU-/SUSE-FU-/SUSE-OU-/openSUSE-SU-,
// UBUNTU-, USN-, V8-, VCPKG-, CLSA-.
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
	// NIST-specific fields
	BaseScore    float64 `json:"base_score,omitempty"`
	BaseSeverity string  `json:"base_severity,omitempty"`
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

// mapNISTToVulnerability maps NIST CVE data to the Vulnerability struct.
func mapNISTToVulnerability(cveData *nist.CVE) Vulnerability {
	vuln := Vulnerability{
		ID:           cveData.ID,
		Summary:      cveData.Description,
		Published:    cveData.Published,
		References:   make([]Reference, len(cveData.References)),
		BaseScore:    cveData.BaseScore,
		BaseSeverity: cveData.BaseSeverity,
	}

	// Map references
	for i, ref := range cveData.References {
		vuln.References[i] = Reference{
			URL: ref.URL,
		}
	}

	return vuln
}

func formatVulnerabilityMarkdown(vuln *Vulnerability) string {
	if vuln == nil {
		return ""
	}

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

	// Add CVSS score if available
	if vuln.BaseScore != 0 {
		fmt.Fprintf(&sb, "**CVSS Score:** %.1f %s\n\n", vuln.BaseScore, vuln.BaseSeverity)
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
			formatAffectedPackage(&sb, pkg)
		}
	}

	return sb.String()
}

func formatAffectedPackage(sb *strings.Builder, pkg AffectedPackage) {
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
			formatRange(sb, r)
		}
		sb.WriteString("\n")
	}
}

func formatRange(sb *strings.Builder, r Range) {
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
