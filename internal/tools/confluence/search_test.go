package confluence

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// mockClient implements the requester.Client interface for testing
type mockClient struct {
	resp *http.Response
	err  error
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

// mockClientFactory implements the requester.ClientFactory interface for testing
type mockClientFactory struct {
	client *mockClient
}

func (f *mockClientFactory) Get() requester.Client {
	return f.client
}

func (f *mockClientFactory) Put(_ requester.Client) {
}

func TestSearchTool_Search(t *testing.T) {
	t.Parallel()

	// Create a mock response
	body := []byte(`{"results":[{"content":{"id":"123","type":"page","status":"current","title":"Test Page","_links":{"webui":"/spaces/TEST/pages/123"}},"title":"Test Page","excerpt":"Test excerpt","url":"/spaces/TEST/pages/123","resultGlobalContainer":{"title":"Test Space"},"entityType":"content","iconCssClass":"aui-icon content-type-page","lastModified":"2024-01-01T00:00:00Z","friendlyLastModified":"01 Jan 2024","timestamp":1704067200000}],"start":0,"limit":25,"size":1}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"application/json"}},
		Body:       &mockReadCloser{data: body, pos: 0},
	}

	// Create mock client
	mockClient := &mockClient{resp: resp, err: nil}

	// Create mock client factory
	clientFactory := &mockClientFactory{client: mockClient}

	// Create config using temp file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	configContent := `version: "1.0"
profile: default
confluence:
  enable: true
  base_url: "https://confluence.example.com"
  api_key: test-token
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.NoError(t, err)

	// Debug: print file content
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	t.Logf("Config file content:\n%s", string(content))

	// Load config
	cfg, err := config.Load(configPath)
	assert.NoError(t, err)

	// Debug: print profile and base_url
	t.Logf("Profile: %s", cfg.Profile())
	t.Logf("confluence.base_url: %s", cfg.String("confluence.base_url"))
	t.Logf("confluence.api_key: %s", cfg.String("confluence.api_key"))

	// Create printer
	p := printer.GetDefault()

	// Create search tool
	tool := NewSearchTool(cfg, p, clientFactory)

	// Execute search
	params := &SearchParams{
		Query: "test query",
		Limit: 25,
	}

	ctx := context.Background()
	result, err := tool.Search(ctx, params)

	assert.NoError(t, err)

	// Verify result is a Response
	respType, ok := result.(*tools.Response)
	assert.True(t, ok, "result should be *tools.Response")

	// Verify response status
	assert.Equal(t, tools.StatusOK, respType.Status)

	// Verify response contains expected content
	assert.Contains(t, respType.Result, "Confluence Search Results")
	assert.Contains(t, respType.Result, "Total Results")
	assert.Contains(t, respType.Result, "Test Page")
	assert.Contains(t, respType.Result, "https://confluence.example.com/spaces/TEST/pages/123")
}

func TestSearchTool_NameAndDescription(t *testing.T) {
	t.Parallel()

	// Create config using temp file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	configContent := `version: "1.0"
profile: default
confluence:
  enable: true
  base_url: "https://confluence.example.com"
  api_key: test-token
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.NoError(t, err)

	// Load config
	cfg, err := config.Load(configPath)
	assert.NoError(t, err)

	// Create printer
	p := printer.GetDefault()

	// Create client factory
	clientFactory := requester.NewFactory(cfg)

	// Create search tool
	tool := NewSearchTool(cfg, p, clientFactory)

	// Verify name and description
	assert.Equal(t, "confluence_search", tool.Name())
	desc := tool.Description()
	assert.Contains(t, desc, "Confluence")
	assert.Contains(t, desc, "search")
}

// mockReadCloser implements io.ReadCloser
type mockReadCloser struct {
	data []byte
	pos  int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}
