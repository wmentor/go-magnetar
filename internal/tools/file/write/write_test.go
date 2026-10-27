package write

import (
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

func newTestTools(t *testing.T, yaml string, readOnly bool) *Tool {
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

	// Create root FS for tests that need file operations
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: readOnly, ResponseStore: responseStore})
}

func newTestToolsWithRoot(t *testing.T) (*Tool, *os.Root) {
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

	rootFS, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore}), rootFS
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

func TestFileWriteSuccess(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	// Test writing to a new file
	filename := "test.txt"
	content := "Hello, World!\nThis is a test file."
	params := &FileWriteParams{Filename: filename, Content: content}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "File written successfully") {
		t.Errorf("Exec() result = %q, want to contain 'File written successfully'", resp.Result)
	}

	// Verify the file was created
	data, err := rootFS.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestFileWriteWithDirectories(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	// Test writing to a file in a non-existent directory
	filename := "nested/dir/file.txt"
	content := "Content in nested directory."
	params := &FileWriteParams{Filename: filename, Content: content}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "File written successfully") {
		t.Errorf("Exec() result = %q, want to contain 'File written successfully'", resp.Result)
	}

	// Verify the file was created
	data, err := rootFS.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestFileWriteReadOnlyMode(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "", true)

	filename := "test.txt"
	content := "This should fail."
	params := &FileWriteParams{Filename: filename, Content: content}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "read-only mode") {
		t.Errorf("Exec() result = %q, want to contain 'read-only mode'", resp.Result)
	}
}

func TestFileWriteCodecFormat(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	// Test writing to a CSV file (codec format)
	filename := "test.csv"
	content := "name,age\nAlice,30\nBob,25"
	params := &FileWriteParams{Filename: filename, Content: content}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "File written successfully") {
		t.Errorf("Exec() result = %q, want to contain 'File written successfully'", resp.Result)
	}

	// Verify the file was created
	data, err := rootFS.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestFileWriteOverwrite(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	// Create initial file
	filename := "overwrite.txt"
	initialContent := "Initial content."
	if err := rootFS.WriteFile(filename, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Overwrite the file
	newContent := "New content."
	params := &FileWriteParams{Filename: filename, Content: newContent}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "File written successfully") {
		t.Errorf("Exec() result = %q, want to contain 'File written successfully'", resp.Result)
	}

	// Verify the file was overwritten
	data, err := rootFS.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != newContent {
		t.Errorf("file content = %q, want %q", string(data), newContent)
	}
}

func TestFileWriteEmptyContent(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	filename := "empty.txt"
	content := ""
	params := &FileWriteParams{Filename: filename, Content: content}

	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	if !strings.Contains(resp.Result, "File written successfully") {
		t.Errorf("Exec() result = %q, want to contain 'File written successfully'", resp.Result)
	}

	// Verify the file was created with empty content
	data, err := rootFS.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}
