package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const jiraDefaultTimeout = time.Minute

// JiraTaskGetParams represents parameters for the jira_task_get tool.
type JiraTaskGetParams struct {
	IssueKey string `json:"issue_key" jsonschema_description:"JIRA issue key (e.g., TASK-60)"`
}

// GetTool provides jira_task_get operation as LLM tool.
type GetTool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// NewGetTool creates a new GetTool instance for jira_task_get.
func NewGetTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *GetTool {
	return &GetTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *GetTool) Name() string {
	return "jira_task_get"
}

// Description returns the tool description.
func (t *GetTool) Description() string {
	return "Fetch a JIRA issue and return its details in Markdown format"
}

// FetchIssue fetches a JIRA issue by its key (e.g., GOARCH-60) and returns its content in Markdown.
func (t *GetTool) FetchIssue(ctx context.Context, params *JiraTaskGetParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "jira_task_get", "issue_key", params.IssueKey)

	ctx, cancel := context.WithTimeout(ctx, jiraDefaultTimeout)
	defer cancel()

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s", t.cfg.String("jira.base_url"), params.IssueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("jira_task_get: failed to create request for issue %q: %w", params.IssueKey, err)
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.String("jira.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira_task_get: failed to fetch issue %q: %w", params.IssueKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira_task_get: issue %q returned status %d", params.IssueKey, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("jira_task_get: decode issue %q error: %w", params.IssueKey, err)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("jira_task_get: failed to read response body: %w", err)
	}

	var result struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
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
				Name string `json:"name"`
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
			Labels []string `json:"labels"`
			Parent struct {
				Key string `json:"key"`
			} `json:"parent"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("jira_task_get: failed to parse response: %w", err)
	}

	var children []struct {
		Key    string `json:"key"`
		ID     string `json:"id"`
		Self   string `json:"self"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Type struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	}

	if result.Fields.Type.Name == "Epic" { //nolint:nestif // Later.
		childURL := fmt.Sprintf("%s/rest/api/2/search", t.cfg.String("jira.base_url"))
		childQuery := fmt.Sprintf(`"Epic Link" = %s`, params.IssueKey)
		requestBody := struct {
			JQL        string   `json:"jql"`
			MaxResults int      `json:"maxResults"`
			StartAt    int      `json:"startAt"`
			Fields     []string `json:"fields"`
		}{
			JQL:        childQuery,
			MaxResults: 100,
			StartAt:    0,
			Fields:     []string{"id", "key", "summary", "status", "issuetype"},
		}

		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("jira_task_get: failed to marshal children request body: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, childURL, strings.NewReader(string(bodyBytes)))
		if err != nil {
			return nil, fmt.Errorf("jira_task_get: failed to create children request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+t.cfg.String("jira.api_key"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				contentType := resp.Header.Get("Content-Type")
				utf8, err := charset.NewReader(resp.Body, contentType)
				if err == nil {
					childBody, err := io.ReadAll(utf8)
					if err == nil {
						var childResult struct {
							Issues []struct {
								Key    string `json:"key"`
								ID     string `json:"id"`
								Self   string `json:"self"`
								Fields struct {
									Summary string `json:"summary"`
									Status  struct {
										Name string `json:"name"`
									} `json:"status"`
									Type struct {
										Name string `json:"name"`
									} `json:"issuetype"`
								} `json:"fields"`
							} `json:"issues"`
						}
						if err := json.Unmarshal(childBody, &childResult); err == nil {
							children = childResult.Issues
						}
					}
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("Issue: ")
	sb.WriteString(result.Key)
	sb.WriteString("\nSummary: ")
	sb.WriteString(result.Fields.Summary)
	sb.WriteString("\nStatus: ")
	sb.WriteString(result.Fields.Status.Name)
	sb.WriteString("\nAssignee: ")
	sb.WriteString(result.Fields.Assignee.DisplayName)
	sb.WriteString("\nReporter: ")
	sb.WriteString(result.Fields.Reporter.DisplayName)
	sb.WriteString("\nProject: ")
	sb.WriteString(result.Fields.Project.Name)
	sb.WriteString("\nType: ")
	sb.WriteString(result.Fields.Type.Name)
	sb.WriteString("\nCreated: ")
	sb.WriteString(result.Fields.Created)
	sb.WriteString("\nUpdated: ")
	sb.WriteString(result.Fields.Updated)
	sb.WriteString("\nDescription:\n")
	sb.WriteString(result.Fields.Description)

	if len(result.Fields.Labels) > 0 {
		sb.WriteString("\nLabels: ")
		for i, label := range result.Fields.Labels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(label)
		}
	}

	if len(result.Fields.Parent.Key) > 0 {
		sb.WriteString("\nParent: ")
		sb.WriteString(result.Fields.Parent.Key)
	}

	if len(children) > 0 {
		fmt.Fprintf(&sb, "\n\nChildren (%d):\n", len(children))
		for i, child := range children {
			fmt.Fprintf(&sb, "%d. [%s] %s (%s)\n", i+1, child.Key, child.Fields.Summary, child.Fields.Status.Name)
		}
	}

	if len(result.Fields.Comment.Comments) > 0 {
		sb.WriteString("\n\nComments:\n")
		for i, comment := range result.Fields.Comment.Comments {
			fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, comment.Author.DisplayName, comment.Created)
			sb.WriteString(comment.Body)
			if i < len(result.Fields.Comment.Comments)-1 {
				sb.WriteString("\n\n")
			}
		}
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}

// Exec executes the jira_task_get tool and returns the result.
func (t *GetTool) Exec(ctx context.Context, params *JiraTaskGetParams) (any, error) {
	return t.FetchIssue(ctx, params)
}

// JiraTools provides backward compatibility for existing code.
// Deprecated: Use GetTool, SearchTool, or CreateTool instead.
type JiraTools struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// New creates a new JiraTools instance.
// Deprecated: Use NewGetTool, NewSearchTool, or NewCreateTool instead.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *JiraTools {
	return &JiraTools{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// FetchIssue fetches a JIRA issue by its key (e.g., GOARCH-60) and returns its content in Markdown.
// Deprecated: Use GetTool.FetchIssue with proper context and params instead.
func (j *JiraTools) FetchIssue(issueKey string) (string, error) {
	result, err := (&GetTool{
		cfg:           j.cfg,
		printer:       j.printer,
		clientFactory: j.clientFactory,
	}).FetchIssue(context.Background(), &JiraTaskGetParams{IssueKey: issueKey})
	if err != nil {
		return "", err
	}
	if resp, ok := result.(*tools.Response); ok {
		return resp.Result, nil
	}
	return "", fmt.Errorf("unexpected result type: %T", result)
}
