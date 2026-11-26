package zip

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/archive"
)

func TestArchive_List(t *testing.T) {
	t.Parallel()

	// Create a temporary ZIP file for testing
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a test ZIP file
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":         "Hello, World!",
		"dir/file2.txt":     "Nested file content",
		"dir/sub/file3.txt": "Deeply nested file",
	})

	a := &Archive{}

	files, err := a.List(zipPath)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(files) != 3 {
		t.Errorf("List() returned %d files, want 3", len(files))
	}

	// Check that all expected files are present
	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Name] = true
		if f.Size <= 0 {
			t.Errorf("File %s has invalid size: %d", f.Name, f.Size)
		}
	}

	expectedFiles := []string{"file1.txt", "dir/file2.txt", "dir/sub/file3.txt"}
	for _, expected := range expectedFiles {
		if !fileNames[expected] {
			t.Errorf("List() missing expected file: %s", expected)
		}
	}
}

func TestArchive_ReadFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	testContent := "Hello, World!"
	createTestZip(t, zipPath, map[string]string{
		"test.txt": testContent,
	})

	a := &Archive{}

	// Test reading existing file
	data, err := a.ReadFile(zipPath, "test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != testContent {
		t.Errorf("ReadFile() = %q, want %q", string(data), testContent)
	}
}

func TestArchive_ReadFile_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"test.txt": "content",
	})

	a := &Archive{}

	// Test reading non-existent file
	_, err := a.ReadFile(zipPath, "nonexistent.txt")
	if err == nil {
		t.Error("ReadFile() expected error for non-existent file, got nil")
	}
}

func TestArchive_List_InvalidArchive(t *testing.T) {
	t.Parallel()

	a := &Archive{}

	// Test with non-existent file
	_, err := a.List("/nonexistent/path/archive.zip")
	if err == nil {
		t.Error("List() expected error for non-existent file, got nil")
	}
}

func TestArchive_ReadFile_InvalidArchive(t *testing.T) {
	t.Parallel()

	a := &Archive{}

	// Test with non-existent file
	_, err := a.ReadFile("/nonexistent/path/archive.zip", "test.txt")
	if err == nil {
		t.Error("ReadFile() expected error for non-existent file, got nil")
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

func TestArchive_ReadFile_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	testContent := "Test content"
	createTestZip(t, zipPath, map[string]string{
		"TestFile.txt": testContent,
	})

	a := &Archive{}

	// Test case-insensitive match
	data, err := a.ReadFile(zipPath, "testfile.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != testContent {
		t.Errorf("ReadFile() = %q, want %q", string(data), testContent)
	}
}

func TestArchive_FileInfo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"test.txt": "content",
	})

	a := &Archive{}

	files, err := a.List(zipPath)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("List() returned %d files, want 1", len(files))
	}

	file := files[0]

	if file.Name != "test.txt" {
		t.Errorf("File.Name = %q, want %q", file.Name, "test.txt")
	}

	if file.Size != 7 { // len("content")
		t.Errorf("File.Size = %d, want 7", file.Size)
	}

	if file.IsDir {
		t.Error("File.IsDir = true, want false")
	}

	// ModTime should not be zero (ZIP format has limited precision, so we just check it's not zero)
	if file.ModTime.IsZero() {
		t.Error("File.ModTime is zero")
	}
}

func TestArchive_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Verify that Archive implements archive.Archive interface
	var _ archive.Archive = &Archive{}
}
