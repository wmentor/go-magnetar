package list

import (
	"bytes"
	"encoding/json"
	"os"
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

func newTestTools(t *testing.T, yaml string) *Tool {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS
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

// parseResult parses the result from Exec which can be either a string or *tools.Response
func parseResult(t *testing.T, result any) tools.Response {
	t.Helper()

	switch v := result.(type) {
	case string:
		var resp tools.Response
		if err := parseJSON(v, &resp); err != nil {
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

// parseJSON is a helper to parse JSON from string
func parseJSON(data string, v any) error {
	decoder := json.NewDecoder(bytes.NewReader([]byte(data)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func TestFileListSuccess(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "")

	// Create some test files
	rootDir := g.root.Name()
	if err := os.WriteFile(filepath.Join(rootDir, "file1.txt"), []byte("content1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "file2.go"), []byte("content2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory with files
	subdir := filepath.Join(rootDir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file3.txt"), []byte("content3"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test list without filter
	result, err := g.Exec(t.Context(), nil)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "file1.txt") {
		t.Errorf("Exec() result does not contain file1.txt: %s", resp.Result)
	}

	if !strings.Contains(resp.Result, "file2.go") {
		t.Errorf("Exec() result does not contain file2.go: %s", resp.Result)
	}

	if !strings.Contains(resp.Result, "subdir/file3.txt") {
		t.Errorf("Exec() result does not contain subdir/file3.txt: %s", resp.Result)
	}
}

func TestFileListWithFilter(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "")

	// Create some test files
	rootDir := g.root.Name()
	if err := os.WriteFile(filepath.Join(rootDir, "file1.txt"), []byte("content1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "file2.go"), []byte("content2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "file3.md"), []byte("content3"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test list with filter
	params := &FileListParams{Filter: "*.txt"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "file1.txt") {
		t.Errorf("Exec() result does not contain file1.txt: %s", resp.Result)
	}

	if strings.Contains(resp.Result, "file2.go") {
		t.Errorf("Exec() result should not contain file2.go: %s", resp.Result)
	}

	if strings.Contains(resp.Result, "file3.md") {
		t.Errorf("Exec() result should not contain file3.md: %s", resp.Result)
	}
}

func TestFileListWithGlobPattern(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "")

	// Create some test files
	rootDir := g.root.Name()
	if err := os.WriteFile(filepath.Join(rootDir, "test_file1.txt"), []byte("content1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "test_file2.txt"), []byte("content2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "other.txt"), []byte("content3"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test list with glob pattern
	params := &FileListParams{Filter: "test_*.txt"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "test_file1.txt") {
		t.Errorf("Exec() result does not contain test_file1.txt: %s", resp.Result)
	}

	if !strings.Contains(resp.Result, "test_file2.txt") {
		t.Errorf("Exec() result does not contain test_file2.txt: %s", resp.Result)
	}

	if strings.Contains(resp.Result, "other.txt") {
		t.Errorf("Exec() result should not contain other.txt: %s", resp.Result)
	}
}

func TestFileListWithDirectories(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "")

	// Create directories
	rootDir := g.root.Name()
	subdir1 := filepath.Join(rootDir, "dir1")
	subdir2 := filepath.Join(rootDir, "dir2")
	if err := os.Mkdir(subdir1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(subdir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test list without filter - should include directories
	result, err := g.Exec(t.Context(), nil)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Files should be listed, but directories (except root) should not
	if !strings.Contains(resp.Result, "file.txt") {
		t.Errorf("Exec() result should contain file.txt: %s", resp.Result)
	}

	// Directories should not appear in the list
	if strings.Contains(resp.Result, "dir1") || strings.Contains(resp.Result, "dir2") {
		t.Errorf("Exec() result should not contain directories: %s", resp.Result)
	}
}

func TestFileListEmptyRoot(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "")

	// Test list on empty directory
	result, err := g.Exec(t.Context(), nil)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	// Result should be empty or just error
	if resp.Result != "" && !strings.Contains(resp.Result, "error") {
		t.Errorf("Exec() result = %q, want empty or error", resp.Result)
	}
}
