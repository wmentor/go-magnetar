package agent

import (
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/agent/chat"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

// Cmd is the CLI command for running the interactive chat agent.
type Cmd struct{}

// Run executes the agent command.
// cfg is injected by kong from the binding set in cmd.Execute.
func (cmd *Cmd) Run(cfg *config.Config) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get work dir error: %w", err)
	}

	root, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open work dir error: %w", err)
	}
	defer root.Close()

	// Provide the agent's working-directory root to plugins that need
	// filesystem access (generic file tools, web preprocessor).
	plugin.SetRoot(root)

	agent, err := chat.New(cfg, root)
	if err != nil {
		return err
	}

	return agent.Run()
}
