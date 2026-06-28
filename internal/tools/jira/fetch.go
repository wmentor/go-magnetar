package jira

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const jiraDefaultTimeout = time.Minute

// JiraTools provides JIRA issue fetching as an LLM tool.
type JiraTools struct {
	cfg *config.Config
}

// New creates a new JiraTools instance.
func New(cfg *config.Config) *JiraTools {
	return &JiraTools{cfg: cfg}
}

// FetchIssue fetches a JIRA issue by its key (e.g., GOARCH-60) and returns its content in Markdown.
func (j *JiraTools) FetchIssue(issueKey string) (string, error) {
	printer.ToolCall(printer.IconSearch, "jira_task_get", "issue_key", issueKey)

	ctx, cancel := context.WithTimeout(context.Background(), jiraDefaultTimeout)
	defer cancel()

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   jiraDefaultTimeout,
		Transport: tr,
	}

	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s", j.cfg.String("jira.base_url"), issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("jira_task_get: failed to create request for issue %q: %w", issueKey, err)
	}

	req.Header.Set("Authorization", "Bearer "+j.cfg.String("jira.api_key"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("jira_task_get: failed to fetch issue %q: %w", issueKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jira_task_get: issue %q returned status %d", issueKey, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	utf8, err1 := charset.NewReader(resp.Body, contentType)
	if err1 != nil {
		return "", fmt.Errorf("jira_task_get: decode issue %q error: %w", issueKey, err1)
	}

	body, err := io.ReadAll(utf8)
	if err != nil {
		return "", fmt.Errorf("jira_task_get: failed to read response body: %w", err)
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
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("jira_task_get: failed to parse response: %w", err)
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

	if len(result.Fields.Comment.Comments) > 0 {
		sb.WriteString("\n\nComments:\n")
		for i, comment := range result.Fields.Comment.Comments {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, comment.Author.DisplayName, comment.Created))
			sb.WriteString(comment.Body)
			if i < len(result.Fields.Comment.Comments)-1 {
				sb.WriteString("\n\n")
			}
		}
	}

	return sb.String(), nil
}

// Definition returns the OpenAI tool schema for jira_task_get.
func (j *JiraTools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "jira_task_get",
			Description: "Fetch a JIRA issue and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"issue_key": map[string]any{
						"type":        "string",
						"description": "JIRA issue key (e.g., GOARCH-60)",
					},
				},
				"required": []string{"issue_key"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (j *JiraTools) Dispatch(name string, args string) string {
	switch name {
	case "jira_task_get":
		var params struct {
			IssueKey string `json:"issue_key"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error: failed to parse arguments"
		}
		content, err := j.FetchIssue(params.IssueKey)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	default:
		return "error: unknown tool " + name
	}
}

// StaticDefinition returns the OpenAI tool schema for jira_task_get without
// requiring an initialised JiraTools instance.
func StaticDefinition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "jira_task_get",
			Description: "Fetch a JIRA issue and return its details in Markdown format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"issue_key": map[string]any{
						"type":        "string",
						"description": "JIRA issue key (e.g., TASK-60)",
					},
				},
				"required": []string{"issue_key"},
			},
		},
	}
}
