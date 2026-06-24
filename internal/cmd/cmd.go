package cmd

import (
	"github.com/alecthomas/kong"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

// Globals holds flags shared across all subcommands.
// It is embedded in the root CLI struct and bound into every Run() call.
type Globals struct {
	Config string `short:"c" type:"path" default:"~/.go-magnetar.yaml" help:"Path to config file" env:"GO_MAGNETAR_CONFIG"`
}

// cli is the root command structure. Subcommands are contributed dynamically
// by CLI plugins via kong.Plugins.
type cli struct {
	Globals
	kong.Plugins
}

// Execute parses CLI arguments, loads config, initialises plugins and
// dispatches to the appropriate subcommand.
//
// Flow:
//  1. kong.Parse — resolves flags including -c/--config
//  2. config.Load — loads the YAML config
//  3. plugin.InitAll — initialises all plugins with the real State
//  4. ctx.Run — dispatches to the selected subcommand
func Execute() error {
	root := &cli{}
	root.Plugins = plugin.KongPlugins()
	ctx := kong.Parse(root,
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

	return ctx.Run(&root.Globals, cfg)
}
