package systemplugin

import (
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

// stateWithTools initialises the system plugin against a fresh tool store and
// returns that store, so a test can read the tools it registered.
func stateWithTools(t *testing.T) *tools.Store {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`version: "1.0"`), 0o644))

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	maskService := mask.NewService(cfg)
	store := tools.NewStore(maskService)
	state := &plugin.State{
		Config:    cfg,
		Printer:   printer.GetDefault(),
		ToolStore: store,
	}

	p := &Plugin{}
	require.NoError(t, p.Init(state, nil))

	return store
}

// The uuid tool is one of the few that can be called in a loop without doing
// anything, so it is rate limited like the other tools that answer from the
// machine (#321).
func TestSystemUUIDToolIsRateLimited(t *testing.T) {
	t.Parallel()

	uuidTool, err := stateWithTools(t).GetTool("system_uuid")
	require.NoError(t, err)

	assert.Equal(t, 10, uuidTool.RateLimit, "system_uuid should allow ten calls per user request")
}

// The date and grep tools answer from the machine too, but they were left
// without a limit on purpose, and this keeps that decision visible.
func TestTheOtherSystemToolsHaveNoRateLimit(t *testing.T) {
	t.Parallel()

	store := stateWithTools(t)

	for _, name := range []string{"system_date", "system_grep"} {
		tool, err := store.GetTool(name)
		if err != nil {
			continue // the tool is registered only when the binary it needs is present
		}
		assert.Equal(t, 0, tool.RateLimit, "%s should have no rate limit", name)
	}
}
