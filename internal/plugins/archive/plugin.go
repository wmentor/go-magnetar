package archiveplugin

import (
	_ "github.com/wmentor/go-magnetar/internal/archive/zip"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/archive/list"
	"github.com/wmentor/go-magnetar/internal/tools/archive/read"
)

func init() {
	plugin.Register("archive", &Plugin{})
}

// Plugin implements the plugin.Plugin interface for archive tools.
type Plugin struct {
	state    *plugin.State
	listTool *list.Tool
	readTool *read.Tool
}

// Init initializes the plugin and registers the tools.
func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	p.listTool = list.NewTool(s.ConcreteConfig(), s.Printer)
	p.readTool = read.NewTool(s.ConcreteConfig(), s.Printer)

	s.ToolStore.Register(p.listTool.Name(), p.listTool.Description(), p.listTool.Exec)
	s.ToolStore.Register(p.readTool.Name(), p.readTool.Description(), p.readTool.Exec)

	return nil
}
