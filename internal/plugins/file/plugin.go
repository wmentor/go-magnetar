package fileplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/file/edit"
	"github.com/wmentor/go-magnetar/internal/tools/file/list"
	"github.com/wmentor/go-magnetar/internal/tools/file/read"
	"github.com/wmentor/go-magnetar/internal/tools/file/write"
)

func init() {
	plugin.Register("file", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	tool  *read.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s
	cfg := s.ConcreteConfig()
	p.tool = read.New(cfg, s.Printer, plugin.GetRoot(), s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)
	s.ToolStore.RegisterOurToolMCP(p.tool.NameMCP(), p.tool.Description(), p.tool.Exec)

	// Register file_list tool
	listTool := list.New(cfg, s.Printer, plugin.GetRoot(), s)
	s.ToolStore.Register(listTool.Name(), listTool.Description(), listTool.Exec)

	// Register file_write tool
	writeTool := write.New(cfg, s.Printer, plugin.GetRoot(), s)
	s.ToolStore.Register(writeTool.Name(), writeTool.Description(), writeTool.Exec, tools.OptionSequential())

	// Register edit tool
	editTool := edit.NewTool(cfg, s.Printer, plugin.GetRoot(), s)
	s.ToolStore.Register(editTool.Name(), editTool.Description(), editTool.FileEdit, tools.OptionSequential())
	return nil
}
