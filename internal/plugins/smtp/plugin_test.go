package smtpplugin

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

func loadConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return cfg
}

func TestInitRegistersSmtpToolWhenEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		expectTool  bool
		errExpected bool
	}{
		{
			name:       "not configured (disabled by default)",
			yaml:       `version: "1.0"`,
			expectTool: false,
		},
		{
			name: "explicitly disabled",
			yaml: `version: "1.0"
smtp:
  enable: false
`,
			expectTool: false,
		},
		{
			name: "enabled",
			yaml: `version: "1.0"
smtp:
  enable: true
  host: smtp.example.com
  port: 587
  username: user@example.com
  password: secret
  encryption: starttls
  from_name: Test User
  from_email: user@example.com
`,
			expectTool: true,
		},
		{
			name: "enabled with minimal config",
			yaml: `version: "1.0"
smtp:
  enable: true
`,
			expectTool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadConfig(t, tt.yaml)
			maskService := mask.NewService(cfg)
			root, err := os.OpenRoot(t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { root.Close() })

			state := &plugin.State{
				Config:    cfg,
				ToolStore: tools.NewStore(maskService),
				Printer:   printer.GetDefault(),
				Root:      root,
			}
			err = (&Plugin{}).Init(state, nil)
			if tt.errExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			toolNames := make([]string, 0)
			for _, tool := range state.ToolStore.Tools() {
				toolNames = append(toolNames, tool.Name)
			}

			if tt.expectTool {
				assert.Contains(t, toolNames, "smtp")
			} else {
				assert.NotContains(t, toolNames, "smtp")
			}
		})
	}
}

func TestInitToolDescription(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, `version: "1.0"
smtp:
  enable: true
  host: smtp.example.com
  port: 587
  username: user@example.com
  password: secret
  encryption: starttls
  from_name: Test User
  from_email: user@example.com
`)
	maskService := mask.NewService(cfg)
	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })

	state := &plugin.State{
		Config:    cfg,
		ToolStore: tools.NewStore(maskService),
		Printer:   printer.GetDefault(),
		Root:      root,
	}
	err = (&Plugin{}).Init(state, nil)
	require.NoError(t, err)

	// Check smtp tool
	smtpTool, err := state.ToolStore.GetTool("smtp")
	require.NoError(t, err)
	smtpDef := smtpTool.Definition()
	assert.Equal(t, "smtp", smtpDef.Function.Name)
	assert.Contains(t, smtpDef.Function.Description, "Send an email via SMTP")
	assert.Contains(t, smtpDef.Function.Description, "Markdown")
}

func TestInitToolParametersExist(t *testing.T) {
	t.Parallel()

	cfg := loadConfig(t, `version: "1.0"
smtp:
  enable: true
  host: smtp.example.com
  port: 587
  username: user@example.com
  password: secret
  encryption: starttls
  from_name: Test User
  from_email: user@example.com
`)
	maskService := mask.NewService(cfg)
	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })

	state := &plugin.State{
		Config:    cfg,
		ToolStore: tools.NewStore(maskService),
		Printer:   printer.GetDefault(),
		Root:      root,
	}
	err = (&Plugin{}).Init(state, nil)
	require.NoError(t, err)

	// Check smtp tool parameters exist
	smtpTool, err := state.ToolStore.GetTool("smtp")
	require.NoError(t, err)
	smtpDef := smtpTool.Definition()
	require.NotNil(t, smtpDef.Function.Parameters)
}

func TestInitToolCallCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedTools int
	}{
		{
			name:          "disabled - no tools",
			yaml:          `version: "1.0"`,
			expectedTools: 0,
		},
		{
			name: "enabled - one tool",
			yaml: `version: "1.0"
smtp:
  enable: true
`,
			expectedTools: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadConfig(t, tt.yaml)
			maskService := mask.NewService(cfg)
			root, err := os.OpenRoot(t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { root.Close() })

			state := &plugin.State{
				Config:    cfg,
				ToolStore: tools.NewStore(maskService),
				Printer:   printer.GetDefault(),
				Root:      root,
			}
			err = (&Plugin{}).Init(state, nil)
			require.NoError(t, err)

			toolsList := state.ToolStore.Tools()
			assert.Len(t, toolsList, tt.expectedTools)

			if tt.expectedTools > 0 {
				_, err := state.ToolStore.GetTool("smtp")
				assert.NoError(t, err)
			} else {
				_, err := state.ToolStore.GetTool("smtp")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "tool not found")
			}
		})
	}
}
