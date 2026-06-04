package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/wmentor/go-magnetar/internal/cmd/agent"
	"github.com/wmentor/go-magnetar/internal/cmd/indexer"
)

// CLI is the root command structure.
type CLI struct {
	Indexer indexer.Cmd `cmd:"indexer" help:"Index files into the RAG knowledge base"`
	Agent   agent.Cmd   `cmd:"agent" help:"Run the interactive chat agent"`
}

// Execute parses CLI arguments and dispatches to the appropriate subcommand.
func Execute() error {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("go-magnetar"),
		kong.Description("RAG indexer and AI chat agent"),
		kong.UsageOnError(),
	)
	return ctx.Run()
}
