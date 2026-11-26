package read

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	// Import archive implementations to register them
	_ "github.com/wmentor/go-magnetar/internal/archive/zip"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type mockPrinter struct{}

func (m *mockPrinter) Verbose(enable bool)                                     {}
func (m *mockPrinter) Debug(message string, params ...any)                     {}
func (m *mockPrinter) Info(message string, params ...any)                      {}
func (m *mockPrinter) Warn(message string, params ...any)                      {}
func (m *mockPrinter) Error(message string, params ...any)                     {}
func (m *mockPrinter) ToolCall(typeIcon string, message string, params ...any) {}
func (m *mockPrinter) Print(typeIcon string, message string, params ...any)    {}
func (m *mockPrinter) PrintRaw(data string)                                    {}
func (m *mockPrinter) Ask(question string) string                              { return "" }
func (m *mockPrinter) ToolEmptyLine()                                          {}
func (m *mockPrinter) EmptyLine()                                              {}

func TestTool_Exec(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	testContent := "Hello, World!"
	createTestZip(t, zipPath, map[string]string{
		"test.txt": testContent,
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	params := &Params{
		ArchivePath: zipPath,
		FilePath:    "test.txt",
	}

	result, err := tool.Exec(context.Background(), params)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	response, ok := result.(*tools.Response)
	if !ok {
		t.Fatalf("Exec() returned %T, want *tools.Response", result)
	}

	if response.Status != tools.StatusOK {
		t.Errorf("Exec() status = %v, want %v", response.Status, tools.StatusOK)
	}

	if response.Result != testContent {
		t.Errorf("Exec() result = %q, want %q", response.Result, testContent)
	}
}

func TestTool_Exec_RelativeArchivePath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	testContent := "Hello, World!"
	createTestZip(t, zipPath, map[string]string{
		"test.txt": testContent,
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	// Change to tmpDir to test relative path
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	params := &Params{
		ArchivePath: "test.zip",
		FilePath:    "test.txt",
	}

	result, err := tool.Exec(context.Background(), params)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	response, ok := result.(*tools.Response)
	if !ok {
		t.Fatalf("Exec() returned %T, want *tools.Response", result)
	}

	if response.Status != tools.StatusOK {
		t.Errorf("Exec() status = %v, want %v", response.Status, tools.StatusOK)
	}

	if response.Result != testContent {
		t.Errorf("Exec() result = %q, want %q", response.Result, testContent)
	}
}

func TestTool_Exec_NilParams(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	_, err := tool.Exec(context.Background(), nil)
	if err == nil {
		t.Error("Exec() expected error for nil params, got nil")
	}
}

func TestTool_Exec_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	rarPath := filepath.Join(tmpDir, "test.rar")

	if err := os.WriteFile(rarPath, []byte{}, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	params := &Params{
		ArchivePath: rarPath,
		FilePath:    "test.txt",
	}

	_, err := tool.Exec(context.Background(), params)
	if err == nil {
		t.Error("Exec() expected error for unsupported format, got nil")
	}
}

func TestTool_Exec_FileNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"test.txt": "content",
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	params := &Params{
		ArchivePath: zipPath,
		FilePath:    "nonexistent.txt",
	}

	_, err := tool.Exec(context.Background(), params)
	if err == nil {
		t.Error("Exec() expected error for non-existent file, got nil")
	}
}

func TestTool_Name(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	if name := tool.Name(); name != "archive_read" {
		t.Errorf("Name() = %q, want %q", name, "archive_read")
	}
}

func TestTool_Description(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	if desc := tool.Description(); desc == "" {
		t.Error("Description() returned empty string")
	}
}

// createTestZip creates a ZIP file with the given files.
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create file in zip: %v", err)
		}

		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
	}
}
