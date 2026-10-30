package ragplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/store"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

func init() {
	plugin.Register("rag", &Plugin{})
}

type Plugin struct {
	tool *rag.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	// Only register rag_search if RAG is enabled
	cfg := s.Config
	if cfg.Bool("rag.enable") && cfg.String("rag.llm.base_url") != "" {
		// Create store instance
		vecStore, err := store.NewStore(cfg)
		if err != nil {
			return err
		}

		// Create RAG tool with the store instance
		p.tool = rag.New(cfg, s.Printer, vecStore, s)
		s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec, tools.OptionSearch())
	}
	return nil
}
