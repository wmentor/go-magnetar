package generic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func newTestTools(t *testing.T, yaml string, readOnly bool) *GenericTools {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	return New(cfg, nil, &plugin.State{Config: cfg, ReadOnly: readOnly})
}

func TestSSHExecEnableFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		readOnly bool
		want     string
	}{
		{
			name: "disabled by default",
			yaml: "",
			want: "error: ssh disabled",
		},
		{
			name: "explicitly disabled",
			yaml: "ssh:\n  enable: false\n",
			want: "error: ssh disabled",
		},
		{
			// With ssh enabled the call must get past the enable check.
			// An empty address stops it right after, before any network access.
			name: "enabled",
			yaml: "ssh:\n  enable: true\n",
			want: "error: no address provided",
		},
		{
			name:     "enabled but read-only mode",
			yaml:     "ssh:\n  enable: true\n",
			readOnly: true,
			want:     "error: the ssh tool is forbidden in read-only mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := newTestTools(t, tt.yaml, tt.readOnly)

			if got := g.SSHExec("ls", "", "", "", ""); got != tt.want {
				t.Errorf("SSHExec() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSystemDate(t *testing.T) {
	t.Parallel()

	g := newTestTools(t, "", false)

	// date(1) prints whole seconds, so truncate the lower bound.
	before := time.Now().Truncate(time.Second)
	got := g.SystemDate()
	after := time.Now()

	parsed, err := time.Parse(time.UnixDate, got)
	if err != nil {
		t.Fatalf("SystemDate() = %q, want time.UnixDate format: %v", got, err)
	}

	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("SystemDate() = %q, want a time between %v and %v", got, before, after)
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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

	// Create a test file with single occurrence
	testFile := "test.txt"
	testContent := "Hello World\nThis is a test file"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test successful edit with single occurrence
	result, ok := g.FileEdit(testFile, "World", "Universe")
	if !ok {
		t.Errorf("FileEdit() returned false for single occurrence, want true: %s", result)
	}
	if !strings.Contains(result, "File edited successfully") {
		t.Errorf("FileEdit() result = %q, want to contain 'File edited successfully'", result)
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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

	// Create a test file with multiple occurrences
	testFile := "test.txt"
	testContent := "Hello World\nThis is a test file\nHello World"
	if err := rootFS.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test edit with multiple occurrences - should fail
	result, ok := g.FileEdit(testFile, "World", "Universe")
	if ok {
		t.Errorf("FileEdit() returned true for multiple occurrences, want false")
	}
	if !strings.Contains(result, "appears multiple times") && !strings.Contains(result, "appears") {
		t.Errorf("FileEdit() result = %q, want to contain 'appears multiple times'", result)
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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

	// Create a test file
	testFile := "test.txt"
	if err := rootFS.WriteFile(testFile, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test edit with non-existent text - returns ok=true with warning
	result, ok := g.FileEdit(testFile, "NotExists", "Modified")
	// FileEdit returns ok=true even when text is not found (it's a warning, not an error)
	if !ok {
		t.Errorf("FileEdit() returned false for non-existent text, want true (warning)")
	}
	// Should get a warning, not an error
	if strings.Contains(result, "error") {
		t.Errorf("FileEdit() result = %q, should not contain 'error'", result)
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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

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

	result, ok := g.FileEditMultiple(testFile, edits, "all")
	if !ok {
		t.Errorf("FileEditMultiple() returned false, want true")
	}
	if !strings.Contains(result, "File edited successfully") {
		t.Errorf("FileEditMultiple() result = %q, want to contain 'File edited successfully'", result)
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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

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

	result, ok := g.FileEditMultiple(testFile, edits, "first")
	if !ok {
		t.Errorf("FileEditMultiple() returned false, want true")
	}
	_ = result // Suppress unused variable warning

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

	g := New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false})

	// Create a test file
	testFile := "test.txt"
	if err := rootFS.WriteFile(testFile, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test with non-existent text
	edits := []FileEditEdit{
		{OldText: "NotExists", NewText: "Modified"},
	}

	result, ok := g.FileEditMultiple(testFile, edits, "all")
	if !ok {
		t.Errorf("FileEditMultiple() returned false, want true")
	}
	if !strings.Contains(result, "no changes were applied") {
		t.Errorf("FileEditMultiple() result = %q, want to contain 'no changes were applied'", result)
	}
}

func newTestToolsWithRoot(t *testing.T) (*GenericTools, *os.Root) {
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

	return New(cfg, rootFS, &plugin.State{Config: cfg, ReadOnly: false}), rootFS
}

// TestFileEditDispatch verifies that file_edit is reachable via GenericTools.Dispatch —
// the same code path used by the plugin.
func TestFileEditDispatch(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	if err := rootFS.WriteFile("hello.txt", []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := g.Dispatch("file_edit", `{"filename":"hello.txt","old_text":"World","new_text":"Go"}`)
	if !strings.Contains(result, "File edited successfully") {
		t.Errorf("Dispatch(file_edit) = %q, want 'File edited successfully'", result)
	}

	data, err := rootFS.ReadFile("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hello Go" {
		t.Errorf("file content after Dispatch = %q, want %q", string(data), "Hello Go")
	}
}

// TestFileEditMultipleDispatch verifies that file_edit_multiple is reachable via GenericTools.Dispatch.
func TestFileEditMultipleDispatch(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	if err := rootFS.WriteFile("hi.txt", []byte("foo bar foo"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := `{"filename":"hi.txt","edits":[{"old_text":"foo","new_text":"baz"}],"strategy":"all"}`
	result := g.Dispatch("file_edit_multiple", args)
	if !strings.Contains(result, "File edited successfully") {
		t.Errorf("Dispatch(file_edit_multiple) = %q, want 'File edited successfully'", result)
	}

	data, err := rootFS.ReadFile("hi.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "baz bar baz" {
		t.Errorf("file content after Dispatch = %q, want %q", string(data), "baz bar baz")
	}
}

// TestFileEditBlocksCodecFormats verifies that file_edit rejects files with codec-handled extensions.
func TestFileEditBlocksCodecFormats(t *testing.T) {
	t.Parallel()

	codecExts := []string{".docx", ".pdf", ".odt", ".xlsx", ".pptx", ".csv", ".tsv", ".html", ".htm"}

	for _, ext := range codecExts {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			g, rootFS := newTestToolsWithRoot(t)

			fname := "testfile" + ext
			if err := rootFS.WriteFile(fname, []byte("some content"), 0o644); err != nil {
				t.Fatal(err)
			}

			result, ok := g.FileEdit(fname, "some", "other")
			if ok {
				t.Errorf("FileEdit(%s) returned ok=true, want false for codec format", ext)
			}
			if !strings.Contains(result, "file_edit cannot edit") {
				t.Errorf("FileEdit(%s) = %q, want to contain 'file_edit cannot edit'", ext, result)
			}
		})
	}
}

// TestFileEditSequentialSafety verifies that two sequential file_edit calls on the
// same file produce the correct cumulative result (second edit sees the output of the first).
func TestFileEditSequentialSafety(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	if err := rootFS.WriteFile("seq.txt", []byte("alpha beta gamma"), 0o644); err != nil {
		t.Fatal(err)
	}

	r1 := g.Dispatch("file_edit", `{"filename":"seq.txt","old_text":"alpha","new_text":"one"}`)
	if !strings.Contains(r1, "File edited successfully") {
		t.Fatalf("first Dispatch(file_edit) = %q", r1)
	}

	r2 := g.Dispatch("file_edit", `{"filename":"seq.txt","old_text":"beta","new_text":"two"}`)
	if !strings.Contains(r2, "File edited successfully") {
		t.Fatalf("second Dispatch(file_edit) = %q", r2)
	}

	data, err := rootFS.ReadFile("seq.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "one two gamma" {
		t.Errorf("file content = %q, want %q", string(data), "one two gamma")
	}
}

func TestStaticDefinitions(t *testing.T) {
	t.Parallel()

	// Test StaticDefinitionFileEdit
	def := StaticDefinitionFileEdit()
	if def.Type != "function" {
		t.Errorf("StaticDefinitionFileEdit().Type = %q, want %q", def.Type, "function")
	}
	if def.Function.Name != "file_edit" {
		t.Errorf("StaticDefinitionFileEdit().Function.Name = %q, want %q", def.Function.Name, "file_edit")
	}

	// Verify required parameters
	params, ok := def.Function.Parameters.(map[string]any)["required"].([]string)
	if !ok {
		t.Errorf("StaticDefinitionFileEdit() required params is not a string slice")
	} else if len(params) != 3 {
		t.Errorf("StaticDefinitionFileEdit() required params count = %d, want 3", len(params))
	}

	// Test StaticDefinitionFileEditMultiple
	defMulti := StaticDefinitionFileEditMultiple()
	if defMulti.Type != "function" {
		t.Errorf("StaticDefinitionFileEditMultiple().Type = %q, want %q", defMulti.Type, "function")
	}
	if defMulti.Function.Name != "file_edit_multiple" {
		t.Errorf("StaticDefinitionFileEditMultiple().Function.Name = %q, want %q", defMulti.Function.Name, "file_edit_multiple")
	}

	// Verify required parameters
	params, ok = defMulti.Function.Parameters.(map[string]any)["required"].([]string)
	if !ok {
		t.Errorf("StaticDefinitionFileEditMultiple() required params is not a string slice")
	} else if len(params) != 2 {
		t.Errorf("StaticDefinitionFileEditMultiple() required params count = %d, want 2", len(params))
	}
}
