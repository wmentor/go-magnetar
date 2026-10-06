package searchplugin

import (
	"context"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/search"
)

func init() {
	plugin.Register("search", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	tools, err := search.New(s.Config, s)
	if err != nil {
		return err
	}

	hub.RegisterTool(plugin.LLMTool{
		Definition:   search.StaticDefinition,
		IsSearchTool: true,
		Execute: func(_ context.Context, args string) (string, error) {
			return tools.Dispatch("search", args), nil
		},
	})

	return nil
}
