package lastresponse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestTool(t *testing.T) *Tool {
	t.Helper()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, responseStore)
}

func parseResult(t *testing.T, result any) tools.Response {
	t.Helper()

	switch v := result.(type) {
	case string:
		var resp tools.Response
		// For string results, parse as JSON
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

func TestLastResponseSuccess(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Push some responses
	g.responseStore.Push("First response")
	g.responseStore.Push("Second response")
	g.responseStore.Push("Third response")

	// Test index 0 (most recent)
	params := &ResponseParams{Index: 0}
	result, err := g.Exec(t.Context(), params)
	assert.NoError(t, err)

	resp := parseResult(t, result)

	assert.Equal(t, tools.StatusOK, resp.Status)
	assert.Contains(t, resp.Result, "Third response")
	assert.Contains(t, resp.Result, "index 0")
}

func TestLastResponseIndex1(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Push some responses
	g.responseStore.Push("First response")
	g.responseStore.Push("Second response")
	g.responseStore.Push("Third response")

	// Test index 1 (second most recent)
	params := &ResponseParams{Index: 1}
	result, err := g.Exec(t.Context(), params)
	assert.NoError(t, err)

	resp := parseResult(t, result)

	assert.Equal(t, tools.StatusOK, resp.Status)
	assert.Contains(t, resp.Result, "Second response")
	assert.Contains(t, resp.Result, "index 1")
}

func TestLastResponseIndex2(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Push some responses
	g.responseStore.Push("First response")
	g.responseStore.Push("Second response")
	g.responseStore.Push("Third response")

	// Test index 2 (third most recent)
	params := &ResponseParams{Index: 2}
	result, err := g.Exec(t.Context(), params)
	assert.NoError(t, err)

	resp := parseResult(t, result)

	assert.Equal(t, tools.StatusOK, resp.Status)
	assert.Contains(t, resp.Result, "First response")
	assert.Contains(t, resp.Result, "index 2")
}

func TestLastResponseInvalidIndex(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Test negative index
	params := &ResponseParams{Index: -1}
	result, err := g.Exec(t.Context(), params)
	assert.NoError(t, err)

	resp := parseResult(t, result)

	assert.Equal(t, tools.StatusOK, resp.Status)
	assert.Contains(t, resp.Result, "index must be non-negative")
}

func TestLastResponseOutOfBounds(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Test index out of bounds (empty store)
	params := &ResponseParams{Index: 0}
	result, err := g.Exec(t.Context(), params)
	assert.NoError(t, err)

	resp := parseResult(t, result)

	assert.Equal(t, tools.StatusOK, resp.Status)
	assert.Contains(t, resp.Result, "no response found at the specified index")
}

func TestLastResponseNilParams(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	// Test nil params - should return error
	result, err := g.Exec(t.Context(), nil)
	// The tool returns error when params is nil, but the test should verify the error is expected
	assert.Error(t, err)
	assert.Nil(t, result)
}
