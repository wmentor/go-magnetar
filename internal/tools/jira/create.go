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
	"github.com/wmentor/go-magnetar/internal/tools/jira/render"
)

// JiraIssueCreateParams represents parameters for the jira_issue_create tool.
type JiraIssueCreateParams struct {
	ProjectKey  string `json:"project_key" jsonschema_description:"JIRA project key (e.g., TASK, PROJ)"`
	Summary     string `json:"summary" jsonschema_description:"Issue summary (short description)"`
	Description string `json:"description" jsonschema_description:"Issue description in Markdown format"`
	IssueType   string `json:"issue_type" jsonschema_description:"Issue type name (e.g., Bug, Task, Story)"`
}

// CreateTool provides jira_issue_create operation as LLM tool.
type CreateTool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// NewCreateTool creates a new CreateTool instance for jira_issue_create.
func NewCreateTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *CreateTool {
	return &CreateTool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *CreateTool) Name() string {
	return "jira_issue_create"
}

// Description returns the tool description.
func (t *CreateTool) Description() string {
	return "Create a new JIRA issue with project, summary, description (Markdown to Jira format), and issue type."
}

// CreateIssue creates a new JIRA issue with the given parameters.
// Description is converted from Markdown to Jira format using md2jira.
func (t *CreateTool) CreateIssue(ctx context.Context, params *JiraIssueCreateParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSave, "jira_issue_create", "project_key", params.ProjectKey,
		"summary", params.Summary, "issue_type", params.IssueType)

	ctx, cancel := context.WithTimeout(ctx, jiraDefaultTimeout)
	defer cancel()

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	// Convert Markdown description to Jira format
	jiraDescription := render.Convert(params.Description)

	// Prepare request body
	requestBody := struct {
		Fields struct {
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	}{
		Fields: struct {
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		}{
			Project: struct {
				Key string `json:"key"`
			}{
				Key: params.ProjectKey,
			},
			Summary:     params.Summary,
			Description: jiraDescription,
			IssueType: struct {
				Name string `json:"name"`
			}{
				Name: params.IssueType,
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("jira_issue_create: failed to marshal request body: %w", err)
	}

	apiURL := fmt.Sprintf("%s/rest/api/2/issue", t.cfg.String("jira.base_url"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("jira_issue_create: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.String("jira.api_key"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira_issue_create: failed to create issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira_issue_create: create issue returned status %d: %s", resp.StatusCode, string(respBody))
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("jira_issue_create: decode response error: %w", err)
	}

	respBody, err := io.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("jira_issue_create: failed to read response body: %w", err)
	}

	var result struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("jira_issue_create: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconDone, "jira_issue_create: issue created successfully", "issue_key", result.Key, "issue_id", result.ID)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: fmt.Sprintf("Issue created successfully:\nID: %s\nKey: %s\nSelf: %s", result.ID, result.Key, result.Self),
	}, nil
}

// Exec executes the jira_issue_create tool and returns the result.
func (t *CreateTool) Exec(ctx context.Context, params *JiraIssueCreateParams) (any, error) {
	return t.CreateIssue(ctx, params)
}
