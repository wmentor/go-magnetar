package list

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/archive"
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

	// Create a temporary ZIP file for testing
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a test ZIP file
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "Hello, World!",
		"dir/file2.txt": "Nested file content",
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	params := &Params{
		Filename: zipPath,
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
}

func TestTool_Exec_WithFilter(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a test ZIP file with various file types
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":      "Text file 1",
		"file2.txt":      "Text file 2",
		"file3.md":       "Markdown file",
		"docs/readme.md": "Documentation",
		"src/main.go":    "Go source",
		"src/util.go":    "Go utility",
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedFiles []string
	}{
		{
			name:          "filter by extension *.txt",
			filter:        "*.txt",
			expectedCount: 2,
			expectedFiles: []string{"file1.txt", "file2.txt"},
		},
		{
			name:          "filter by extension *.md",
			filter:        "*.md",
			expectedCount: 2,
			expectedFiles: []string{"file3.md", "docs/readme.md"},
		},
		{
			name:          "filter by extension *.go",
			filter:        "*.go",
			expectedCount: 2,
			expectedFiles: []string{"src/main.go", "src/util.go"},
		},
		{
			name:          "filter with no matches",
			filter:        "*.json",
			expectedCount: 0,
			expectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &Params{
				Filename: zipPath,
				Filter:   tt.filter,
			}

			result, err := tool.Exec(context.Background(), params)
			if err != nil {
				t.Fatalf("Exec() error = %v", err)
			}

			response, ok := result.(*tools.Response)
			if !ok {
				t.Fatalf("Exec() returned %T, want *tools.Response", result)
			}

			// Parse the result
			var files []archive.FileInfo
			if err := json.Unmarshal([]byte(response.Result), &files); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(files))
			}

			// Check that expected files are present
			fileNames := make(map[string]bool)
			for _, f := range files {
				fileNames[f.Name] = true
			}

			for _, expected := range tt.expectedFiles {
				if !fileNames[expected] {
					t.Errorf("Expected file %q not found in results", expected)
				}
			}
		})
	}
}

func TestTool_Exec_RelativePath(t *testing.T) {
	t.Parallel()

	// Create a temporary ZIP file for testing
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a test ZIP file
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "Hello, World!",
		"dir/file2.txt": "Nested file content",
	})

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	// Change to tmpDir to test relative path
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	params := &Params{
		Filename: "test.zip",
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

	// Create an empty file
	if err := os.WriteFile(rarPath, []byte{}, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	params := &Params{
		Filename: rarPath,
	}

	_, err := tool.Exec(context.Background(), params)
	if err == nil {
		t.Error("Exec() expected error for unsupported format, got nil")
	}
}

func TestTool_Name(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	tool := NewTool(cfg, &mockPrinter{})

	if name := tool.Name(); name != "archive_list" {
		t.Errorf("Name() = %q, want %q", name, "archive_list")
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

func TestFilterFiles(t *testing.T) {
	t.Parallel()

	files := []archive.FileInfo{
		{Name: "file1.txt", IsDir: false},
		{Name: "file2.txt", IsDir: false},
		{Name: "file3.md", IsDir: false},
		{Name: "docs/readme.md", IsDir: false},
		{Name: "src/main.go", IsDir: false},
		{Name: "src/util.go", IsDir: false},
		{Name: "docs", IsDir: true},
		{Name: "src", IsDir: true},
	}

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "filter *.txt",
			pattern:       "*.txt",
			expectedCount: 2,
			expectedNames: []string{"file1.txt", "file2.txt"},
		},
		{
			name:          "filter *.md",
			pattern:       "*.md",
			expectedCount: 2,
			expectedNames: []string{"file3.md", "docs/readme.md"},
		},
		{
			name:          "filter *.go",
			pattern:       "*.go",
			expectedCount: 2,
			expectedNames: []string{"src/main.go", "src/util.go"},
		},
		{
			name:          "filter with no matches",
			pattern:       "*.json",
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterFiles(files, tt.pattern)

			if len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(filtered))
			}

			// Check that expected files are present
			fileNames := make(map[string]bool)
			for _, f := range filtered {
				fileNames[f.Name] = true
			}

			for _, expected := range tt.expectedNames {
				if !fileNames[expected] {
					t.Errorf("Expected file %q not found in results", expected)
				}
			}
		})
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
