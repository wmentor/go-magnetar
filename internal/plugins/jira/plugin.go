package jiraplug

import (
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/jira"
)

func init() {
	plugin.Register("jira", &Plugin{})
}

// Plugin wraps the JIRA tools and exposes them as LLM tools.
type Plugin struct {
	state      *plugin.State
	getTool    *jira.GetTool
	searchTool *jira.SearchTool
	createTool *jira.CreateTool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	if !s.Config.Bool("jira.enable") || s.Config.String("jira.base_url") == "" {
		return nil
	}

	// Initialize tools with Printer and RequesterFactory for proper dependency injection
	p.getTool = jira.NewGetTool(s.Config, s.Printer, s.RequesterFactory)
	p.searchTool = jira.NewSearchTool(s.Config, s.Printer, s.RequesterFactory)
	p.createTool = jira.NewCreateTool(s.Config, s.Printer, s.RequesterFactory)

	// Register jira_task_get tool
	s.ToolStore.Register(p.getTool.Name(), p.getTool.Description(), p.getTool.Exec)

	// Register jira_task_search tool
	s.ToolStore.Register(p.searchTool.Name(), p.searchTool.Description(), p.searchTool.Exec)

	// Register jira_issue_create tool if not in read-only mode
	if !s.Config.Bool("jira.readonly") {
		s.ToolStore.Register(p.createTool.Name(), p.createTool.Description(), p.createTool.Exec)
	}

	return nil
}
