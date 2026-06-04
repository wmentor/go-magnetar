package indexer

import (
	"fmt"

	"github.com/wmentor/go-magnetar/internal/agent/indexer"
	"github.com/wmentor/go-magnetar/internal/config"
)

// Cmd is the CLI command for indexing files into RAG.
type Cmd struct {
	Config    string `short:"c" help:"Path to config file" required:""`
	File      string `short:"f" help:"File to index (.md or .txt)"`
	Directory string `short:"d" help:"Directory to index (all .md and .txt files)"`
}

// Run executes the indexer command.
func (cmd *Cmd) Run() error {
	if cmd.File == "" && cmd.Directory == "" {
		return fmt.Errorf("--file or --directory is required")
	}

	cfg, err := config.Load(cmd.Config)
	if err != nil {
		return err
	}

	config.SetupLogger(cfg)

	idx, err := indexer.New(cfg)
	if err != nil {
		return err
	}

	if cmd.File != "" {
		return idx.IndexFile(cmd.File)
	}

	return idx.IndexDirectory(cmd.Directory)
}
