package initplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

type mockHub struct {
	RegisteredCommands []plugin.ChatCommand
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.RegisteredCommands = append(m.RegisteredCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(_ plugin.PreprocessorFunc) {}
func (m *mockHub) Go(_ func(ctx context.Context))                 {}
func (m *mockHub) Stop()                                          {}

func TestPluginInitRegistration(t *testing.T) {
	p := &Plugin{}
	hub := &mockHub{}

	err := p.Init(nil, hub)
	require.NoError(t, err)
	require.Len(t, hub.RegisteredCommands, 1)

	cmd := hub.RegisteredCommands[0]
	assert.Equal(t, "init", cmd.Name)
	assert.NotEmpty(t, cmd.Help)
}

func TestExecuteWhenAgentsMdExists(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agents"), 0o644)
	require.NoError(t, err)

	p := &Plugin{}
	hub := &mockHub{}
	err = p.Init(nil, hub)
	require.NoError(t, err)

	cmd := hub.RegisteredCommands[0]
	input := &plugin.ChatCommandInput{
		Args: tmpDir,
	}

	resp, err := cmd.Execute(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Contains(t, resp.Stdout, "Project is already initialized: AGENTS.md already exists.")
	assert.Empty(t, resp.Content)
	assert.False(t, resp.Exit)
}

func TestExecuteWhenClaudeMdExists(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("# Claude"), 0o644)
	require.NoError(t, err)

	p := &Plugin{}
	hub := &mockHub{}
	err = p.Init(nil, hub)
	require.NoError(t, err)

	cmd := hub.RegisteredCommands[0]
	input := &plugin.ChatCommandInput{
		Args: tmpDir,
	}

	resp, err := cmd.Execute(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Contains(t, resp.Stdout, "Project is already initialized: CLAUDE.md already exists.")
	assert.Empty(t, resp.Content)
	assert.False(t, resp.Exit)
}

func TestExecuteGeneratesInitContent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("all:\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Example\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "cmd"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0o755))

	p := &Plugin{}
	hub := &mockHub{}
	err := p.Init(nil, hub)
	require.NoError(t, err)

	cmd := hub.RegisteredCommands[0]
	input := &plugin.ChatCommandInput{
		Args: tmpDir,
	}

	resp, err := cmd.Execute(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Contains(t, resp.Stdout, "Initializing project")
	assert.Contains(t, resp.Content, "AGENTS.md")
	assert.Contains(t, resp.Content, "go.mod")
	assert.Contains(t, resp.Content, "Makefile")
	assert.Contains(t, resp.Content, "README.md")
	assert.Contains(t, resp.Content, "cmd/")
	assert.NotContains(t, resp.Content, ".git")
	assert.False(t, resp.Exit)
}

func TestExecuteNonExistentDirectory(t *testing.T) {
	p := &Plugin{}
	hub := &mockHub{}
	err := p.Init(nil, hub)
	require.NoError(t, err)

	cmd := hub.RegisteredCommands[0]
	input := &plugin.ChatCommandInput{
		Args: filepath.Join(t.TempDir(), "nonexistent-dir"),
	}

	resp, err := cmd.Execute(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Contains(t, resp.Stdout, "Failed to scan project directory")
	assert.Empty(t, resp.Content)
}

func TestScanProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files and directories
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "random.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "src"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".venv"), 0o755))

	topLevel, keyFiles, err := scanProject(tmpDir)
	require.NoError(t, err)

	assert.Contains(t, topLevel, "src/")
	assert.Contains(t, topLevel, "package.json")
	assert.Contains(t, topLevel, "Dockerfile")
	assert.Contains(t, topLevel, "random.txt")
	assert.NotContains(t, topLevel, "node_modules/")
	assert.NotContains(t, topLevel, ".venv/")

	assert.Equal(t, []string{"Dockerfile", "package.json"}, keyFiles)
}
