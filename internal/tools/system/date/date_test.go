package date

import (
	"encoding/json"
	"os"
	"testing"
	"time"

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

func TestSystemDateSuccess(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	result, err := g.Exec(t.Context(), nil)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Verify the date is in the expected format
	_, err = time.Parse(time.UnixDate, resp.Result)
	if err != nil {
		t.Errorf("Exec() result = %q, expected format %q: %v", resp.Result, time.UnixDate, err)
	}

	// Verify the date is approximately now (within 1 second)
	expectedDate := time.Now().Format(time.UnixDate)
	if resp.Result != expectedDate {
		t.Errorf("Exec() result = %q, want %q", resp.Result, expectedDate)
	}
}

func TestSystemDateWithParams(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	// Even with params (should be ignored), should return current date
	params := &SystemDateParams{}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Verify the date is in the expected format
	_, err = time.Parse(time.UnixDate, resp.Result)
	if err != nil {
		t.Errorf("Exec() result = %q, expected format %q: %v", resp.Result, time.UnixDate, err)
	}
}

func TestSystemDateInvalidParams(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	// With nil params should still work (no params needed)
	result, err := g.Exec(t.Context(), nil)
	if err != nil {
		t.Fatalf("Exec() with nil params failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}
}
