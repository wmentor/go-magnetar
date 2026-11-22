package jiraplug

import (
	"context"
	"fmt"
	"strings"

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

	cfg := s.ConcreteConfig()

	// Initialize tools with Printer and RequesterFactory for proper dependency injection
	p.getTool = jira.NewGetTool(cfg, s.Printer, s.RequesterFactory)
	p.searchTool = jira.NewSearchTool(cfg, s.Printer, s.RequesterFactory)
	p.createTool = jira.NewCreateTool(cfg, s.Printer, s.RequesterFactory)

	// Register jira_task_get tool
	s.ToolStore.Register(p.getTool.Name(), p.getTool.Description(), p.getTool.Exec)

	// Register jira_task_search tool
	s.ToolStore.Register(p.searchTool.Name(), p.searchTool.Description(), p.searchTool.Exec)

	// Register jira_issue_create tool if not in read-only mode
	if !s.Config.Bool("jira.readonly") {
		s.ToolStore.Register(p.createTool.Name(), p.createTool.Description(), p.createTool.Exec)
	}

	s.WebProcStore.AddProcessorFunc(p.webFetchProc)

	return nil
}

func (p *Plugin) webFetchProc(ctx context.Context, urlSrc string) (string, error) {
	cfg := p.state.Config
	if strings.HasPrefix(urlSrc, cfg.String("jira.base_url")) && (strings.Contains(urlSrc, "/browse/") || strings.Contains(urlSrc, "/issues/")) {
		issueKey, err := ExtractIssueKeyFromJIRAURL(urlSrc)
		if err == nil && issueKey != "" {
			return jira.New(cfg, p.state.Printer, p.state.RequesterFactory).FetchIssue(issueKey)
		}
	}
	return "", plugin.ErrNoWebProcessor
}

// ExtractIssueKeyFromJIRAURL extracts the issue key (e.g., GOARCH-60) from a JIRA URL.
func ExtractIssueKeyFromJIRAURL(url string) (string, error) {
	for _, path := range []string{"/browse/", "/issues/"} {
		_, idPart, ok := strings.Cut(url, path)
		if !ok {
			continue
		}
		if idx := strings.IndexAny(idPart, "/?#"); idx != -1 {
			idPart = idPart[:idx]
		}
		if idPart == "" {
			return "", fmt.Errorf("issue key is empty")
		}
		return idPart, nil
	}

	return "", fmt.Errorf("not a JIRA issue URL")
}
