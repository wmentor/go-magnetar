package jira

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read([]byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}

func newGetTestConfig(t *testing.T, baseURL string) *config.Config {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	content := "version: \"1.0\"\njira:\n  base_url: \"" + baseURL + "\"\n  api_key: \"test-api-key\"\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	return cfg
}

func newMockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestJiraTaskGet_FetchIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		issueKey   string
		baseURL    string
		setupMocks func(t *testing.T, client *testdata.Client)
		wantErr    string
		check      func(t *testing.T, result string)
	}{
		{
			name:     "standard issue with labels parent and comments",
			issueKey: "GOARCH-60",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				payload := `{
					"id": "10001",
					"key": "GOARCH-60",
					"self": "https://jira.example.com/rest/api/2/issue/10001",
					"fields": {
						"summary": "Implement task fetcher",
						"description": "Add robust issue retrieval",
						"status": {"name": "In Progress"},
						"assignee": {"displayName": "Alice"},
						"reporter": {"displayName": "Bob"},
						"project": {"name": "Architecture"},
						"issuetype": {"name": "Task"},
						"created": "2026-01-10T10:00:00Z",
						"updated": "2026-01-11T12:00:00Z",
						"labels": ["backend", "jira"],
						"parent": {"key": "GOARCH-10"},
						"comment": {
							"comments": [
								{
									"author": {"displayName": "Alice"},
									"body": "Initial implementation pushed.",
									"created": "2026-01-10T15:00:00Z"
								},
								{
									"author": {"displayName": "Bob"},
									"body": "Looks good to me.",
									"created": "2026-01-11T09:00:00Z"
								}
							]
						}
					}
				}`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(t, "https://jira.example.com/rest/api/2/issue/GOARCH-60", req.URL.String())
						require.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
						return newMockHTTPResponse(http.StatusOK, payload), nil
					},
				).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Issue: GOARCH-60")
				require.Contains(t, result, "Summary: Implement task fetcher")
				require.Contains(t, result, "Status: In Progress")
				require.Contains(t, result, "Assignee: Alice")
				require.Contains(t, result, "Reporter: Bob")
				require.Contains(t, result, "Project: Architecture")
				require.Contains(t, result, "Type: Task")
				require.Contains(t, result, "Description:\nAdd robust issue retrieval")
				require.Contains(t, result, "Labels: backend, jira")
				require.Contains(t, result, "Parent: GOARCH-10")
				require.Contains(t, result, "Comments:")
				require.Contains(t, result, "1. [Alice] 2026-01-10T15:00:00Z\nInitial implementation pushed.")
				require.Contains(t, result, "2. [Bob] 2026-01-11T09:00:00Z\nLooks good to me.")
			},
		},
		{
			name:     "issue with empty optional fields",
			issueKey: "GOARCH-61",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				payload := `{
					"id": "10002",
					"key": "GOARCH-61",
					"fields": {
						"summary": "Minimal issue",
						"status": {"name": "Open"},
						"project": {"name": "Architecture"},
						"issuetype": {"name": "Bug"}
					}
				}`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, payload), nil).Once()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Issue: GOARCH-61")
				require.Contains(t, result, "Summary: Minimal issue")
				require.NotContains(t, result, "Labels:")
				require.NotContains(t, result, "Parent:")
				require.NotContains(t, result, "Children (")
				require.NotContains(t, result, "Comments:")
			},
		},
		{
			name:     "epic issue fetches child issues",
			issueKey: "EPIC-1",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				epicPayload := `{
					"id": "20001",
					"key": "EPIC-1",
					"fields": {
						"summary": "Platform epic",
						"status": {"name": "In Progress"},
						"issuetype": {"name": "Epic"}
					}
				}`
				childrenPayload := `{
					"issues": [
						{
							"id": "20002",
							"key": "EPIC-2",
							"fields": {
								"summary": "Child story one",
								"status": {"name": "Done"},
								"issuetype": {"name": "Story"}
							}
						},
						{
							"id": "20003",
							"key": "EPIC-3",
							"fields": {
								"summary": "Child task two",
								"status": {"name": "Open"},
								"issuetype": {"name": "Task"}
							}
						}
					]
				}`

				callCount := 0
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						callCount++
						if callCount == 1 {
							require.Equal(t, http.MethodGet, req.Method)
							require.Equal(t, "https://jira.example.com/rest/api/2/issue/EPIC-1", req.URL.String())
							return newMockHTTPResponse(http.StatusOK, epicPayload), nil
						}

						require.Equal(t, http.MethodPost, req.Method)
						require.Equal(t, "https://jira.example.com/rest/api/2/search", req.URL.String())

						body, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						var payload struct {
							JQL string `json:"jql"`
						}
						require.NoError(t, json.Unmarshal(body, &payload))
						require.Equal(t, `"Epic Link" = EPIC-1`, payload.JQL)

						return newMockHTTPResponse(http.StatusOK, childrenPayload), nil
					},
				).Twice()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Type: Epic")
				require.Contains(t, result, "Children (2):")
				require.Contains(t, result, "1. [EPIC-2] Child story one (Done)")
				require.Contains(t, result, "2. [EPIC-3] Child task two (Open)")
			},
		},
		{
			name:     "epic ignores failing children lookup",
			issueKey: "EPIC-9",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				epicPayload := `{
					"id": "20009",
					"key": "EPIC-9",
					"fields": {
						"summary": "Resilient epic",
						"status": {"name": "Open"},
						"issuetype": {"name": "Epic"}
					}
				}`
				callCount := 0
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
					func(req *http.Request) (*http.Response, error) {
						callCount++
						if callCount == 1 {
							return newMockHTTPResponse(http.StatusOK, epicPayload), nil
						}
						return newMockHTTPResponse(http.StatusInternalServerError, `{"errorMessages":["boom"]}`), nil
					},
				).Twice()
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				require.Contains(t, result, "Issue: EPIC-9")
				require.NotContains(t, result, "Children (")
			},
		},
		{
			name:     "HTTP transport error",
			issueKey: "ERR-1",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(nil, errors.New("connection refused")).Once()
			},
			wantErr: "failed to fetch issue",
		},
		{
			name:     "non-200 HTTP status 404",
			issueKey: "MISSING-404",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusNotFound, `{"errorMessages":["Issue Does Not Exist"]}`), nil).Once()
			},
			wantErr: "returned status 404",
		},
		{
			name:     "invalid JSON response",
			issueKey: "BADJSON-1",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, `{not-valid-json`), nil).Once()
			},
			wantErr: "failed to parse response",
		},
		{
			name:     "empty response body",
			issueKey: "EMPTY-1",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newMockHTTPResponse(http.StatusOK, ""), nil).Once()
			},
			wantErr: "decode issue",
		},
		{
			name:     "response body read error",
			issueKey: "READERR-1",
			baseURL:  "https://jira.example.com",
			setupMocks: func(t *testing.T, client *testdata.Client) {
				t.Helper()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(&http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body: struct {
							io.Reader
							io.Closer
						}{
							Reader: io.MultiReader(strings.NewReader(strings.Repeat(" ", 1024)), errReadCloser{err: errors.New("stream broken")}),
							Closer: io.NopCloser(nil),
						},
					}, nil).Once()
			},
			wantErr: "failed to read response body",
		},
		{
			name:     "malformed base URL returns request creation error",
			issueKey: "BADURL-1",
			baseURL:  "://invalid-url",
			setupMocks: func(t *testing.T, _ *testdata.Client) {
				t.Helper()
			},
			wantErr: "failed to create request",
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
			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Return().Once()
			tt.setupMocks(t, client)

			tool := NewGetTool(cfg, p, factory)
			res, err := tool.FetchIssue(t.Context(), &JiraTaskGetParams{IssueKey: tt.issueKey})

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

func TestJiraTaskGet_ParamsMetaAndLegacy(t *testing.T) {
	t.Parallel()

	cfg := newGetTestConfig(t, "https://jira.example.com")
	var buf bytes.Buffer
	p := printer.New(&buf, false)

	factory := testdata.NewClientFactory(t)
	tool := NewGetTool(cfg, p, factory)

	require.Equal(t, "jira_task_get", tool.Name())
	require.Contains(t, tool.Description(), "Fetch a JIRA issue")

	_, err := tool.FetchIssue(t.Context(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")

	_, err = tool.Exec(t.Context(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")

	client := testdata.NewClient(t)
	factory.EXPECT().Get().Return(client).Twice()
	factory.EXPECT().Put(client).Return().Twice()
	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).RunAndReturn(
		func(_ *http.Request) (*http.Response, error) {
			return newMockHTTPResponse(http.StatusOK, `{"key":"TASK-1","fields":{"summary":"From Exec"}}`), nil
		},
	).Twice()

	execRes, err := tool.Exec(t.Context(), &JiraTaskGetParams{IssueKey: "TASK-1"})
	require.NoError(t, err)
	resp, ok := execRes.(*tools.Response)
	require.True(t, ok)
	require.Contains(t, resp.Result, "From Exec")

	legacy := New(cfg, p, factory)
	legacyRes, err := legacy.FetchIssue("TASK-1")
	require.NoError(t, err)
	require.Contains(t, legacyRes, "From Exec")
}
