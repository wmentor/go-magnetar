package rag_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

func TestContentUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty string",
			content: "",
		},
		{
			name:    "simple text",
			content: "hello world",
		},
		{
			name:    "text with newlines",
			content: "line1\nline2\nline3",
		},
		{
			name:    "text with spaces",
			content: "hello   world   test",
		},
		{
			name:    "unicode text",
			content: "Привет мир",
		},
		{
			name:    "same content produces same UUID",
			content: "repeat this text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uuid := rag.ContentUUID(tt.content)
			if uuid == "" {
				t.Error("ContentUUID() returned empty string")
			}

			if uuid != rag.ContentUUID(tt.content) {
				t.Error("ContentUUID() is not deterministic")
			}
		})
	}
}

func TestNewEmbeddingClient(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		client := rag.NewEmbeddingClient(nil)
		if client == nil {
			t.Fatal("NewEmbeddingClient(nil) returned nil")
		}
	})

	t.Run("configured client", func(t *testing.T) {
		t.Parallel()

		cfgPath := filepath.Join(t.TempDir(), "config.yml")
		yaml := "version: \"1.0\"\nrag:\n  llm:\n    api_key: \"test-key\"\n    base_url: \"http://localhost:11434/v1\"\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			t.Fatal(err)
		}

		client := rag.NewEmbeddingClient(cfg)
		if client == nil {
			t.Fatal("NewEmbeddingClient(cfg) returned nil")
		}
	})
}
