package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTempDir returns a temporary directory that is removed when the test ends.
func newTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

// embeddedFiles returns all non-hidden embedded template files as paths relative to "templates".
func embeddedFiles(t *testing.T) []string {
	t.Helper()

	var files []string

	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || isHidden(path) {
			return nil
		}
		files = append(files, strings.TrimPrefix(path, "templates/"))

		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded templates: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected embedded templates, found none")
	}

	return files
}

// assertFileExists fails the test if the file does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected %s to be a file", path)
	}
}

// assertDirExists fails the test if the directory does not exist.
func assertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", path)
	}
}

func TestIsHidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"regular file", "templates/config.yml", false},
		{"nested regular file", "templates/plugins/github.yml", false},
		{"hidden file", "templates/.hidden", true},
		{"hidden directory", "templates/.git/config", true},
		{"hidden at start", ".hidden", true},
		{"hidden nested file", "templates/plugins/.secret.yml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isHidden(tt.path); got != tt.want {
				t.Errorf("isHidden(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestEnsureConfigDir(t *testing.T) {
	t.Parallel()

	t.Run("creates new directory with templates and store", func(t *testing.T) {
		t.Parallel()

		configDir := filepath.Join(newTempDir(t), ".go-magnetar")

		if err := EnsureConfigDir(configDir); err != nil {
			t.Fatalf("EnsureConfigDir() error = %v", err)
		}

		assertDirExists(t, configDir)
		assertDirExists(t, filepath.Join(configDir, "store"))
		for _, f := range embeddedFiles(t) {
			assertFileExists(t, filepath.Join(configDir, f))
		}
	})

	t.Run("existing directory gets store and missing templates", func(t *testing.T) {
		t.Parallel()

		configDir := newTempDir(t)

		if err := EnsureConfigDir(configDir); err != nil {
			t.Fatalf("EnsureConfigDir() error = %v", err)
		}

		assertDirExists(t, filepath.Join(configDir, "store"))
		for _, f := range embeddedFiles(t) {
			assertFileExists(t, filepath.Join(configDir, f))
		}
	})

	t.Run("is idempotent and preserves user changes", func(t *testing.T) {
		t.Parallel()

		configDir := filepath.Join(newTempDir(t), "cfg")

		if err := EnsureConfigDir(configDir); err != nil {
			t.Fatalf("first EnsureConfigDir() error = %v", err)
		}

		custom := filepath.Join(configDir, "config.yml")
		if err := os.WriteFile(custom, []byte("custom: true\n"), 0o644); err != nil {
			t.Fatalf("write custom file: %v", err)
		}

		if err := EnsureConfigDir(configDir); err != nil {
			t.Fatalf("second EnsureConfigDir() error = %v", err)
		}

		got, err := os.ReadFile(custom)
		if err != nil {
			t.Fatalf("read custom file: %v", err)
		}
		if string(got) != "custom: true\n" {
			t.Errorf("user file overwritten, got %q", got)
		}
	})

	t.Run("error when config path is a file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(newTempDir(t), "file")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		if err := EnsureConfigDir(path); err == nil {
			t.Error("expected error when config path is a regular file")
		}
	})

	t.Run("error when parent is a file", func(t *testing.T) {
		t.Parallel()

		parent := filepath.Join(newTempDir(t), "parent")
		if err := os.WriteFile(parent, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		if err := EnsureConfigDir(filepath.Join(parent, "cfg")); err == nil {
			t.Error("expected error when parent path is a regular file")
		}
	})
}

