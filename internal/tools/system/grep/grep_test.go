package grep

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestTools(t *testing.T, readOnly bool) (*Tool, *os.Root) {
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

	// Create root FS for tests that need file operations
	rootFS, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: readOnly, ResponseStore: responseStore}), rootFS
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

// Ensure rg is available for testing
func ensureRgAvailable(t *testing.T) {
	t.Helper()
	_, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("ripgrep (rg) is not installed, skipping system_grep test")
	}
}

func TestSystemGrepSuccess(t *testing.T) {
	t.Parallel()

	ensureRgAvailable(t)
	g, rootFS := newTestTools(t, false)

	// Create a test file inside rootFS
	filename := "test.txt"
	content := "Hello, World!\nThis is a test file.\nAnother line with test keyword."
	if err := rootFS.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	params := &SystemGrepParams{Filename: filename, Pattern: "test"}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "test") {
		t.Errorf("Exec() result = %q, want to contain 'test'", resp.Result)
	}
}

func TestSystemGrepNoMatch(t *testing.T) {
	t.Parallel()

	ensureRgAvailable(t)
	g, rootFS := newTestTools(t, false)

	// Create a test file inside rootFS
	filename := "test.txt"
	content := "Hello, World!\nThis is a test file."
	if err := rootFS.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	params := &SystemGrepParams{Filename: filename, Pattern: "nonexistent"}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "No match found") {
		t.Errorf("Exec() result = %q, want to contain 'No match found'", resp.Result)
	}
}

func TestSystemGrepDirectory(t *testing.T) {
	t.Parallel()

	ensureRgAvailable(t)
	g, rootFS := newTestTools(t, false)

	// Create a directory with test files inside rootFS
	dirname := "testdir"
	if err := rootFS.MkdirAll(dirname, 0o755); err != nil {
		t.Fatal(err)
	}

	filename1 := filepath.Join(dirname, "file1.txt")
	content1 := "Content with keyword1"
	if err := rootFS.WriteFile(filename1, []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}

	filename2 := filepath.Join(dirname, "file2.txt")
	content2 := "Content with keyword2"
	if err := rootFS.WriteFile(filename2, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	params := &SystemGrepParams{Filename: dirname, Pattern: "keyword"}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "keyword") {
		t.Errorf("Exec() result = %q, want to contain 'keyword'", resp.Result)
	}
}

func TestSystemGrepReadOnlyMode(t *testing.T) {
	t.Parallel()

	ensureRgAvailable(t)
	g, rootFS := newTestTools(t, true)

	// Create a test file inside rootFS
	filename := "test.txt"
	content := "Hello, World!"
	if err := rootFS.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	params := &SystemGrepParams{Filename: filename, Pattern: "World"}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// system_grep should work in read-only mode since it only reads files
	if !strings.Contains(resp.Result, "Hello, World!") {
		t.Errorf("Exec() result = %q, want to contain 'Hello, World!'", resp.Result)
	}
}

func TestSystemGrepInvalidParams(t *testing.T) {
	t.Parallel()

	g, _ := newTestTools(t, false)

	result, err := g.Exec(t.Context(), nil)
	if err == nil {
		t.Errorf("Exec() expected error, got nil")
	}

	if result != nil {
		t.Errorf("Exec() expected nil result for invalid params, got %v", result)
	}
}
