package indexercli

import (
	"github.com/wmentor/go-magnetar/internal/cmd/indexer"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("cli.indexer", &Plugin{})
}

// wrapper is the kong-annotated struct that exposes indexer.Cmd as a subcommand.
type wrapper struct {
	Indexer indexer.Cmd `cmd:"" help:"Index files into the RAG knowledge base"`
}

// Plugin registers the "indexer" CLI subcommand.
type Plugin struct {
	w wrapper
}

// RegisterCLI is called before kong.Parse to make the subcommand available.
func (p *Plugin) RegisterCLI(add func(cmd any)) {
	add(&p.w)
}

func (p *Plugin) Init(_ plugin.State, _ plugin.Hub) error { return nil }
