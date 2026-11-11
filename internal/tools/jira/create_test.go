package jira

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// TestJiraIssueCreate_CreateIssue tests the CreateIssue method
func TestJiraIssueCreate_CreateIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     *JiraIssueCreateParams
		baseURL    string
		setupMocks func(t *testing.T, client *testdata.Client)
		wantErr    string
		check      func(t *testing.T, result string)
	}{
		{
			name: "successful issue creation",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test issue summary",
				Description: "Test description in **Markdown** format",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				responsePayload := `{
					"id": "10001",
					"key": "TASK-123",
					"self": "https://jira.example.com/rest/api/2/issue/10001"
				}`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodPost, req.Method)
						require.Equal(t, "https://jira.example.com/rest/api/2/issue", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Accept"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))

						// Verify request body
						body, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						var requestBody struct {
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
						}
						err = json.Unmarshal(body, &requestBody)
						require.NoError(t, err)
						require.Equal(t, "TASK", requestBody.Fields.Project.Key)
						require.Equal(t, "Test issue summary", requestBody.Fields.Summary)
						require.Contains(t, requestBody.Fields.Description, "Test description")
						require.Equal(t, "Task", requestBody.Fields.IssueType.Name)

						return newMockHTTPResponse(http.StatusCreated, responsePayload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Issue created successfully")
				require.Contains(t, result, "ID: 10001")
				require.Contains(t, result, "Key: TASK-123")
				require.Contains(t, result, "Self: https://jira.example.com/rest/api/2/issue/10001")
			},
		},
		{
			name: "issue creation with empty description",
			params: &JiraIssueCreateParams{
				ProjectKey:  "BUG",
				Summary:     "Bug description",
				Description: "",
				IssueType:   "Bug",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				responsePayload := `{
					"id": "20005",
					"key": "BUG-456",
					"self": "https://jira.example.com/rest/api/2/issue/20005"
				}`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusCreated, responsePayload), nil).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Issue created successfully")
				require.Contains(t, result, "Key: BUG-456")
			},
		},
		{
			name: "HTTP error response",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusBadRequest, `{"errorMessages":["Field 'summary' is required"]}`), nil).Once()
			},
			wantErr: "create issue returned status 400",
		},
		{
			name: "HTTP 401 unauthorized",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusUnauthorized, `{"errorMessages":["Invalid credentials"]}`), nil).Once()
			},
			wantErr: "create issue returned status 401",
		},
		{
			name: "HTTP transport error",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(nil, errors.New("connection refused")).Once()
			},
			wantErr: "failed to create issue",
		},
		{
			name:    "nil params returns error",
			params:  nil,
			baseURL: "https://jira.example.com",
			wantErr: "invalid params",
			setupMocks: func(t *testing.T, _ *testdata.Client) {
				t.Helper()
				// Don't set up any mocks since nil params means no HTTP call
			},
		},
		{
			name: "invalid JSON response",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusCreated, `{invalid json`), nil).Once()
			},
			wantErr: "failed to parse response",
		},
		{
			name: "empty response body",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusCreated, ""), nil).Once()
			},
			wantErr: "decode response",
		},
		{
			name: "response body read error",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(&http.Response{
						StatusCode: http.StatusCreated,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body: struct {
							io.Reader
							io.Closer
						}{
							Reader: io.MultiReader(strings.NewReader(`{"id":"1","key":"X"}`), errReadCloser{err: errors.New("stream broken")}),
							Closer: io.NopCloser(nil),
						},
					}, nil).Once()
			},
			wantErr: "decode response",
		},
		{
			name: "malformed base URL returns request creation error",
			params: &JiraIssueCreateParams{
				ProjectKey:  "TASK",
				Summary:     "Test",
				Description: "Test description",
				IssueType:   "Task",
			},
			baseURL: "://invalid-url",
			wantErr: "failed to create request",
			setupMocks: func(t *testing.T, _ *testdata.Client) {
				t.Helper()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newGetTestConfig(t, tt.baseURL)
			var buf bytes.Buffer
			p := printer.New(&buf, true)

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			// Skip Get/Put expectations for nil params tests
			if tt.params != nil {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Return().Once()
			}
			tt.setupMocks(t, client)

			tool := NewCreateTool(cfg, p, factory)
			res, err := tool.CreateIssue(t.Context(), tt.params)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			resp, ok := res.(*tools.Response)
			require.True(t, ok, "expected *tools.Response, got %T", res)
			require.Equal(t, tools.StatusOK, resp.Status)
			if tt.check != nil {
				tt.check(t, resp.Result)
			}
		})
	}
}

