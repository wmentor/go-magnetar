package archiveplugin

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/mask"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func newTestPluginState(t *testing.T) *plugin.State {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	cfgContent := "version: \"1.0\"\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0o600))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	maskService := mask.NewService(cfg)
	return &plugin.State{
		Config:    cfg,
		Printer:   printer.GetDefault(),
		ToolStore: tools.NewStore(maskService),
	}
}

func TestPluginInit(t *testing.T) {
	t.Parallel()

	state := newTestPluginState(t)
	p := &Plugin{}

	require.NoError(t, p.Init(state, nil))
	assert.Equal(t, state, p.state)
	require.NotNil(t, p.listTool)
	require.NotNil(t, p.readTool)

	registered := state.ToolStore.Tools()
	require.Len(t, registered, 2)

	for _, toolName := range []string{"archive_list", "archive_read"} {
		tool, err := state.ToolStore.GetTool(toolName)
		require.NoError(t, err)
		assert.Equal(t, toolName, tool.Name)
		assert.NotNil(t, tool.Definition)
		assert.NotEmpty(t, tool.Definition().Function.Description)
		assert.NotNil(t, tool.Execute)
	}
}

func TestPluginInitAll(t *testing.T) {
	state := newTestPluginState(t)

	require.NoError(t, plugin.InitAll(state))

	for _, toolName := range []string{"archive_list", "archive_read"} {
		tool, err := state.ToolStore.GetTool(toolName)
		require.NoError(t, err)
		assert.Equal(t, toolName, tool.Name)
	}
}

func TestPluginToolsExecuteOnZip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "sample.zip")
	f, err := os.Create(zipPath)
	require.NoError(t, err)

	zw := zip.NewWriter(f)
	w, err := zw.Create("hello.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("hello archive"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	state := newTestPluginState(t)
	p := &Plugin{}
	require.NoError(t, p.Init(state, nil))

	listTool, err := state.ToolStore.GetTool("archive_list")
	require.NoError(t, err)
	listRes := listTool.Execute(context.Background(), []byte(`{"filename":"`+filepath.ToSlash(zipPath)+`"}`))
	assert.Contains(t, listRes, "hello.txt")

	readTool, err := state.ToolStore.GetTool("archive_read")
	require.NoError(t, err)
	readRes := readTool.Execute(context.Background(), []byte(`{"archive_path":"`+filepath.ToSlash(zipPath)+`","file_path":"hello.txt"}`))
	assert.Contains(t, readRes, "hello archive")
}
