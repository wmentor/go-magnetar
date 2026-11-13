package changes

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestTool(t *testing.T) *Tool {
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

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})
}

func parseResult(t *testing.T, result any) tools.Response {
	t.Helper()

	switch v := result.(type) {
	case string:
		var resp tools.Response
		if err := json.Unmarshal([]byte(v), &resp); err == nil {
			return resp
		}
		return tools.Response{
			Status: tools.StatusOK,
			Result: v,
		}
	case *tools.Response:
		return *v
	default:
		t.Fatalf("unexpected result type: %T", result)
		panic("unreachable")
	}
}

func TestLocalChangesSuccess(t *testing.T) {
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
	t.Cleanup(func() { rootFS.Close() })

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified content"), 0o644); err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)
	g := New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore})

	result, err := g.Exec(t.Context(), &Params{})
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	resp := parseResult(t, result)

	if resp.Status != tools.StatusOK {
		t.Errorf("Exec() status = %q, want %q", resp.Status, tools.StatusOK)
	}

	resultStr := resp.Result
	if !strings.Contains(resultStr, "git status") {
		t.Errorf("Exec() result should contain 'git status'")
	}

	if !strings.Contains(resultStr, "git diff") {
		t.Errorf("Exec() result should contain 'git diff'")
	}

	if !strings.Contains(resultStr, "git diff --staged") {
		t.Errorf("Exec() result should contain 'git diff --staged'")
	}
}

func TestLocalChangesNoGitRepo(t *testing.T) {
	t.Parallel()

	g := newTestTool(t)

	result, err := g.Exec(t.Context(), &Params{})
	// Expected to fail when no git repo exists
	if err == nil {
		t.Fatalf("Exec() should return error when no git repo exists, got result: %v", result)
	}

	// Check that the error message contains expected text
	if !strings.Contains(err.Error(), "git status") {
		t.Errorf("Exec() error should contain 'git status', got: %v", err)
	}
}