// TestJiraIssueCreate_ParamsMetaAndLegacy tests the Name, Description, and Exec methods
func TestJiraIssueCreate_ParamsMetaAndLegacy(t *testing.T) {
	t.Parallel()

	cfg := newGetTestConfig(t, "https://jira.example.com")
	var buf bytes.Buffer
	p := printer.New(&buf, false)

	factory := testdata.NewClientFactory(t)
	tool := NewCreateTool(cfg, p, factory)

	require.Equal(t, "jira_issue_create", tool.Name())
	require.Contains(t, tool.Description(), "Create a new JIRA issue")
	require.Contains(t, tool.Description(), "project")

	// Test with nil params
	_, err := tool.CreateIssue(t.Context(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")

	_, err = tool.Exec(t.Context(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")

	// Test with valid params (mocked) - need to reset factory to avoid duplicate expectations
	// For the invalid params tests, no HTTP calls are made, so we need separate factory
	client := testdata.NewClient(t)
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()
	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
		func(_ *http.Request) (*http.Response, error) {
			return newMockHTTPResponse(http.StatusCreated, `{"id":"1","key":"TASK-1","self":"https://jira.example.com/rest/api/2/issue/1"}`), nil
		},
	).Once()

	execRes, err := tool.Exec(t.Context(), &JiraIssueCreateParams{
		ProjectKey:  "TASK",
		Summary:     "From Exec",
		Description: "Test description",
		IssueType:   "Task",
	})
	require.NoError(t, err)
	resp, ok := execRes.(*tools.Response)
	require.True(t, ok)
	require.Contains(t, resp.Result, "TASK-1")
}

// TestJiraIssueCreate_MarkdownConversion tests that Markdown is converted to Jira format
func TestJiraIssueCreate_MarkdownConversion(t *testing.T) {
	t.Parallel()

	cfg := newGetTestConfig(t, "https://jira.example.com")
	var buf bytes.Buffer
	p := printer.New(&buf, true)

	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			// Parse to verify the description field
			var requestBody struct {
				Fields struct {
					Description string `json:"description"`
				} `json:"fields"`
			}
			err = json.Unmarshal(body, &requestBody)
			require.NoError(t, err)

			// Verify that Markdown was converted to Jira format
			// **bold** -> *bold*, __italic__ -> *italic*, # header -> h1. header
			require.Contains(t, requestBody.Fields.Description, "*bold text*")
			require.Contains(t, requestBody.Fields.Description, "*italic text*")
			require.Contains(t, requestBody.Fields.Description, "h1. header1")

			return newMockHTTPResponse(http.StatusCreated, `{"id":"1","key":"TASK-1","self":"https://jira.example.com/rest/api/2/issue/1"}`), nil
		},
	).Once()
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()

	tool := NewCreateTool(cfg, p, factory)
	_, err := tool.CreateIssue(t.Context(), &JiraIssueCreateParams{
		ProjectKey:  "TASK",
		Summary:     "Test with Markdown",
		Description: "**bold text**\n\n__italic text__\n\n# header1",
		IssueType:   "Task",
	})
	require.NoError(t, err)
}

// TestJiraIssueCreate_ToolCallLogging tests that tool calls are properly logged
func TestJiraIssueCreate_ToolCallLogging(t *testing.T) {
	t.Parallel()

	cfg := newGetTestConfig(t, "https://jira.example.com")
	var buf bytes.Buffer
	p := printer.New(&buf, true)

	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newMockHTTPResponse(http.StatusCreated, `{"id":"1","key":"TASK-1","self":"https://jira.example.com/rest/api/2/issue/1"}`), nil).Once()
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Return().Once()

	tool := NewCreateTool(cfg, p, factory)
	_, err := tool.CreateIssue(t.Context(), &JiraIssueCreateParams{
		ProjectKey:  "PROJ",
		Summary:     "Test Summary",
		Description: "Test description",
		IssueType:   "Bug",
	})
	require.NoError(t, err)

	// Verify that tool call was logged
	logOutput := buf.String()
	require.Contains(t, logOutput, "jira_issue_create")
	require.Contains(t, logOutput, "PROJ")
	require.Contains(t, logOutput, "Test Summary")
	require.Contains(t, logOutput, "Bug")
	require.Contains(t, logOutput, "TASK-1")
}
