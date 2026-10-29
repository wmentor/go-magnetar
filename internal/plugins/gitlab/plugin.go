package gitlabplug

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/file"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/mr"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/repo"
	"github.com/wmentor/go-magnetar/internal/tools/gitlab/tree"
)

func init() {
	plugin.Register("gitlab", &Plugin{})
}

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("gitlab.enable") || s.Config.String("gitlab.base_url") == "" {
		return nil
	}

	// Register gitlab_fetch_mr tool
	mrTool := mr.New(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(mrTool.Name(), mrTool.Description(), mrTool.Exec)

	// Register gitlab_repo tool
	repoTool := repo.New(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(repoTool.Name(), repoTool.Description(), repoTool.Exec)

	// Register gitlab_file tool
	fileTool := file.New(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(fileTool.Name(), fileTool.Description(), fileTool.Exec)

	// Register gitlab_tree tool
	treeTool := tree.New(s.Config, s.Printer, s.RequesterFactory)
	s.ToolStore.Register(treeTool.Name(), treeTool.Description(), treeTool.Exec)

	return nil
}
