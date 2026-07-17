package genericplugin

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/generic"
)

func init() {
	plugin.Register("generic", &Plugin{})
}

// Plugin wraps the generic file tools and exposes file_read, file_list,
// file_write, file_exists, and system_grep (if grep is available) as LLM tools.
// The GenericTools instance is created lazily on first use so that the
// agent's working-directory Root (set via plugin.SetRoot) is available.
type Plugin struct {
	mu             sync.Mutex
	state          *plugin.State
	tools          *generic.GenericTools
	grepOnce       sync.Once
	preprocessOnce sync.Once
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
	if _, err := exec.LookPath("grep"); err == nil {
		hub.RegisterTool(plugin.LLMTool{
			Definition: generic.StaticDefinitionSystemGrep,
			Execute: func(_ context.Context, args string) (string, error) {
				return p.get().Dispatch("system_grep", args), nil
			},
		})
	}
	hub.RegisterTool(plugin.LLMTool{
		Definition: generic.StaticDefinitionExec,
		Execute: func(_ context.Context, args string) (string, error) {
			return p.get().Dispatch("exec", args), nil
		},
	})
	if _, err := exec.LookPath("date"); err == nil {
		hub.RegisterTool(plugin.LLMTool{
			Definition: generic.StaticDefinitionSystemDate,
			Execute: func(_ context.Context, args string) (string, error) {
				return p.get().Dispatch("system_date", args), nil
			},
		})
	}
	p.registerPreprocessor(hub)
	return nil
}

func (p *Plugin) registerPreprocessor(hub plugin.Hub) {
	hub.RegisterPreprocessor(func(ctx context.Context, text string) (string, error) {
		if home, err := os.UserHomeDir(); err == nil {
			text = replaceAll(text, "{{home}}", home)
		}

		text = replaceAll(text, "{{uuid}}", uuid.New().String())

		text = replaceAll(text, "{{date}}", time.Now().Format("2006-01-02"))
		text = replaceAll(text, "{{now}}", time.Now().Format("2006-01-02 15:04:05"))

		return text, nil
	})
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	return strings.ReplaceAll(s, old, new)
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
