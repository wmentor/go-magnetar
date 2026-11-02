package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func TestGitLabFileNew(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	tool := New(cfg, p, clientFactory)
	if tool == nil {
		t.Fatal("New() returned nil")
	}

	if tool.Name() != "gitlab_file" {
		t.Errorf("Name() = %s, want gitlab_file", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestGitLabFileParamsValidation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	tool := New(cfg, p, clientFactory)

	// Test with nil params
	result, err := tool.Exec(t.Context(), nil)
	if err == nil {
		t.Error("Exec() with nil params should return an error")
	}
	if result != nil {
		t.Log("Exec() returned non-nil result with nil params (expected)")
	}
}
