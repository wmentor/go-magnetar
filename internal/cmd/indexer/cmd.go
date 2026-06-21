package indexer

import (
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/agent/indexer"
	"github.com/wmentor/go-magnetar/internal/config"
)

// Cmd is the CLI command for indexing files into RAG.
type Cmd struct {
	Config  string `short:"c" type:"path" default:"~/.go-magnetar.yaml" help:"Path to config file"`
	File    string `short:"f" help:"File to index (.md or .txt)"`
	URL     string `help:"URL to index"`
	Message string `short:"m" help:"Message to prepend to each chunk for improved search"`
}

// Run executes the indexer command.
func (cmd *Cmd) Run() error {
	if cmd.File == "" && cmd.URL == "" {
		return fmt.Errorf("--file or --url is required")
	}

	root, err := os.OpenRoot("/")
	if err != nil {
		return fmt.Errorf("open root error: %w", err)
	}
	defer root.Close()

	cfg, err := config.Load(cmd.Config)
	if err != nil {
		return err
	}

	config.SetupLogger(cfg)

	idx, err := indexer.New(cfg, root)
	if err != nil {
		return err
	}

	if cmd.File != "" {
		return idx.IndexFile(cmd.File, cmd.Message)
	}

	return idx.IndexURL(cmd.URL, cmd.Message)
}
