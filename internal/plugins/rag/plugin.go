package ragplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("rag", &Plugin{})
}

// Plugin wraps the RAG tools.
// rag_search tool is no longer exposed to LLM (replaced by unified search tool).
type Plugin struct{}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	return nil
}
