package file

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// TestGitLabFileNew tests the New function
func TestGitLabFileNew(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)

	tool := New(cfg, p, clientFactory)
	if tool == nil {
		t.Fatal("New() returned nil")
	}

	if tool.Name() != "gitlab_file" {
		t.Errorf("Name() = %s, want gitlab_file", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
}

// TestGitLabFileParamsValidation tests parameter validation
func TestGitLabFileParamsValidation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)

	tool := New(cfg, p, clientFactory)

	// Test with nil params
	result, err := tool.Exec(t.Context(), nil)
	if err == nil {
		t.Error("Exec() with nil params should return an error")
	}
	if result != nil {
		t.Log("Exec() returned non-nil result with nil params (expected)")
	}
}

// TestGitLabFileExecWithMockServer tests the Exec function with a mock HTTP server
func TestGitLabFileExecWithMockServer(t *testing.T) {
	t.Parallel()

	// Create a mock GitLab server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Verify query parameters
		ref := r.URL.Query().Get("ref")
		if ref != "main" {
			t.Errorf("Expected ref=query=main, got %s", ref)
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-gitlab-token" {
			t.Errorf("Expected Authorization header 'Bearer test-gitlab-token', got %s", authHeader)
		}

		// Return mock response
		response := map[string]any{
			"name":         "main.go",
			"path":         "src/main.go",
			"sha":          "a1b2c3d4e5f6",
			"size":         1234,
			"encoding":     "base64",
			"content":      "ZmZsZiBtYWluKCkgewogIHByaW50bG4oIkhlbGxvIFdvcmxkIikKfQ==", // base64 encoded content
			"checkout":     "",
			"ref":          "main",
			"external_url": "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create config with mock GitLab URL
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-gitlab-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	// Test params
	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "src/main.go",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	// Verify response
	resp, ok := result.(*tools.Response)
	if !ok {
		t.Fatalf("Expected *tools.Response, got %T", result)
	}

	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Verify decoded content (base64 decoded content is: "fflf main() {\n\tprintln(\"Hello World\")\n}")
	// Note: fmt.Fprintf uses spaces instead of tabs by default for indentation
	expectedContent := "fflf main() {\n  println(\"Hello World\")\n}"
	if resp.Result != expectedContent {
		t.Errorf("Result = %q, want %q", resp.Result, expectedContent)
	}
}

// TestGitLabFileExecWithPlainEncoding tests the Exec function with plain text encoding
func TestGitLabFileExecWithPlainEncoding(t *testing.T) {
	t.Parallel()

	// Create a mock GitLab server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock response with plain encoding
		response := map[string]any{
			"name":         "README.md",
			"path":         "README.md",
			"sha":          "abc123",
			"size":         500,
			"encoding":     "text",
			"content":      "# Project README\n\nThis is a test project.",
			"checkout":     "",
			"ref":          "main",
			"external_url": "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create config
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	// Test params
	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "README.md",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp, ok := result.(*tools.Response)
	if !ok {
		t.Fatalf("Expected *tools.Response, got %T", result)
	}

	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}

	expectedContent := "# Project README\n\nThis is a test project."
	if resp.Result != expectedContent {
		t.Errorf("Result = %q, want %q", resp.Result, expectedContent)
	}
}

// TestGitLabFileExecDefaultBranch tests that default branch is 'main'
func TestGitLabFileExecDefaultBranch(t *testing.T) {
	t.Parallel()

	// Create a mock GitLab server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that branch defaults to 'main'
		ref := r.URL.Query().Get("ref")
		if ref != "main" {
			t.Errorf("Expected ref=query=main (default), got %s", ref)
		}

		response := map[string]any{
			"name":         "file.txt",
			"path":         "file.txt",
			"sha":          "sha123",
			"size":         100,
			"encoding":     "base64",
			"content":      "dGVzdCBjb250ZW50", // base64 encoded "test content"
			"checkout":     "",
			"ref":          "main",
			"external_url": "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create config
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	// Test params without branch (should default to 'main')
	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		File:        "file.txt",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp, ok := result.(*tools.Response)
	if !ok {
		t.Fatalf("Expected *tools.Response, got %T", result)
	}

	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}
}

// TestGitLabFileExecFileNotFound tests handling of 404 errors
func TestGitLabFileExecFileNotFound(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "404 File Not Found"}`))
	}))
	defer mockServer.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "nonexistent.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error for 404")
	}
	require.Error(t, err)
}

// TestGitLabFileExecEmptyContent tests handling of empty content
func TestGitLabFileExecEmptyContent(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"name":         "empty.txt",
			"path":         "empty.txt",
			"sha":          "abc123",
			"size":         0,
			"encoding":     "base64",
			"content":      "",
			"checkout":     "",
			"ref":          "main",
			"external_url": "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "empty.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error for empty content")
	}
	require.Error(t, err)
}

// TestGitLabFileExecInvalidBase64 tests handling of invalid base64 encoding
func TestGitLabFileExecInvalidBase64(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"name":         "invalid.txt",
			"path":         "invalid.txt",
			"sha":          "abc123",
			"size":         100,
			"encoding":     "base64",
			"content":      "!!!invalid base64!!!",
			"checkout":     "",
			"ref":          "main",
			"external_url": "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "invalid.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error for invalid base64")
	}
	require.Error(t, err)
}

// TestGitLabFileExecInvalidJSON tests handling of invalid JSON response
func TestGitLabFileExecInvalidJSON(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer mockServer.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "file.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error for invalid JSON")
	}
	require.Error(t, err)
}

// TestGitLabFileExecHTTPError tests handling of HTTP errors
func TestGitLabFileExecHTTPError(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal Server Error"}`))
	}))
	defer mockServer.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: ` + mockServer.URL + `
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory(cfg)
	tool := New(cfg, p, clientFactory)

	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "file.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error for 500 status")
	}
	require.Error(t, err)
}

// TestGitLabFileExecWithMockClientFactory tests using mocked ClientFactory
func TestGitLabFileExecWithMockClientFactory(t *testing.T) {
	t.Parallel()

	// Create a mock client factory
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := `
version: "1.0"
gitlab:
  base_url: https://gitlab.example.com
  api_key: test-token
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()

	// Create a mock ClientFactory
	mockClientFactory := &mockClientFactory{}
	tool := New(cfg, p, mockClientFactory)

	// Test params
	params := &GitLabFileParams{
		ProjectPath: "namespace/project",
		Branch:      "main",
		File:        "file.txt",
	}

	_, err = tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec() should return error when mock client fails")
	}
	require.Error(t, err)

	// Verify that the mock was called
	if !mockClientFactory.called {
		t.Error("Mock client should have been called")
	}
}

// mockClientFactory implements requester.ClientFactory for testing
type mockClientFactory struct {
	called bool
}

func (m *mockClientFactory) Get() requester.Client {
	m.called = true
	return &mockClient{}
}

func (m *mockClientFactory) Put(c requester.Client) {
	_ = c
}

// mockClient implements requester.Client for testing
type mockClient struct{}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	// Return an error to test error handling
	return nil, &httpError{message: "mock HTTP error"}
}

// httpError implements error for testing
type httpError struct {
	message string
}

func (h *httpError) Error() string {
	return h.message
}
