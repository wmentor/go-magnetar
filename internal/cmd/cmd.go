package cmd

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/wmentor/go-magnetar/internal/agent/chat"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

type Globals struct {
	Config string `short:"c" type:"path" default:"~/.go-magnetar.yaml" help:"Path to config file" env:"GO_MAGNETAR_CONFIG"`
}

type cli struct {
	Globals
}

func Execute() error {
	root := &cli{}
	kong.Parse(root,
		kong.Name("go-magnetar"),
		kong.Description("RAG indexer and AI chat agent"),
		kong.UsageOnError(),
		kong.Bind(&root.Globals),
	)

	cfg, err := config.Load(root.Globals.Config)
	if err != nil {
		return err
	}

	config.SetupLogger(cfg)

	if err := plugin.InitAll(&plugin.State{Config: cfg}); err != nil {
		return err
	}
	defer plugin.Stop()

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get work dir error: %w", err)
	}

	rootFS, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open work dir error: %w", err)
	}
	defer rootFS.Close()

	// Provide the agent's working-directory root to plugins that need
	// filesystem access (generic file tools, web preprocessor).
	plugin.SetRoot(rootFS)

	agent, err := chat.New(cfg, rootFS)
	if err != nil {
		return err
	}

	return agent.Run()
}
