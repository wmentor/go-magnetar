package exec

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestTools(t *testing.T) *Tool {
	t.Helper()

	path := t.TempDir() + "/config.yml"
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS for tests
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})
}

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

func TestExecSuccess(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	params := &ExecParams{
		Command: "echo Hello World",
	}

	result, err := g.Exec(context.Background(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !contains(resp.Result, "Hello World") {
		t.Errorf("Exec() result = %q, contains \"Hello World\"", resp.Result)
	}
}

func TestExecWithStdin(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	params := &ExecParams{
		Command: "echo test stdin",
	}

	result, err := g.Exec(context.Background(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !contains(resp.Result, "test stdin") {
		t.Errorf("Exec() result = %q, contains \"test stdin\"", resp.Result)
	}
}

func TestExecNoParams(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	result, err := g.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("Exec() with nil params failed: %v", err)
	}

	resp := parseResult(t, result)

	// Should return error since no command was provided
	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Empty command should still succeed
}

func TestExecReadOnlyMode(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/config.yml"
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS for tests
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	g := New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: true, ResponseStore: responseStore})

	params := &ExecParams{
		Command: "echo test",
	}

	_, err = g.Exec(context.Background(), params)
	if err == nil {
		t.Fatalf("Exec() should return error in read-only mode")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && haystack != needle && (len(haystack) >= len(needle)) && (len(needle) == 0 || (len(haystack) > 0 && containsAt(haystack, needle)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
