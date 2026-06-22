package agentcli

import (
	"github.com/wmentor/go-magnetar/internal/cmd/agent"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("cli.agent", &Plugin{})
}

// wrapper is the kong-annotated struct that exposes agent.Cmd as a subcommand.
type wrapper struct {
	Agent agent.Cmd `cmd:"" help:"Run the interactive chat agent"`
}

// Plugin registers the "agent" CLI subcommand.
type Plugin struct {
	w wrapper
}

// RegisterCLI is called before kong.Parse to make the subcommand available.
func (p *Plugin) RegisterCLI(add func(cmd any)) {
	add(&p.w)
}

func (p *Plugin) Init(_ plugin.State, _ plugin.Hub) error { return nil }
