package mr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const gitlabDefaultTimeout = time.Minute

// GitLabMRParams represents parameters for the gitlab_fetch_mr tool.
type GitLabMRParams struct {
	ProjectPath string `json:"project_path" jsonschema_description:"GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')"`
	MRID        string `json:"mr_id" jsonschema_description:"Merge request ID (numeric ID or 'iid')"`
}

// Tool provides gitlab_fetch_mr operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory plugin.ClientFactory
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory plugin.ClientFactory) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "gitlab_fetch_mr"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Fetch a GitLab merge request and return its details in Markdown format"
}

// Exec executes the gitlab_fetch_mr tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *GitLabMRParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab_fetch_mr", "project_path", params.ProjectPath, "mr_id", params.MRID)

	ctx, cancel := context.WithTimeout(ctx, gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(params.ProjectPath)

	// Fetch MR details
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s", t.cfg.String("gitlab.base_url"), encodedProjectPath, params.MRID)

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create request for MR %q/%q: %w", params.ProjectPath, params.MRID, err)
	}

	request.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch MR %q/%q: %w", params.ProjectPath, params.MRID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: MR %q/%q returned status %d", params.ProjectPath, params.MRID, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var result struct {
		ID             int    `json:"id"`
		IID            int    `json:"iid"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		State          string `json:"state"`
		Created        string `json:"created_at"`
		Updated        string `json:"updated_at"`
		WebURL         string `json:"web_url"`
		SourceBranch   string `json:"source_branch"`
		TargetBranch   string `json:"target_branch"`
		WorkInProgress bool   `json:"work_in_progress"`
		Author         struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"author"`
		Labels    []string `json:"labels"`
		Milestone struct {
			Title string `json:"title"`
		} `json:"milestone"`
		Draft             bool   `json:"draft"`
		ChangesCount      string `json:"changes_count"`
		FirstCommitAuthor string `json:"first_commit_author"`
		FirstCommitSHA    string `json:"first_commit_sha"`
		LastCommit        struct {
			AuthorName  string `json:"author_name"`
			AuthorEmail string `json:"author_email"`
			CreatedAt   string `json:"created_at"`
		} `json:"last_commit"`
		Project struct {
			Name              string `json:"name"`
			WebURL            string `json:"web_url"`
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab: MR fetched", "project_path", params.ProjectPath, "mr_id", params.MRID)

	// Fetch MR changes (diffs)
	changesURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/changes", t.cfg.String("gitlab.base_url"), encodedProjectPath, params.MRID)
	requestChanges, err := http.NewRequestWithContext(ctx, http.MethodGet, changesURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create changes request for MR %q/%q: %w", params.ProjectPath, params.MRID, err)
	}

	requestChanges.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	requestChanges.Header.Set("Content-Type", "application/json")

	respChanges, err := client.Do(requestChanges)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch MR changes %q/%q: %w", params.ProjectPath, params.MRID, err)
	}
	defer respChanges.Body.Close()

	if respChanges.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: MR changes %q/%q returned status %d", params.ProjectPath, params.MRID, respChanges.StatusCode)
	}

	changesBody, err := io.ReadAll(respChanges.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read changes response body: %w", err)
	}

	var changesResult struct {
		Changes []struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
			AMode   string `json:"a_mode"`
			BMode   string `json:"b_mode"`
			Created bool   `json:"new_file"`
			Deleted bool   `json:"deleted_file"`
			Renamed bool   `json:"renamed_file"`
			Diff    string `json:"diff"`
		} `json:"changes"`
	}

	if err := json.Unmarshal(changesBody, &changesResult); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse changes response: %w", err)
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab: MR changes fetched", "project_path",
		params.ProjectPath, "mr_id", params.MRID, "changes_count", len(changesResult.Changes))

	var sb strings.Builder
	sb.WriteString("Project: ")
	sb.WriteString(params.ProjectPath)
	sb.WriteString("\nTitle: ")
	sb.WriteString(result.Title)
	sb.WriteString("\nStatus: ")
	sb.WriteString(result.State)
	sb.WriteString("\nAuthor: ")
	sb.WriteString(result.Author.Username)
	sb.WriteString("\nSource Branch: ")
	sb.WriteString(result.SourceBranch)
	sb.WriteString("\nTarget Branch: ")
	sb.WriteString(result.TargetBranch)
	if result.Milestone.Title != "" {
		sb.WriteString("\nMilestone: ")
		sb.WriteString(result.Milestone.Title)
	}
	if len(result.Labels) > 0 {
		sb.WriteString("\nLabels: ")
		for i, label := range result.Labels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(label)
		}
	}
	if result.Draft {
		sb.WriteString("\nDraft: true")
	}
	if result.WorkInProgress {
		sb.WriteString("\nWIP: true")
	}
	if result.ChangesCount != "" {
		sb.WriteString("\nChanges: ")
		sb.WriteString(result.ChangesCount)
	}
	sb.WriteString("\nCreated: ")
	sb.WriteString(result.Created)
	sb.WriteString("\nUpdated: ")
	sb.WriteString(result.Updated)
	sb.WriteString("\nWeb URL: ")
	sb.WriteString(result.WebURL)
	sb.WriteString("\n\nDescription:\n")
	sb.WriteString(result.Description)

	if len(changesResult.Changes) > 0 {
		sb.WriteString("\n\nFile Changes:\n")
		for i, diff := range changesResult.Changes {
			fmt.Fprintf(&sb, "%d. ", i+1)
			if diff.Deleted {
				sb.WriteString("Deleted: ")
				sb.WriteString(diff.OldPath)
			} else if diff.Created {
				sb.WriteString("Created: ")
				sb.WriteString(diff.NewPath)
			} else if diff.Renamed {
				sb.WriteString("Renamed: ")
				sb.WriteString(diff.OldPath)
				sb.WriteString(" -> ")
				sb.WriteString(diff.NewPath)
			} else {
				sb.WriteString("Modified: ")
				sb.WriteString(diff.NewPath)
			}
			sb.WriteString("\n")
			sb.WriteString("Diff:\n")
			sb.WriteString(diff.Diff)
			sb.WriteString("\n\n")
		}
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}

