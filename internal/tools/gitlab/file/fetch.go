package file

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const gitlabDefaultTimeout = time.Minute

// GitLabFileParams represents parameters for the gitlab_file tool.
type GitLabFileParams struct {
	ProjectPath string `json:"project_path" jsonschema_description:"GitLab project path (e.g., 'namespace/project' or 'group/subgroup/project')"`
	Branch      string `json:"branch" jsonschema_description:"Branch name (default: 'main')"`
	File        string `json:"file" jsonschema_description:"Path to the file in the repository"`
}

// Tool provides gitlab_file operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "gitlab_file"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Fetch a file from GitLab repository and return its content"
}

// Exec executes the gitlab_file tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *GitLabFileParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	// Default branch to 'main' if not specified
	branch := params.Branch
	if branch == "" {
		branch = "main"
	}

	t.printer.ToolCall(printer.IconSearch, "gitlab_file", "project_path", params.ProjectPath, "branch", branch, "file", params.File)

	ctx, cancel := context.WithTimeout(ctx, gitlabDefaultTimeout)
	defer cancel()

	// URL-encode the project path and file path
	encodedProjectPath := url.PathEscape(params.ProjectPath)
	encodedFilePath := url.PathEscape(params.File)

	// Build API URL
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s?ref=%s",
		t.cfg.String("gitlab.base_url"), encodedProjectPath, encodedFilePath, branch)

	client := t.clientFactory.Get()
	defer t.clientFactory.Put(client)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to create request for file %q in project %q: %w",
			params.File, params.ProjectPath, err)
	}

	request.Header.Set("Authorization", "Bearer "+t.cfg.String("gitlab.api_key"))
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to fetch file %q in project %q: %w",
			params.File, params.ProjectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: file %q in project %q returned status %d",
			params.File, params.ProjectPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to read response body: %w", err)
	}

	var fileData struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		SHA         string `json:"sha"`
		Size        int    `json:"size"`
		Encoding    string `json:"encoding"`
		Content     string `json:"content,omitempty"`
		Checkout    string `json:"checkout"`
		Reference   string `json:"ref"`
		ExternalURL string `json:"external_url"`
	}

	if err := json.Unmarshal(body, &fileData); err != nil {
		return nil, fmt.Errorf("gitlab: failed to parse response: %w", err)
	}

	if fileData.Content == "" {
		return nil, fmt.Errorf("gitlab: file %s has no content", params.File)
	}

	var content string
	if fileData.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileData.Content)
		if err != nil {
			return nil, fmt.Errorf("gitlab: failed to decode file %s: %w", params.File, err)
		}
		content = string(decoded)
	} else {
		content = fileData.Content
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: content,
	}, nil
}
