package indexer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestListIndexableFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string // paths relative to the root, created as empty files
		want  []string // expected result, relative to the root
	}{
		{
			name:  "empty directory",
			files: nil,
			want:  nil,
		},
		{
			name:  "single supported file",
			files: []string{"guide.md"},
			want:  []string{"guide.md"},
		},
		{
			name: "all supported extensions",
			files: []string{
				"a.md", "b.txt", "c.csv", "d.tsv", "e.docx",
				"f.pdf", "g.odt", "h.pptx", "i.xlsx", "j.html", "k.htm",
			},
			want: []string{
				"a.md", "b.txt", "c.csv", "d.tsv", "e.docx",
				"f.pdf", "g.odt", "h.pptx", "i.xlsx", "j.html", "k.htm",
			},
		},
		{
			name:  "nested directories",
			files: []string{"top.md", "docs/api.md", "docs/deep/er/notes.txt"},
			want:  []string{"docs/api.md", "docs/deep/er/notes.txt", "top.md"},
		},
		{
			name:  "unsupported extensions are skipped",
			files: []string{"keep.md", "image.png", "binary.exe", "main.go", "Makefile"},
			want:  []string{"keep.md"},
		},
		{
			name:  "extension match is case insensitive",
			files: []string{"REPORT.PDF", "Notes.TXT"},
			want:  []string{"Notes.TXT", "REPORT.PDF"},
		},
		{
			name:  "git directory is skipped",
			files: []string{"keep.md", ".git/config.txt", ".git/hooks/readme.md"},
			want:  []string{"keep.md"},
		},
		{
			name:  "node_modules is skipped",
			files: []string{"keep.md", "node_modules/pkg/readme.md"},
			want:  []string{"keep.md"},
		},
		{
			name:  "pycache is skipped",
			files: []string{"keep.md", "__pycache__/cache.txt"},
			want:  []string{"keep.md"},
		},
		{
			name:  "excluded directory nested deeper is skipped",
			files: []string{"docs/keep.md", "docs/node_modules/readme.md", "docs/sub/.git/x.txt"},
			want:  []string{"docs/keep.md"},
		},
		{
			name:  "excluded name only matches directories",
			files: []string{"node_modules.md", "docs/.git.txt"},
			want:  []string{"docs/.git.txt", "node_modules.md"},
		},
		{
			name:  "directory with a supported extension in its name is not a file",
			files: []string{"archive.md/inside.txt"},
			want:  []string{"archive.md/inside.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			createFiles(t, root, tt.files)

			got, err := listIndexableFiles(root)
			if err != nil {
				t.Fatalf("listIndexableFiles() unexpected error: %v", err)
			}

			rel := relativePaths(t, root, got)
			if !slices.Equal(rel, tt.want) {
				t.Errorf("listIndexableFiles() = %v, want %v", rel, tt.want)
			}
		})
	}
}

func TestListIndexableFilesMissingRoot(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if _, err := listIndexableFiles(missing); err == nil {
		t.Fatal("listIndexableFiles() expected an error for a missing root, got nil")
	}
}

// createFiles creates every relative path under root as an empty file,
// making parent directories as needed.
func createFiles(t *testing.T, root string, files []string) {
	t.Helper()

	for _, f := range files {
		full := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %q: %v", f, err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatalf("write %q: %v", f, err)
		}
	}
}

// relativePaths converts paths to slash-separated paths relative to root
// and sorts them, so the comparison does not depend on walk order.
func relativePaths(t *testing.T, root string, paths []string) []string {
	t.Helper()

	var out []string
	for _, p := range paths {
		r, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatalf("rel %q: %v", p, err)
		}
		out = append(out, filepath.ToSlash(r))
	}
	slices.Sort(out)
	return out
}
