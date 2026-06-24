package genericplugin

import (
	"context"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
)

func init() {
	plugin.Register("generic", &Plugin{})
}

// Plugin wraps the generic file tools and exposes file_read, file_list,
// file_write, file_exists as LLM tools.
// The GenericTools instance is created lazily on first use so that the
// agent's working-directory Root (set via plugin.SetRoot) is available.
type Plugin struct {
	mu    sync.Mutex
	state *plugin.State
	tools *generic.GenericTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	hub.RegisterTool(plugin.LLMTool{
		Definition: generic.StaticDefinitionFileRead,
		Execute: func(_ context.Context, args string) (string, error) {
			return p.get().Dispatch("file_read", args), nil
		},
	})
	hub.RegisterTool(plugin.LLMTool{
		Definition: generic.StaticDefinitionFileList,
		Execute: func(_ context.Context, args string) (string, error) {
			return p.get().Dispatch("file_list", args), nil
		},
	})
	hub.RegisterTool(plugin.LLMTool{
		Definition: generic.StaticDefinitionFileWrite,
		Execute: func(_ context.Context, args string) (string, error) {
			return p.get().Dispatch("file_write", args), nil
		},
	})
	hub.RegisterTool(plugin.LLMTool{
		Definition: generic.StaticDefinitionFileExists,
		Execute: func(_ context.Context, args string) (string, error) {
			return p.get().Dispatch("file_exists", args), nil
		},
	})
	return nil
}

func (p *Plugin) get() *generic.GenericTools {
	p.mu.Lock()
	defer p.mu.Unlock()
	root := plugin.GetRoot()
	if p.tools == nil || p.tools.Root() != root {
		p.tools = generic.New(p.state.Config, root, p.state)
	}
	return p.tools
}
