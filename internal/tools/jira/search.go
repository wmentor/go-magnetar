package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// JiraTaskSearchParams represents parameters for the jira_task_search tool.
type JiraTaskSearchParams struct {
	JQL        string `json:"jql" jsonschema_description:"JIRA Query Language (JQL) search query (e.g., 'project = GOARCH AND status = Open')"`
	MaxResults int    `json:"max_results" jsonschema_description:"Maximum number of results to return (default: 100)"`
	StartAt    int    `json:"start_at" jsonschema_description:"Starting index for pagination (default: 0)"`
}

// SearchTool provides jira_task_search operation as LLM tool.
type SearchTool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// NewSearchTool creates a new SearchTool instance for jira_task_search.
func NewSearchTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *SearchTool {
	return &SearchTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return "jira_task_search"
}

// Description returns the tool description.
func (t *SearchTool) Description() string {
	return "Search JIRA issues using a JQL query and return an array of issue keys and summaries " +
		"with pagination support. Use jira_task_get to fetch details for a specific issue."
}

// FetchIssuesByJQL searches JIRA issues using a JQL query and returns results as an array of {key, summary} objects.
func (t *SearchTool) FetchIssuesByJQL(ctx context.Context, params *JiraTaskSearchParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	// Set default values
	if params.MaxResults <= 0 {
		params.MaxResults = 100
	}
	if params.StartAt < 0 {
		params.StartAt = 0
	}

	t.printer.ToolCall(printer.IconSearch, "jira_task_search", "jql", params.JQL, "max_results", params.MaxResults, "start_at", params.StartAt)

	ctx, cancel := context.WithTimeout(ctx, jiraDefaultTimeout)
	defer cancel()

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	apiURL := fmt.Sprintf("%s/rest/api/2/search", t.cfg.String("jira.base_url"))

	requestBody := struct {
		JQL        string   `json:"jql"`
		MaxResults int      `json:"maxResults"`
		StartAt    int      `json:"startAt"`
		Fields     []string `json:"fields"`
	}{
		JQL:        params.JQL,
		MaxResults: params.MaxResults,
		StartAt:    params.StartAt,
		Fields:     []string{"summary", "description", "status", "assignee", "reporter", "project", "issuetype", "created", "updated", "comment"},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("jira_task_search: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("jira_task_search: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.String("jira.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira_task_search: failed to fetch issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira_task_search: search returned status %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("jira_task_search: decode error: %w", err)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("jira_task_search: failed to read response body: %w", err)
	}

	var result struct {
		Total      int `json:"total"`
		Start      int `json:"startAt"`
		MaxResults int `json:"maxResults"`
		Issues     []struct {
			Key    string `json:"key"`
			ID     string `json:"id"`
			Self   string `json:"self"`
			Fields struct {
				Summary     string `json:"summary"`
				Description string `json:"description"`
				Status      struct {
					Name string `json:"name"`
				} `json:"status"`
				Assignee struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				Reporter struct {
					DisplayName string `json:"displayName"`
				} `json:"reporter"`
				Project struct {
					Name string `json:"name"`
				} `json:"project"`
				Type struct {
					Name string `json:"issuetype"`
				} `json:"issuetype"`
				Created string `json:"created"`
				Updated string `json:"updated"`
				Comment struct {
					Comments []struct {
						Author struct {
							DisplayName string `json:"displayName"`
						} `json:"author"`
						Body    string `json:"body"`
						Created string `json:"created"`
					} `json:"comments"`
				} `json:"comment"`
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("jira_task_search: failed to parse response: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d issues (showing %d-%d):\n", result.Total, result.Start+1, result.Start+len(result.Issues))
	sb.WriteString("\n")
	sb.WriteString("[\n")

	for i, issue := range result.Issues {
		fmt.Fprintf(&sb, "  {\"key\": \"%s\", \"summary\": %q}", issue.Key, issue.Fields.Summary)
		if i < len(result.Issues)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("]")

	return &tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}

// Exec executes the jira_task_search tool and returns the result.
func (t *SearchTool) Exec(ctx context.Context, params *JiraTaskSearchParams) (any, error) {
	return t.FetchIssuesByJQL(ctx, params)
}
