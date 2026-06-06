package agent

import (
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

	config.SetupLogger(cfg)

	agent, err := chat.New(cfg)
	if err != nil {
		return err
	}

	return agent.Run()
}