func TestSyncTemplates(t *testing.T) {
	t.Parallel()

	t.Run("adds missing templates", func(t *testing.T) {
		t.Parallel()

		configDir := newTempDir(t)

		if err := SyncTemplates(configDir); err != nil {
			t.Fatalf("SyncTemplates() error = %v", err)
		}

		for _, f := range embeddedFiles(t) {
			assertFileExists(t, filepath.Join(configDir, f))
		}
	})

	t.Run("does not overwrite existing files", func(t *testing.T) {
		t.Parallel()

		configDir := newTempDir(t)
		existing := filepath.Join(configDir, "config.yml")
		if err := os.WriteFile(existing, []byte("keep: me\n"), 0o644); err != nil {
			t.Fatalf("write existing: %v", err)
		}

		if err := SyncTemplates(configDir); err != nil {
			t.Fatalf("SyncTemplates() error = %v", err)
		}

		got, err := os.ReadFile(existing)
		if err != nil {
			t.Fatalf("read existing: %v", err)
		}
		if string(got) != "keep: me\n" {
			t.Errorf("existing file overwritten, got %q", got)
		}
	})

	t.Run("restores a deleted template", func(t *testing.T) {
		t.Parallel()

		configDir := newTempDir(t)
		if err := SyncTemplates(configDir); err != nil {
			t.Fatalf("SyncTemplates() error = %v", err)
		}

		files := embeddedFiles(t)
		removed := filepath.Join(configDir, files[0])
		if err := os.Remove(removed); err != nil {
			t.Fatalf("remove: %v", err)
		}

		if err := SyncTemplates(configDir); err != nil {
			t.Fatalf("SyncTemplates() error = %v", err)
		}
		assertFileExists(t, removed)
	})

	t.Run("error when base is a file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(newTempDir(t), "file")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		if err := SyncTemplates(path); err == nil {
			t.Error("expected error when base directory is a regular file")
		}
	})
}

func TestCopyDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{"valid prefix copies nested files", "templates", false},
		{"subdirectory prefix", "templates/plugins", false},
		{"missing prefix", "does-not-exist", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseDir := newTempDir(t)

			err := copyDir(baseDir, tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Fatalf("copyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Compare against the embedded files under the prefix.
			count := 0
			walkErr := fs.WalkDir(templatesFS, tt.prefix, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || isHidden(path) {
					return nil
				}
				count++
				rel := strings.TrimPrefix(path, tt.prefix+"/")
				assertFileExists(t, filepath.Join(baseDir, rel))

				return nil
			})
			if walkErr != nil {
				t.Fatalf("walk: %v", walkErr)
			}
			if count == 0 {
				t.Fatal("expected at least one file to be copied")
			}
		})
	}

	t.Run("does not copy hidden files", func(t *testing.T) {
		t.Parallel()

		baseDir := newTempDir(t)
		if err := copyDir(baseDir, "templates"); err != nil {
			t.Fatalf("copyDir() error = %v", err)
		}

		err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != baseDir && strings.HasPrefix(d.Name(), ".") {
				t.Errorf("unexpected hidden entry copied: %s", path)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	})
}

func TestExtractFile(t *testing.T) {
	t.Parallel()

	t.Run("extracts file content", func(t *testing.T) {
		t.Parallel()

		baseDir := newTempDir(t)

		if err := extractFile(baseDir, "templates/config.yml", "config.yml"); err != nil {
			t.Fatalf("extractFile() error = %v", err)
		}

		want, err := templatesFS.ReadFile("templates/config.yml")
		if err != nil {
			t.Fatalf("read embedded: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(baseDir, "config.yml"))
		if err != nil {
			t.Fatalf("read extracted: %v", err)
		}
		if string(got) != string(want) {
			t.Error("extracted content differs from embedded content")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		baseDir := newTempDir(t)

		if err := extractFile(baseDir, "templates/plugins/github.yml", "a/b/github.yml"); err != nil {
			t.Fatalf("extractFile() error = %v", err)
		}
		assertFileExists(t, filepath.Join(baseDir, "a", "b", "github.yml"))
	})

	t.Run("error on missing embedded file", func(t *testing.T) {
		t.Parallel()

		baseDir := newTempDir(t)

		if err := extractFile(baseDir, "templates/nope.yml", "nope.yml"); err == nil {
			t.Error("expected error for missing embedded file")
		}
		if _, err := os.Stat(filepath.Join(baseDir, "nope.yml")); !os.IsNotExist(err) {
			t.Error("no file should be created on error")
		}
	})

	t.Run("error when parent is a file", func(t *testing.T) {
		t.Parallel()

		baseDir := newTempDir(t)
		blocker := filepath.Join(baseDir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatalf("write blocker: %v", err)
		}

		if err := extractFile(baseDir, "templates/config.yml", "blocker/config.yml"); err == nil {
			t.Error("expected error when parent path is a file")
		}
	})
}
