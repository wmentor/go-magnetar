package agent

import (
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/agent/chat"
	"github.com/wmentor/go-magnetar/internal/config"
)

// Cmd is the CLI command for running the interactive chat agent.
type Cmd struct {
	Config string `short:"c" type:"path" default:"~/.go-magnetar.yaml" help:"Path to config file"`
}

// Run executes the agent command.
func (cmd *Cmd) Run() error {
	cfg, err := config.Load(cmd.Config)
	if err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get work dir error: %w", err)
	}

	root, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open work dir error: %w", err)
	}
	defer root.Close()

	config.SetupLogger(cfg)

	agent, err := chat.New(cfg, root)
	if err != nil {
		return err
	}

	return agent.Run()
}
