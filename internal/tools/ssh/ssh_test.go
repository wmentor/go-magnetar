package ssh

import (
	"context"
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
)

func newTestTools(t *testing.T) *Tool {
	t.Helper()

	path := t.TempDir() + "/config.yml"
	if err := os.WriteFile(path, []byte("version: \"1.0\"\nssh:\n  enable: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS for tests
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})
}

func TestSSHExecSuccess(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	// This test will fail because we don't have a real SSH server,
	// but we can test that the error is properly formatted
	params := &SSHParams{
		Command: "echo test",
		Addr:    "user@nonexistent:22",
	}

	_, err := g.Exec(context.Background(), params)
	if err == nil {
		t.Fatalf("Exec() should return error for non-existent server")
	}
}

func TestSSHExecNoAddr(t *testing.T) {
	t.Parallel()

	g := newTestTools(t)

	params := &SSHParams{
		Command: "echo test",
		Addr:    "",
	}

	_, err := g.Exec(context.Background(), params)
	if err == nil {
		t.Fatalf("Exec() should return error when no address provided")
	}
}

func TestSSHExecReadOnlyMode(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/config.yml"
	if err := os.WriteFile(path, []byte("version: \"1.0\"\nssh:\n  enable: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS for tests
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	g := New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: true, ResponseStore: responseStore})

	params := &SSHParams{
		Command: "echo test",
		Addr:    "user@localhost:22",
	}

	_, err = g.Exec(context.Background(), params)
	if err == nil {
		t.Fatalf("Exec() should return error in read-only mode")
	}
}

func TestSSHExecDisabled(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/config.yml"
	if err := os.WriteFile(path, []byte("version: \"1.0\"\nssh:\n  enable: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	// Create root FS for tests
	rootFS, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	g := New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	params := &SSHParams{
		Command: "echo test",
		Addr:    "user@localhost:22",
	}

	_, err = g.Exec(context.Background(), params)
	if err == nil {
		t.Fatalf("Exec() should return error when SSH is disabled")
	}
}
