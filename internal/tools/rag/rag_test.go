package rag_test

import (
	"encoding/json"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

var testConfig *config.Config

func init() {
	var err error
	testConfig, err = config.Load("./testdata/config.yml")
	if err != nil {
		panic(err)
	}
}

type mockStore struct {
	searchResults map[string]string
	saveResults   map[string]bool
}

func (m *mockStore) RagSearch(query string) string {
	if result, ok := m.searchResults[query]; ok {
		return result
	}
	return ""
}

func (m *mockStore) RagSave(content string, prepend string, part int) bool {
	if m.saveResults == nil {
		m.saveResults = make(map[string]bool)
	}
	key := content + prepend + string(rune(part))
	if result, ok := m.saveResults[key]; ok {
		return result
	}
	return true
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *config.Config
		store     rag.Store
		wantErr   bool
		errString string
	}{
		{
			name:    "valid config and store",
			cfg:     testConfig,
			store:   &mockStore{},
			wantErr: false,
		},
		{
			name:      "nil config",
			cfg:       nil,
			store:     &mockStore{},
			wantErr:   true,
			errString: "invalid value of config",
		},
		{
			name:      "nil store",
			cfg:       testConfig,
			store:     nil,
			wantErr:   true,
			errString: "invalid value of verctor store",
		},
		{
			name:      "store as nil interface",
			cfg:       testConfig,
			store:     rag.Store(nil),
			wantErr:   true,
			errString: "invalid value of verctor store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := rag.New(tt.cfg, tt.store)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if tt.errString != "" && err.Error() != tt.errString {
					t.Errorf("New() error = %v, want %q", err, tt.errString)
				}
				return
			}
			if r == nil {
				t.Error("New() returned nil result without error")
			}
		})
	}
}

func TestDefinitionSearch(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	r, err := rag.New(testConfig, store)
	if err != nil {
		t.Fatal("New() failed:", err)
	}

	tool := r.DefinitionSearch()

	if tool.Type != "function" {
		t.Errorf("Tool.Type = %s, want %s", tool.Type, "function")
	}

	if tool.Function == nil {
		t.Fatal("Tool.Function is nil")
	}

	if tool.Function.Name != "rag_search" {
		t.Errorf("Tool.Function.Name = %s, want %s", tool.Function.Name, "rag_search")
	}

	if tool.Function.Description != "Search the knowledge base for relevant information" {
		t.Errorf("Tool.Function.Description = %s, want %s",
			tool.Function.Description, "Search the knowledge base for relevant information")
	}

	params := tool.Function.Parameters
	if params == nil {
		t.Fatal("Tool.Function.Parameters is nil")
	}

	propsMap, ok := params.(map[string]any)
	if !ok {
		t.Fatal("Tool.Function.Parameters is not map[string]any")
	}

	props, ok := propsMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("Tool.Function.Parameters.properties is not map[string]any")
	}

	if _, ok := props["query"]; !ok {
		t.Error("Tool.Function.Parameters.properties.query is missing")
	}

	queryProp, ok := props["query"].(map[string]any)
	if !ok {
		t.Fatal("Tool.Function.Parameters.properties.query is not map[string]any")
	}

	if queryProp["type"] != "string" {
		t.Errorf("Tool.Function.Parameters.properties.query.type = %v, want %v", queryProp["type"], "string")
	}

	if queryProp["description"] != "Search query" {
		t.Errorf("Tool.Function.Parameters.properties.query.description = %v, want %v",
			queryProp["description"], "Search query")
	}

	req, ok := propsMap["required"].([]string)
	if !ok {
		t.Fatalf("Tool.Function.Parameters.required is not []string, got %T", propsMap["required"])
	}

	if len(req) != 1 {
		t.Errorf("Tool.Function.Parameters.required has %d items, want %d", len(req), 1)
	}

	if req[0] != "query" {
		t.Errorf("Tool.Function.Parameters.required[0] = %v, want %v", req[0], "query")
	}
}

func TestRagSave(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	r, err := rag.New(testConfig, store)
	if err != nil {
		t.Fatal("New() failed:", err)
	}

	result := r.RagSave("test content", "test prepend", 1)
	if !result {
		t.Error("RagSave() returned false, expected true")
	}
}

func TestDispatch(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		searchResults: map[string]string{
			"test query": "search result 1\n\n---\n\nsearch result 2",
		},
	}
	r, err := rag.New(testConfig, store)
	if err != nil {
		t.Fatal("New() failed:", err)
	}

	t.Run("valid rag_search call", func(t *testing.T) {
		t.Parallel()

		args, err := json.Marshal(map[string]string{"query": "test query"})
		if err != nil {
			t.Fatal("failed to marshal args:", err)
		}

		result := r.Dispatch("rag_search", string(args))
		if result != "search result 1\n\n---\n\nsearch result 2" {
			t.Errorf("Dispatch() = %q, want %q", result, "search result 1\n\n---\n\nsearch result 2")
		}
	})

	t.Run("rag_search with empty result", func(t *testing.T) {
		t.Parallel()

		args, err := json.Marshal(map[string]string{"query": "empty query"})
		if err != nil {
			t.Fatal("failed to marshal args:", err)
		}

		result := r.Dispatch("rag_search", string(args))
		if result != "no relevant information found" {
			t.Errorf("Dispatch() = %q, want %q", result, "no relevant information found")
		}
	})

	t.Run("rag_search with invalid JSON", func(t *testing.T) {
		t.Parallel()

		result := r.Dispatch("rag_search", "invalid json")
		if result != "error: failed to parse arguments" {
			t.Errorf("Dispatch() = %q, want %q", result, "error: failed to parse arguments")
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		t.Parallel()

		result := r.Dispatch("unknown_tool", "{}")
		if result != "error: unknown tool unknown_tool" {
			t.Errorf("Dispatch() = %q, want %q", result, "error: unknown tool unknown_tool")
		}
	})

	t.Run("rag_search with missing query field", func(t *testing.T) {
		t.Parallel()

		args, err := json.Marshal(map[string]string{"other": "value"})
		if err != nil {
			t.Fatal("failed to marshal args:", err)
		}

		result := r.Dispatch("rag_search", string(args))
		if result != "no relevant information found" {
			t.Errorf("Dispatch() = %q, want %q", result, "no relevant information found")
		}
	})
}

func TestStoreInterface(t *testing.T) {
	t.Parallel()

	// Test that mockStore implements Store interface
	var _ rag.Store = &mockStore{}

	t.Log("mockStore implements Store interface")
}
