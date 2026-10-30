package rag_test

import (
	"encoding/json"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

// mockStore implements rag.Store interface for testing
type mockStore struct {
	results map[string]string
	saved   map[string]bool
}

func (m *mockStore) Name() string {
	return "mock"
}

func (m *mockStore) RagSearch(query string) string {
	if result, ok := m.results[query]; ok {
		return result
	}
	return ""
}

func (m *mockStore) RagSave(content string, prepend string, part int) bool {
	if m.saved == nil {
		m.saved = make(map[string]bool)
	}
	key := content + prepend + string(rune(part))
	if result, ok := m.saved[key]; ok {
		return result
	}
	return true
}

// parseResult parses the result from Exec which can be either a string or *tools.Response
func parseResult(t *testing.T, result any) tools.Response {
	t.Helper()

	switch v := result.(type) {
	case string:
		var resp tools.Response
		if err := json.Unmarshal([]byte(v), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		return resp
	case *tools.Response:
		return *v
	default:
		t.Fatalf("unexpected result type: %T", result)
	}
	panic("unreachable")
}

func TestToolName(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t)

	if tool.Name() != "rag_search" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "rag_search")
	}
}

func TestToolDescription(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t)

	if tool.Description() != "Search the knowledge base for relevant information" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Search the knowledge base for relevant information")
	}
}

func TestExecWithNilParams(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t)

	result, err := tool.Exec(t.Context(), nil)

	if err == nil {
		t.Errorf("Exec() expected error with nil params, got nil")
	}

	if result != nil {
		t.Errorf("Exec() expected nil result with nil params, got %v", result)
	}
}

func TestExecWithEmptyQuery(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t)

	params := &rag.RagSearchParams{
		Query: "",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	switch v := result.(type) {
	case *tools.Response:
		if v.Status != tools.StatusOK {
			t.Errorf("Status = %q, want %q", v.Status, tools.StatusOK)
		}
	default:
		t.Errorf("unexpected result type: %T", v)
	}
}

func TestExecWithMockStore(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t)
	mock := &mockStore{
		results: map[string]string{
			"test query": "test result 1\n\n---\n\ntest result 2",
		},
	}
	tool := rag.New(cfg, printer.GetDefault(), mock, &plugin.State{Config: cfg})

	params := &rag.RagSearchParams{
		Query: "test query",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)
	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}
	if resp.Result != "test result 1\n\n---\n\ntest result 2" {
		t.Errorf("Result = %q, want %q", resp.Result, "test result 1\n\n---\n\ntest result 2")
	}
}

func TestExecWithNoResults(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t)
	mock := &mockStore{
		results: map[string]string{},
	}
	tool := rag.New(cfg, printer.GetDefault(), mock, &plugin.State{Config: cfg})

	params := &rag.RagSearchParams{
		Query: "nonexistent query",
	}

	result, err := tool.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)
	if resp.Status != tools.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, tools.StatusOK)
	}
	if resp.Result != "no relevant information found" {
		t.Errorf("Result = %q, want %q", resp.Result, "no relevant information found")
	}
}

func TestStoreInterface(t *testing.T) {
	t.Parallel()

	// Test that mockStore implements rag.Store interface
	var _ rag.Store = &mockStore{}

	t.Log("mockStore implements Store interface")
}

func loadTestConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load("./testdata/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newTestTool(t *testing.T) *rag.Tool {
	t.Helper()

	// Create a mock store for testing
	mock := &mockStore{
		results: map[string]string{},
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	p := printer.GetDefault()

	// Create a config with RAG enabled
	cfg := loadTestConfig(t)

	return rag.New(cfg, p, mock, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})
}
