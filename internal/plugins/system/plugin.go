package systemplugin

import (
	"os/exec"

	"github.com/wmentor/go-magnetar/internal/plugin"
	execTool "github.com/wmentor/go-magnetar/internal/tools/exec"
	sshTool "github.com/wmentor/go-magnetar/internal/tools/ssh"
	"github.com/wmentor/go-magnetar/internal/tools/system/date"
	"github.com/wmentor/go-magnetar/internal/tools/system/grep"
)

func init() {
	plugin.Register("system", &Plugin{})
}

type Plugin struct {
	state *plugin.State
	exec  *execTool.Tool
	ssh   *sshTool.Tool
	tool  *date.Tool
	grep  *grep.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	// Register exec tool
	p.exec = execTool.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(p.exec.Name(), p.exec.Description(), p.exec.Exec)

	// Register ssh tool (only if ssh.enable is true in config)
	if s.Config.Bool("ssh.enable") {
		p.ssh = sshTool.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
		s.ToolStore.Register(p.ssh.Name(), p.ssh.Description(), p.ssh.Exec)
	}

	// Register system_date tool
	p.tool = date.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
	s.ToolStore.Register(p.tool.Name(), p.tool.Description(), p.tool.Exec)

	// Register system_grep tool (only if grep or rg is available)
	if _, err := exec.LookPath("grep"); err == nil {
		p.grep = grep.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
		s.ToolStore.Register(p.grep.Name(), p.grep.Description(), p.grep.Exec)
	} else if _, err := exec.LookPath("rg"); err == nil {
		p.grep = grep.New(s.Config, s.Printer, s.RequesterFactory, plugin.GetRoot(), s)
		s.ToolStore.Register(p.grep.Name(), p.grep.Description(), p.grep.Exec)
	}

	return nil
}
