package fileplugin

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
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
	p.tool = read.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)

	// Register file_list tool
	listTool := list.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(listTool.Name(), listTool.Description(), listTool.Exec)

	// Register file_write tool
	writeTool := write.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(writeTool.Name(), writeTool.Description(), writeTool.Exec)

	// Register file_edit and file_edit_multiple tools
	editTool := edit.NewTool(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(editTool.Name(), editTool.Description(), editTool.FileEdit)
	s.ToolStore.Register(editTool.Name()+"_multiple", editTool.Description()+" (multiple edits)", editTool.FileEditMultiple)
	return nil
}