// FetchMergeRequestsParams represents parameters for fetching list of merge requests.
type FetchMergeRequestsParams struct {
	ProjectPath string `json:"project_path" jsonschema_description:"GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')"`
	State       string `json:"state" jsonschema_description:"State of MRs to fetch (e.g., 'opened', 'closed', 'merged', 'all'); default is 'opened'"`
}

// FetchMergeRequests fetches a list of merge requests and returns them in Markdown format.
func (t *Tool) FetchMergeRequests(ctx context.Context, params *FetchMergeRequestsParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	if params.State == "" {
		params.State = "opened"
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab_fetch_mr_list", "project_path", params.ProjectPath, "state", params.State)

	ctx, cancel := context.WithTimeout(ctx, gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path
	encodedProjectPath := url.PathEscape(params.ProjectPath)

	// Fetch MR list with state filter
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests?state=%s", t.cfg.String("gitlab.base_url"), encodedProjectPath, params.State)

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create request for MR list %q: %w", params.ProjectPath, err)
	}

	request.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch MR list %q: %w", params.ProjectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: MR list %q returned status %d", params.ProjectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var mrList []struct {
		ID             int    `json:"id"`
		IID            int    `json:"iid"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		State          string `json:"state"`
		Created        string `json:"created_at"`
		WebURL         string `json:"web_url"`
		SourceBranch   string `json:"source_branch"`
		TargetBranch   string `json:"target_branch"`
		WorkInProgress bool   `json:"work_in_progress"`
		Author         struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"author"`
		Labels       []string `json:"labels"`
		ChangesCount string   `json:"changes_count"`
		Project      struct {
			Path string `json:"path"`
		} `json:"project"`
	}

	if err := json.Unmarshal(body, &mrList); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab: MR list fetched", "project_path", params.ProjectPath, "state", params.State, "count", len(mrList))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Merge Requests (%s)\n\n", strings.Title(params.State)))
	sb.WriteString(fmt.Sprintf("Project: %s\n\n", params.ProjectPath))

	if len(mrList) == 0 {
		sb.WriteString("No merge requests found.\n")
		return &tools.Response{
			Status: tools.StatusOK,
			Result: sb.String(),
		}, nil
	}

	sb.WriteString(fmt.Sprintf("Total: %d\n\n", len(mrList)))

	for i, mr := range mrList {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		sb.WriteString(fmt.Sprintf("## MR !%d\n\n", mr.IID))
		sb.WriteString(fmt.Sprintf("**Number:** %d\n\n", mr.IID))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n\n", mr.WebURL))
		sb.WriteString(fmt.Sprintf("**Author:** %s\n\n", mr.Author.Username))
		sb.WriteString(fmt.Sprintf("**Title:** %s\n\n", mr.Title))
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", mr.Description))
		sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", mr.Created))
		sb.WriteString(fmt.Sprintf("**State:** %s\n", mr.State))
		sb.WriteString(fmt.Sprintf("**Source Branch:** %s\n", mr.SourceBranch))
		sb.WriteString(fmt.Sprintf("**Target Branch:** %s\n", mr.TargetBranch))
		if mr.WorkInProgress {
			sb.WriteString("**WIP:** true\n")
		}
		if len(mr.Labels) > 0 {
			sb.WriteString(fmt.Sprintf("**Labels:** %s\n", strings.Join(mr.Labels, ", ")))
		}
		if mr.ChangesCount != "" {
			sb.WriteString(fmt.Sprintf("**Changes:** %s\n", mr.ChangesCount))
		}
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}
