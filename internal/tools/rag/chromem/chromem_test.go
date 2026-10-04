package chromem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/rag/chromem"
)

func TestParseDataDir(t *testing.T) {
	t.Parallel()

	t.Run("empty data dir", func(t *testing.T) {
		t.Parallel()
		_, err := chromem.ParseDataDir("")
		if err == nil {
			t.Error("ParseDataDir() should return error for empty data dir")
		}
	})

	t.Run("with tilde", func(t *testing.T) {
		t.Parallel()
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home dir: %v", err)
		}

		result, err := chromem.ParseDataDir("~/.go-magnetar/store")
		if err != nil {
			t.Errorf("ParseDataDir() error = %v", err)
		}
		if result != filepath.Join(homeDir, ".go-magnetar/store") {
			t.Errorf("ParseDataDir() = %q, want %q", result, filepath.Join(homeDir, ".go-magnetar/store"))
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		t.Parallel()
		result, err := chromem.ParseDataDir("/tmp/chromem-data")
		if err != nil {
			t.Errorf("ParseDataDir() error = %v", err)
		}
		if result != "/tmp/chromem-data" {
			t.Errorf("ParseDataDir() = %q, want %q", result, "/tmp/chromem-data")
		}
	})
}

func TestPreview(t *testing.T) {
	t.Parallel()

	t.Run("short string", func(t *testing.T) {
		t.Parallel()
		result := chromem.Preview("hello", 10)
		if result != "hello" {
			t.Errorf("Preview() = %q, want %q", result, "hello")
		}
	})

	t.Run("exact length", func(t *testing.T) {
		t.Parallel()
		result := chromem.Preview("hello", 5)
		if result != "hello" {
			t.Errorf("Preview() = %q, want %q", result, "hello")
		}
	})

	t.Run("truncated", func(t *testing.T) {
		t.Parallel()
		result := chromem.Preview("hello world", 5)
		if result != "hello…" {
			t.Errorf("Preview() = %q, want %q", result, "hello…")
		}
	})

	t.Run("unicode string", func(t *testing.T) {
		t.Parallel()
		result := chromem.Preview("Привет мир", 6)
		expected := "Привет…"
		if result != expected {
			t.Errorf("Preview() = %q, want %q", result, expected)
		}
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Parallel()

	t.Run("parallel vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 0}
		b := []float32{1, 0}
		result := chromem.CosineSimilarity(a, b)
		if result != 1.0 {
			t.Errorf("CosineSimilarity() = %v, want 1.0", result)
		}
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 0}
		b := []float32{0, 1}
		result := chromem.CosineSimilarity(a, b)
		if result != 0 {
			t.Errorf("CosineSimilarity() = %v, want 0", result)
		}
	})

	t.Run("opposite vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 0}
		b := []float32{-1, 0}
		result := chromem.CosineSimilarity(a, b)
		if result != -1 {
			t.Errorf("CosineSimilarity() = %v, want -1", result)
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 0}
		b := []float32{1}
		result := chromem.CosineSimilarity(a, b)
		if result != 0 {
			t.Errorf("CosineSimilarity() = %v, want 0", result)
		}
	})

	t.Run("empty vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{}
		b := []float32{}
		result := chromem.CosineSimilarity(a, b)
		if result != 0 {
			t.Errorf("CosineSimilarity() = %v, want 0", result)
		}
	})

	t.Run("zero vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{0, 0}
		b := []float32{0, 0}
		result := chromem.CosineSimilarity(a, b)
		if result != 0 {
			t.Errorf("CosineSimilarity() = %v, want 0", result)
		}
	})

	t.Run("3D vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 1, 1}
		b := []float32{1, 1, 1}
		result := chromem.CosineSimilarity(a, b)
		expected := float32(1.0)
		if result != expected {
			t.Errorf("CosineSimilarity() = %v, want %v", result, expected)
		}
	})

	t.Run("unit vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1, 0}
		b := []float32{0.70710677, 0.70710677}
		result := chromem.CosineSimilarity(a, b)
		expected := float32(0.70710677)
		if result != expected {
			t.Errorf("CosineSimilarity() = %v, want %v", result, expected)
		}
	})
}
