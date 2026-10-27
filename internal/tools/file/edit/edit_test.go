package edit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// testPrinter - a simple Printer implementation for testing
type testPrinter struct {
	output *bytes.Buffer
}

func (p *testPrinter) Print(icon string, msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Info(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Warn(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Error(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Debug(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) ToolCall(icon string, msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) EmptyLine() {
	// Not needed for these tests
}

func (p *testPrinter) ToolEmptyLine() {
	// Not needed for these tests
}

func (p *testPrinter) Verbose(enabled bool) {
	// Not needed for these tests
}

func (p *testPrinter) Ask(question string) string {
	// Not needed for these tests
	return ""
}

func (p *testPrinter) PrintRaw(msg string) {
	p.output.WriteString(msg)
}

func TestOutputDiff_BasicFunctionality(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Hello World"
	newText := "Hello Universe"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}
}

func TestOutputDiff_MultipleEdits(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	edits := []struct {
		oldText string
		newText string
	}{
		{"World", "Universe"},
		{"Hello", "Hi"},
	}

	for _, edit := range edits {
		outputDiff(tp, edit.oldText, edit.newText)
	}

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}
}

func TestFileEditSingleOccurrence(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file with single occurrence
	testFile := "test.txt"
	testContent := "Hello World\nThis is a test file"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test successful edit with single occurrence
	result, err := g.FileEdit(context.Background(), &FileEditParams{
		Filename: testFile,
		OldText:  "World",
		NewText:  "Universe",
	})
	if err != nil {
		t.Fatalf("FileEdit() returned error: %v", err)
	}
	if resp, ok := result.(*tools.Response); ok {
		if resp.Status != tools.StatusOK {
			t.Errorf("FileEdit() returned status %s, want %s: %s", resp.Status, tools.StatusOK, resp.Result)
		}
		if !strings.Contains(resp.Result, "File edited successfully") {
			t.Errorf("FileEdit() result = %q, want to contain 'File edited successfully'", resp.Result)
		}
	} else {
		t.Errorf("FileEdit() returned unexpected type: %T", result)
	}

	// Verify the file was modified
	content, err := rootFS.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Hello Universe\nThis is a test file"
	if string(content) != expected {
		t.Errorf("File content after edit = %q, want %q", string(content), expected)
	}
}

func TestFileEditMultipleOccurrences(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file with multiple occurrences
	testFile := "test.txt"
	testContent := "Hello World\nThis is a test file\nHello World"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test edit with multiple occurrences - should fail
	_, err = g.FileEdit(context.Background(), &FileEditParams{
		Filename: testFile,
		OldText:  "World",
		NewText:  "Universe",
	})
	if err == nil {
		t.Errorf("FileEdit() returned nil error for multiple occurrences, want error")
	}
	if !strings.Contains(err.Error(), "appears multiple times") && !strings.Contains(err.Error(), "appears") {
		t.Errorf("FileEdit() error = %v, want to contain 'appears multiple times'", err)
	}

	// Verify the file was NOT modified
	content, err := rootFS.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Hello World\nThis is a test file\nHello World" {
		t.Errorf("File content after edit should not change")
	}
}

func TestFileEditNotFound(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file
	testFile := "test.txt"
	if err := rootFS.WriteFile(testFile, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test edit with non-existent text - returns warning (ok=true)
	result, err := g.FileEdit(context.Background(), &FileEditParams{
		Filename: testFile,
		OldText:  "NotExists",
		NewText:  "Modified",
	})
	if err != nil {
		t.Fatalf("FileEdit() returned error: %v", err)
	}
	if resp, ok := result.(*tools.Response); ok {
		// FileEdit returns ok=true even when text is not found (it's a warning, not an error)
		if resp.Status != tools.StatusOK {
			t.Errorf("FileEdit() returned status %s, want %s (warning)", resp.Status, tools.StatusOK)
		}
		// Should get a warning, not an error
		if strings.Contains(resp.Result, "error") {
			t.Errorf("FileEdit() result = %q, should not contain 'error'", resp.Result)
		}
	} else {
		t.Errorf("FileEdit() returned unexpected type: %T", result)
	}
}

func TestFileEditMultipleAllStrategy(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file
	testFile := "test.txt"
	testContent := "Hello World\nHello World\nHello World"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test multiple edits with "all" strategy
	edits := []FileEditEdit{
		{OldText: "World", NewText: "Universe"},
		{OldText: "Hello", NewText: "Hi"},
	}

	result, err := g.FileEditMultiple(context.Background(), &FileEditParamsMultiple{
		Filename: testFile,
		Edits:    edits,
		Strategy: "all",
	})
	if err != nil {
		t.Fatalf("FileEditMultiple() returned error: %v", err)
	}
	if resp, ok := result.(*tools.Response); ok {
		if resp.Status != tools.StatusOK {
			t.Errorf("FileEditMultiple() returned status %s, want %s", resp.Status, tools.StatusOK)
		}
		if !strings.Contains(resp.Result, "File edited successfully") {
			t.Errorf("FileEditMultiple() result = %q, want to contain 'File edited successfully'", resp.Result)
		}
	} else {
		t.Errorf("FileEditMultiple() returned unexpected type: %T", result)
	}

	// Verify the file was modified
	content, err := rootFS.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Hi Universe\nHi Universe\nHi Universe"
	if string(content) != expected {
		t.Errorf("File content after edit = %q, want %q", string(content), expected)
	}
}

func TestFileEditMultipleFirstStrategy(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file
	testFile := "test.txt"
	testContent := "Hello World\nHello World\nHello World"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test multiple edits with "first" strategy
	edits := []FileEditEdit{
		{OldText: "World", NewText: "Universe"},
		{OldText: "Hello", NewText: "Hi"},
	}

	result, err := g.FileEditMultiple(context.Background(), &FileEditParamsMultiple{
		Filename: testFile,
		Edits:    edits,
		Strategy: "first",
	})
	if err != nil {
		t.Fatalf("FileEditMultiple() returned error: %v", err)
	}
	if resp, ok := result.(*tools.Response); ok {
		if resp.Status != tools.StatusOK {
			t.Errorf("FileEditMultiple() returned status %s, want %s", resp.Status, tools.StatusOK)
		}
	} else {
		t.Errorf("FileEditMultiple() returned unexpected type: %T", result)
	}

	// Verify the file was modified - only first occurrence should be changed
	content, err := rootFS.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Hi Universe\nHello World\nHello World"
	if string(content) != expected {
		t.Errorf("File content after edit = %q, want %q", string(content), expected)
	}
}

func TestFileEditMultipleNoEdits(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	// Create a test file
	testFile := "test.txt"
	if err := rootFS.WriteFile(testFile, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test with non-existent text
	edits := []FileEditEdit{
		{OldText: "NotExists", NewText: "Modified"},
	}

	result, err := g.FileEditMultiple(context.Background(), &FileEditParamsMultiple{
		Filename: testFile,
		Edits:    edits,
		Strategy: "all",
	})
	if err != nil {
		t.Fatalf("FileEditMultiple() returned error: %v", err)
	}
	if resp, ok := result.(*tools.Response); ok {
		if resp.Status != tools.StatusOK {
			t.Errorf("FileEditMultiple() returned status %s, want %s", resp.Status, tools.StatusOK)
		}
		if !strings.Contains(resp.Result, "no changes were applied") {
			t.Errorf("FileEditMultiple() result = %q, want to contain 'no changes were applied'", resp.Result)
		}
	} else {
		t.Errorf("FileEditMultiple() returned unexpected type: %T", result)
	}
}

func TestFileEditReadonlyMode(t *testing.T) {
	t.Parallel()

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
	defer rootFS.Close()

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	tp := &testPrinter{output: &bytes.Buffer{}}
	g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: true, ResponseStore: responseStore})

	// Create a test file
	testFile := "test.txt"
	if err := rootFS.WriteFile(testFile, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test in read-only mode
	_, err = g.FileEdit(context.Background(), &FileEditParams{
		Filename: testFile,
		OldText:  "World",
		NewText:  "Universe",
	})
	if err == nil {
		t.Errorf("FileEdit() in read-only mode returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "read-only mode") {
		t.Errorf("FileEdit() error = %v, want to contain 'read-only mode'", err)
	}
}

func TestFileEditCodecFormats(t *testing.T) {
	t.Parallel()

	codecExts := []string{".docx", ".pdf", ".odt", ".xlsx", ".pptx", ".csv", ".tsv", ".html", ".htm"}

	for _, ext := range codecExts {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

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
			defer rootFS.Close()

			responseStore := response.New(plugin.DefaultResponseStoreSize)
			tp := &testPrinter{output: &bytes.Buffer{}}
			g := NewTool(cfg, tp, nil, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

			fname := "testfile" + ext
			if err := rootFS.WriteFile(fname, []byte("some content"), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err = g.FileEdit(context.Background(), &FileEditParams{
				Filename: fname,
				OldText:  "some",
				NewText:  "other",
			})
			if err == nil {
				t.Errorf("FileEdit(%s) returned nil error, want error for codec format", ext)
			}
			if !strings.Contains(err.Error(), "file_edit cannot edit") {
				t.Errorf("FileEdit(%s) error = %v, want to contain 'file_edit cannot edit'", ext, err)
			}
		})
	}
}

// Helper function for tests (copied from edit.go)
func outputDiff(p printer.Printer, oldText string, newText string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)
	p.PrintRaw(dmp.DiffPrettyText(diffs))
}
