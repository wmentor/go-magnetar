package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// TestJiraTaskSearch_FetchIssuesByJQL tests the FetchIssuesByJQL method
func TestJiraTaskSearch_FetchIssuesByJQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jql      string
		maxRes   int
		startAt  int
		response string
		wantErr  bool
		checkRes func(t *testing.T, result string, err error)
	}{
		{
			name:    "successful search with results",
			jql:     "project = GOARCH AND status = Open",
			maxRes:  10,
			startAt: 0,
			response: `{
				"total": 2,
				"startAt": 0,
				"maxResults": 10,
				"issues": [
					{
						"key": "GOARCH-60",
						"id": "12345",
						"self": "https://jira.example.com/rest/api/2/issue/12345",
						"fields": {
							"summary": "Issue summary 1",
							"description": "Issue description 1",
							"status": {"name": "Open"},
							"assignee": {"displayName": "John Doe"},
							"reporter": {"displayName": "Jane Smith"},
							"project": {"name": "Go Architecture"},
							"issuetype": {"name": "Task"},
							"created": "2024-01-01T10:00:00Z",
							"updated": "2024-01-01T11:00:00Z",
							"comment": {"comments": []}
						}
					},
					{
						"key": "GOARCH-61",
						"id": "12346",
						"self": "https://jira.example.com/rest/api/2/issue/12346",
						"fields": {
							"summary": "Issue summary 2",
							"description": "Issue description 2",
							"status": {"name": "In Progress"},
							"assignee": {"displayName": "Bob Wilson"},
							"reporter": {"displayName": "Alice Brown"},
							"project": {"name": "Go Architecture"},
							"issuetype": {"name": "Bug"},
							"created": "2024-01-02T10:00:00Z",
							"updated": "2024-01-02T11:00:00Z",
							"comment": {"comments": []}
						}
					}
				]
			}`,
			wantErr: false,
			checkRes: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "Found 2 issues")
				require.Contains(t, result, "GOARCH-60")
				require.Contains(t, result, "Issue summary 1")
				require.Contains(t, result, "GOARCH-61")
				require.Contains(t, result, "Issue summary 2")
			},
		},
		{
			name:    "empty search results",
			jql:     "project = NONEXISTENT",
			maxRes:  10,
			startAt: 0,
			response: `{
				"total": 0,
				"startAt": 0,
				"maxResults": 10,
				"issues": []
			}`,
			wantErr: false,
			checkRes: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "Found 0 issues")
			},
		},
		{
			name:     "API returns error status",
			jql:      "invalid query",
			maxRes:   10,
			startAt:  0,
			response: `{"errorMessages": ["Query parse error"]}`,
			wantErr:  true,
			checkRes: func(t *testing.T, result string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "search returned status 400")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a test server to mock JIRA API
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request is POST to search endpoint
				require.Equal(t, http.MethodPost, r.Method)
				require.Contains(t, r.URL.Path, "/rest/api/2/search")

				// Read and verify request body
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NotEmpty(t, body)

				// Parse request body to verify JQL is present
				var requestBody struct {
					JQL string `json:"jql"`
				}
				err = json.Unmarshal(body, &requestBody)
				require.NoError(t, err)
				require.Contains(t, requestBody.JQL, tt.jql)

				// Set response
				w.Header().Set("Content-Type", "application/json")
				if tt.wantErr {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			// Create config
			tmpDir := t.TempDir()
			configPath := tmpDir + "/config.yml"
			cfg := `version: "1.0"
jira:
  base_url: "` + server.URL + `"
  api_key: "test-api-key"
`
			if err := writeStringToFile(configPath, cfg); err != nil {
				t.Fatal(err)
			}

			config, err := config.Load(configPath)
			require.NoError(t, err)

			// Create printer (verbose enabled to capture tool call logs)
			var buf bytes.Buffer
			printer := printer.New(&buf, true)

			// Create tool
			tool := NewSearchTool(config, printer, requester.NewFactory())

			// Create params
			params := &JiraTaskSearchParams{
				JQL:        tt.jql,
				MaxResults: tt.maxRes,
				StartAt:    tt.startAt,
			}

			// Call the method
			ctx := context.Background()
			res, err := tool.FetchIssuesByJQL(ctx, params)

			// Parse result if not error
			if err == nil && res != nil {
				if resp, ok := res.(*tools.Response); ok {
					tt.checkRes(t, resp.Result, err)
				}
			} else {
				tt.checkRes(t, "", err)
			}
		})
	}
}

// TestJiraTaskSearch_Exec tests the Exec method// TestJiraTaskSearch_Exec tests the Exec method
func TestJiraTaskSearch_Exec(t *testing.T) {
	t.Parallel()

	// Create config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  base_url: "https://jira.example.com"
  api_key: "test-api-key"
`
	if err := writeStringToFile(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	config, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer
	var buf bytes.Buffer
	printer := printer.New(&buf, true)

	// Create factory
	factory := requester.NewFactory()

	// Create tool
	tool := NewSearchTool(config, printer, factory)

	// Test with nil params
	_, err = tool.Exec(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")

	// Test with valid params (but no real server, so will fail)
	params := &JiraTaskSearchParams{
		JQL:        "project = TEST",
		MaxResults: 10,
		StartAt:    0,
	}
	_, err = tool.Exec(context.Background(), params)
	// Expect error since we don't have a real JIRA server
	require.Error(t, err)
}

// TestJiraTaskSearch_FetchIssuesByJQL_MockClient tests with mocked client using MockClient
func TestJiraTaskSearch_FetchIssuesByJQL_MockClient(t *testing.T) {
	t.Parallel()

	// Create config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  base_url: "https://jira.example.com"
  api_key: "test-api-key"
`
	if err := writeStringToFile(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	config, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer
	var buf bytes.Buffer
	printer := printer.New(&buf, true)

	// Create mock client factory
	factoryMock := testdata.NewClientFactory(t)

	// Create tool
	tool := NewSearchTool(config, printer, factoryMock)

	// Test with nil params
	_, err = tool.FetchIssuesByJQL(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")
}

// TestJiraTaskSearch_Name tests the Name method
func TestJiraTaskSearch_Name(t *testing.T) {
	t.Parallel()

	// Create config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  base_url: "https://jira.example.com"
  api_key: "test-api-key"
`
	if err := writeStringToFile(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	config, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer
	var buf bytes.Buffer
	printer := printer.New(&buf, true)

	// Create factory
	factory := requester.NewFactory()

	// Create tool
	tool := NewSearchTool(config, printer, factory)

	// Test Name()
	name := tool.Name()
	require.Equal(t, "jira_task_search", name)
}

// TestJiraTaskSearch_Description tests the Description method
func TestJiraTaskSearch_Description(t *testing.T) {
	t.Parallel()

	// Create config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"
jira:
  base_url: "https://jira.example.com"
  api_key: "test-api-key"
`
	if err := writeStringToFile(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	config, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer
	var buf bytes.Buffer
	printer := printer.New(&buf, true)

	// Create factory
	factory := requester.NewFactory()

	// Create tool
	tool := NewSearchTool(config, printer, factory)

	// Test Description()
	description := tool.Description()
	require.Contains(t, description, "Search JIRA issues")
	require.Contains(t, description, "JQL")
}

// writeStringToFile writes a string to a file
func writeStringToFile(path string, content string) error {
	return writeStringToFileWithMode(path, content, 0o644)
}

func writeStringToFileWithMode(path string, content string, mode int) error {
	return os.WriteFile(path, []byte(content), os.FileMode(mode))
}
