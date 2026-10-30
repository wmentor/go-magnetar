package nist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/wmentor/go-magnetar/internal/http/requester"
)

const (
	nistAPIURL = "https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=%s"

	// Without an API Key (Public/Anonymous): 5 requests in a rolling 30-second window.
	sendTimeout = time.Second * 7
)

var (
	ErrCveIDRequired = errors.New("cveID is required")

	limiter = rate.NewLimiter(rate.Every(sendTimeout), 1)
)

// CVE represents a Common Vulnerabilities and Exposures entry from NIST.
type CVE struct {
	ID           string      `json:"id"`
	Description  string      `json:"description"`
	BaseScore    float64     `json:"baseScore"`    //nolint:tagliatelle // NIST API uses camelCase
	BaseSeverity string      `json:"baseSeverity"` //nolint:tagliatelle // NIST API uses camelCase
	Published    string      `json:"published"`
	References   []Reference `json:"references"`
}

// Response from NIST API.
type Response struct {
	ResultsPerPage  int             `json:"resultsPerPage"` //nolint:tagliatelle // NIST API uses camelCase
	TotalResults    int             `json:"totalResults"`   //nolint:tagliatelle // NIST API uses camelCase
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// Vulnerability contains CVE data.
type Vulnerability struct {
	CVE CVEData `json:"cve"`
}

// CVEData contains detailed CVE information.
type CVEData struct {
	ID           string        `json:"id"`
	Descriptions []Description `json:"descriptions"`
	Metrics      Metrics       `json:"metrics"`
	References   []Reference   `json:"references"`
	Published    string        `json:"published"`
}

// Description represents a CVE description.
type Description struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

// Metrics contains vulnerability metrics.
type Metrics struct {
	CvssMetricV31 []CvssMetric `json:"cvssMetricV31"` //nolint:tagliatelle // NIST API uses camelCase
	CvssMetricV40 []CvssMetric `json:"cvssMetricV40"` //nolint:tagliatelle // NIST API uses camelCase
}

// CvssMetric contains CVSS v3.1 metric data.
type CvssMetric struct {
	CvssData CvssData `json:"cvssData"` //nolint:tagliatelle // NIST API uses camelCase
}

// CvssData contains CVSS v3.1 data.
type CvssData struct {
	Version      string  `json:"version"`
	VectorString string  `json:"vectorString"` //nolint:tagliatelle // NIST API uses camelCase
	BaseScore    float64 `json:"baseScore"`    //nolint:tagliatelle // NIST API uses camelCase
	BaseSeverity string  `json:"baseSeverity"` //nolint:tagliatelle // NIST API uses camelCase
}

// Reference represents a CVE reference.
type Reference struct {
	URL string `json:"url"`
}

// Adapter for NIST NVD API.
type Adapter struct {
	client requester.Client
}

// NewAdapterWithClient creates a new NIST adapter with custom HTTP client.
func NewAdapter(client requester.Client) *Adapter {
	return &Adapter{client: client}
}

// Get retrieves CVE information from NIST API.
func (a *Adapter) Get(ctx context.Context, cveID string) (*CVE, error) {
	if cveID == "" {
		return nil, errors.New("cveID is required")
	}

	if err := limiter.Wait(ctx); err != nil {
		return nil, errors.Wrap(ctx.Err(), "nist.Adapter.Get")
	}

	url := fmt.Sprintf(nistAPIURL, url.QueryEscape(cveID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "create request for cveID=%s", cveID)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "send request for cveID=%s", cveID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("nist API returned status code=%d for cveID=%s", resp.StatusCode, cveID)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read response body for cveID=%s", cveID)
	}

	var nistResp Response
	if err := json.Unmarshal(data, &nistResp); err != nil {
		return nil, errors.Wrapf(err, "unmarshal response for cveID=%s", cveID)
	}

	if len(nistResp.Vulnerabilities) == 0 {
		return nil, errors.Errorf("no CVE found for cveID=%s", cveID)
	}

	cve := a.extractCVE(nistResp.Vulnerabilities[0].CVE)
	return cve, nil
}

// extractCVE extracts CVE information from CVEData.
func (a *Adapter) extractCVE(data CVEData) *CVE {
	cve := &CVE{
		ID:          data.ID,
		Description: a.extractDescription(data.Descriptions),
		Published:   data.Published,
		References:  data.References,
	}

	// Extract base score and severity from CVSS metrics.
	// Prefer v3.1 over v4.0 for backward compatibility.
	if len(data.Metrics.CvssMetricV31) > 0 {
		cve.BaseScore = data.Metrics.CvssMetricV31[0].CvssData.BaseScore
		cve.BaseSeverity = data.Metrics.CvssMetricV31[0].CvssData.BaseSeverity
	} else if len(data.Metrics.CvssMetricV40) > 0 {
		cve.BaseScore = data.Metrics.CvssMetricV40[0].CvssData.BaseScore
		cve.BaseSeverity = data.Metrics.CvssMetricV40[0].CvssData.BaseSeverity
	}

	return cve
}

// extractDescription extracts the English description from the description list.
func (a *Adapter) extractDescription(descriptions []Description) string {
	for _, desc := range descriptions {
		if desc.Lang == "en" {
			return desc.Value
		}
	}
	if len(descriptions) > 0 {
		return descriptions[0].Value
	}
	return ""
}
