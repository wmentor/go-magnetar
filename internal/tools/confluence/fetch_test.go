package confluence_test

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func newFetchTestConfig(t *testing.T, baseURL string) *config.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	content := fmt.Sprintf(
		"version: \"1.0\"\nconfluence:\n  base_url: %q\n  api_key: \"test-token\"\n",
		baseURL,
	)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	return cfg
}

func newFetchTestResponse(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchTool_FetchPage(t *testing.T) {
	t.Parallel()

	const pageBody = `{
		"id": "12345",
		"title": "Test Page",
		"space": {"key": "TEAM"},
		"version": {"number": 7, "when": "2026-01-02T03:04:05.000Z"},
		"body": {"storage": {"value": "<p>Page content</p>"}},
		"history": {
			"createdBy": {"username": "alice", "displayName": "Alice Example"},
			"createdDate": "2025-12-31T12:00:00.000Z"
		}
	}`

	tests := []struct {
		name        string
		baseURL     string
		params      *confluence.FetchPageParams
		response    *http.Response
		requestErr  error
		doRequest   bool
		wantPageID  string
		wantErr     string
		wantContent []string
	}{
		{
			name:       "numeric page ID parses and formats response",
			baseURL:    "https://confluence.example.com",
			params:     &confluence.FetchPageParams{PageID: "12345"},
			response:   newFetchTestResponse(http.StatusOK, "application/json", pageBody),
			doRequest:  true,
			wantPageID: "12345",
			wantContent: []string{
				"Confluence page ID: 12345",
				"Space: TEAM",
				"Title: Test Page",
				"Created: 2025-12-31T12:00:00.000Z",
				"Author: Alice Example (alice)",
				"Updated: 2026-01-02T03:04:05.000Z",
				"Version: 7",
				"Body:\n<p>Page content</p>",
			},
		},
		{
			name:       "short page ID resolves and fetches numeric page",
			baseURL:    "https://confluence.example.com",
			params:     &confluence.FetchPageParams{PageID: "A4HhC", IsShort: true},
			response:   newFetchTestResponse(http.StatusOK, "application/json", pageBody),
			doRequest:  true,
			wantPageID: "148996355",
			wantContent: []string{
				"Confluence page ID: 12345",
				"Title: Test Page",
				"Body:\n<p>Page content</p>",
			},
		},
		{
			name:    "invalid short page ID",
			baseURL: "https://confluence.example.com",
			params:  &confluence.FetchPageParams{PageID: "!!!@@@", IsShort: true},
			wantErr: "failed to resolve short page ID",
		},
		{
			name:       "network error",
			baseURL:    "https://confluence.example.com",
			params:     &confluence.FetchPageParams{PageID: "12345"},
			requestErr: errors.New("connection refused"),
			doRequest:  true,
			wantPageID: "12345",
			wantErr:    "connection refused",
		},
		{
			name:       "HTTP error response",
			baseURL:    "https://confluence.example.com",
			params:     &confluence.FetchPageParams{PageID: "12345"},
			response:   newFetchTestResponse(http.StatusForbidden, "application/json", `{"message":"forbidden"}`),
			doRequest:  true,
			wantPageID: "12345",
			wantErr:    "returned status 403",
		},
		{
			name:       "invalid JSON response",
			baseURL:    "https://confluence.example.com",
			params:     &confluence.FetchPageParams{PageID: "12345"},
			response:   newFetchTestResponse(http.StatusOK, "application/json", "{"),
			doRequest:  true,
			wantPageID: "12345",
			wantErr:    "failed to parse response",
		},
		{
			name:    "response body read error",
			baseURL: "https://confluence.example.com",
			params:  &confluence.FetchPageParams{PageID: "12345"},
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: &failingReadCloser{
					data: []byte(`{"body":{"storage":{"value":"` + strings.Repeat("x", 2048) + `"}}}`),
					err:  errors.New("response read failed"),
				},
			},
			doRequest:  true,
			wantPageID: "12345",
			wantErr:    "failed to read response body",
		},
		{
			name:    "response charset detection read error",
			baseURL: "https://confluence.example.com",
			params:  &confluence.FetchPageParams{PageID: "12345"},
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       &failingReadCloser{err: errors.New("response read failed")},
			},
			doRequest:  true,
			wantPageID: "12345",
			wantErr:    "decode page",
		},
		{
			name:    "invalid request URL",
			baseURL: "https://[::1",
			params:  &confluence.FetchPageParams{PageID: "12345"},
			wantErr: "failed to create request",
		},
		{
			name:    "nil params",
			baseURL: "https://confluence.example.com",
			wantErr: "invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			usesClient := tt.params != nil && (tt.doRequest || tt.wantErr == "failed to create request")
			if usesClient {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
			}
			if tt.doRequest {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, http.MethodGet, req.Method)
						require.Equal(
							t,
							fmt.Sprintf(
								"%s/rest/api/content/%s?expand=body.storage,space,version,history.createdBy",
								tt.baseURL,
								tt.wantPageID,
							),
							req.URL.String(),
						)
						require.NotEmpty(t, req.Header.Get("Authorization"))
						require.Equal(t, "application/json", req.Header.Get("Content-Type"))
						return tt.response, tt.requestErr
					}).Once()
			}

			tool := confluence.NewFetchTool(
				newFetchTestConfig(t, tt.baseURL),
				printer.GetDefault(),
				factory,
			)

			got, err := tool.FetchPage(context.Background(), tt.params)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			response, ok := got.(*tools.Response)
			require.True(t, ok)
			require.Equal(t, tools.StatusOK, response.Status)
			for _, content := range tt.wantContent {
				require.Contains(t, response.Result, content)
			}
		})
	}
}

type failingReadCloser struct {
	data []byte
	err  error
}

func (r *failingReadCloser) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, r.err
}

func (r *failingReadCloser) Close() error {
	return nil
}
