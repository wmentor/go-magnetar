package nist

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockClient implements ClientHTTP interface for testing.
type mockClient struct {
	resp *http.Response
	err  error
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func TestAdapter_Get_EmptyCveID(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(&mockClient{})

	ctx := context.Background()
	cve, err := adapter.Get(ctx, "")

	require.Error(t, err)
	require.Nil(t, cve)
	require.Contains(t, err.Error(), "cveID is required")
}

func TestAdapter_Get_WithMockClient(t *testing.T) {
	t.Parallel()

	// Create a mock response matching the NIST API structure
	responseData := Response{
		ResultsPerPage: 1,
		TotalResults:   1,
		Vulnerabilities: []Vulnerability{{CVE: CVEData{
			ID:           "CVE-2026-78659",
			Descriptions: []Description{{Lang: "en", Value: "Test description"}},
			Published:    "2026-10-15T00:00:00Z",
			Metrics: Metrics{
				CvssMetricV31: []CvssMetric{{CvssData: CvssData{
					Version:      "3.1",
					BaseScore:    7.5,
					BaseSeverity: "HIGH",
				}}},
			},
			References: []Reference{{URL: "https://example.com"}},
		}}},
	}
	jsonData, err := json.Marshal(responseData)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(jsonData))),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	mock := &mockClient{resp: resp}
	adapter := NewAdapter(mock)

	ctx := context.Background()
	cve, err := adapter.Get(ctx, "CVE-2026-78659")

	require.NoError(t, err)
	require.Equal(t, "CVE-2026-78659", cve.ID)
	require.Equal(t, "Test description", cve.Description)
	require.Equal(t, 7.5, cve.BaseScore)
	require.Equal(t, "HIGH", cve.BaseSeverity)
	require.Equal(t, "2026-10-15T00:00:00Z", cve.Published)
}

func TestAdapter_Get_WithMockClientError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("request failed")
	mock := &mockClient{err: expectedErr}
	adapter := NewAdapter(mock)

	ctx := context.Background()
	cve, err := adapter.Get(ctx, "CVE-2026-78659")

	require.Error(t, err)
	require.Nil(t, cve)
	require.Contains(t, err.Error(), "request failed")
}
