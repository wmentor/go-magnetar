package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/agent/chat"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	version "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/version"
	"github.com/wmentor/go-magnetar/internal/printer"
)

type Globals struct {
	Config string `short:"c" type:"path" default:"~/.go-magnetar.yaml" help:"Path to config file" env:"GO_MAGNETAR_CONFIG"`
	File   string `short:"f" type:"path" help:"Input file"`
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

	printer.SetDefault(printer.New(false))

	if root.Globals.File == "" {
		version.PrintVersion(cfg.String("llm.model"))
		printEnabledModules(cfg)
	}

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

	printer.Verbose(cfg.Bool("verbose") && root.Globals.File == "")

	agent, err := chat.New(cfg, rootFS)
	if err != nil {
		return err
	}

	if root.Globals.File != "" {
		data, err := os.ReadFile(root.Globals.File)
		if err != nil {
			return errors.Wrapf(err, "read file %s error", root.Globals.File)
		}

		txt := string(data)

		for _, preprocessor := range plugin.Preprocessors() {
			if str, err := preprocessor(context.Background(), txt); err == nil {
				txt = str
			} else {
				printer.Error("chat: preprocessor error", "err", err)
			}
		}

		answer, err := agent.Ask(txt)
		if err != nil {
			return errors.Wrap(err, "agent error")
		}

		fmt.Println(answer)

		return nil
	}

	return agent.Run()
}

func printEnabledModules(cfg *config.Config) {
	has := false
	if cfg.String("confluence.base_url") != "" && !cfg.Bool("confluence.disable") {
		printer.Print(printer.IconModule, "confluence plugin is enabled")
		has = true
	}
	if cfg.String("gitlab.base_url") != "" && !cfg.Bool("gitlab.disable") {
		printer.Print(printer.IconModule, "gitlab plugin is enabled")
		has = true
	}
	if cfg.String("github.base_url") != "" && !cfg.Bool("github.disable") {
		printer.Print(printer.IconModule, "github plugin is enabled")
		has = true
	}
	if cfg.String("jira.base_url") != "" && !cfg.Bool("jira.disable") {
		printer.Print(printer.IconModule, "jira plugin is enabled")
		has = true
	}
	if cfg.String("rag.llm.base_url") != "" && !cfg.Bool("rag.disable") {
		printer.Print(printer.IconModule, "rag plugin is enabled")
		has = true
	}
	if has {
		printer.EmptyLine()
	}
}
