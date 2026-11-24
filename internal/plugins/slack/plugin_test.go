package slackplugin

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

func stateFor(t *testing.T, yaml string) *plugin.State {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return &plugin.State{
		Config:    cfg,
		ToolStore: tools.NewStore(mask.NewService(cfg)),
		Printer:   printer.GetDefault(),
	}
}

func TestInitRegistersSlackOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		yaml   string
		expect bool
	}{
		{name: "not configured", yaml: `version: "1.0"`},
		{name: "disabled", yaml: "version: \"1.0\"\nslack:\n  enable: false\n  token: xoxb-test\n"},
		{name: "enabled", yaml: "version: \"1.0\"\nslack:\n  enable: true\n  token: xoxb-test\n  default_channel_id: C1\n", expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := stateFor(t, tt.yaml)
			require.NoError(t, (&Plugin{}).Init(state, nil))

			names := make([]string, 0, len(state.ToolStore.Tools()))
			for _, tool := range state.ToolStore.Tools() {
				names = append(names, tool.Name)
			}
			if tt.expect {
				assert.Contains(t, names, "slack")
			} else {
				assert.NotContains(t, names, "slack")
			}
		})
	}
}

func TestRegisteredToolDescribesItsConfig(t *testing.T) {
	t.Parallel()

	state := stateFor(t, "version: \"1.0\"\nslack:\n  enable: true\n  token: xoxb-test\n")
	require.NoError(t, (&Plugin{}).Init(state, nil))

	tool, err := state.ToolStore.GetTool("slack")
	require.NoError(t, err)
	description := tool.Definition().Function.Description
	assert.Contains(t, description, "slack.enable=true")
	assert.Contains(t, description, "slack.token")
}
