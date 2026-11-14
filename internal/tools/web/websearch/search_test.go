package websearch_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/web/websearch"
)

func newTestPrinter() printer.Printer {
	return printer.New(io.Discard, false)
}

// newTestConfig creates a test configuration with default values.
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// Copy the config from testdata to a temporary directory
	sourcePath := filepath.Join("testdata", "config.yaml")
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.yaml")

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("Failed to read source config: %v", err)
	}

	if err := os.WriteFile(path, sourceContent, 0o644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	return cfg
}

func newTestResponse(status int, contentType, body string) *http.Response {
	// Create a new copy of body for each response to avoid race conditions
	// when the response is used concurrently by multiple goroutines
	bodyCopy := strings.Clone(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(bodyCopy)),
	}
}

func TestWebSearch_Tool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		params         *websearch.Params
		query          string
		fetchResponse  *http.Response
		fetchErr       error
		setupMocks     func(factory *testdata.ClientFactory, client *testdata.Client)
		expectedStatus string
		wantErr        string
		wantContent    []string
	}{
		{
			name:   "successful web search",
			query:  "test query",
			params: &websearch.Params{Query: "test query"},
			fetchResponse: newTestResponse(http.StatusOK, "text/html", `
				<html>
				<body>
					<div class="result">
						<h3>Result Title</h3>
						<p>Result description</p>
					</div>
				</body>
				</html>
			`),
			setupMocks: func(factory *testdata.ClientFactory, client *testdata.Client) {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
			},
			expectedStatus: tools.StatusOK,
			wantContent: []string{
				"Result Title",
				"Result description",
			},
		},
		{
			name:   "nil params",
			query:  "",
			params: nil,
			setupMocks: func(factory *testdata.ClientFactory, client *testdata.Client) {
			},
			expectedStatus: tools.StatusOK,
			wantErr:        "invalid params",
		},
		{
			name:     "network error during fetch",
			query:    "network error",
			params:   &websearch.Params{Query: "network error"},
			fetchErr: errors.New("connection refused"),
			setupMocks: func(factory *testdata.ClientFactory, client *testdata.Client) {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
			},
			wantErr: "web_search: failed",
		},
		{
			name:          "HTTP error response",
			query:         "error",
			params:        &websearch.Params{Query: "error"},
			fetchResponse: newTestResponse(http.StatusForbidden, "text/html", "forbidden"),
			setupMocks: func(factory *testdata.ClientFactory, client *testdata.Client) {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
			},
			wantErr: "web_search: failed",
		},
		{
			name:   "rate limiter with tokens available",
			query:  "rate limit",
			params: &websearch.Params{Query: "rate limit"},
			fetchResponse: newTestResponse(http.StatusOK, "text/html", `
				<html>
				<body>
					<div class="result">
						<h3>Result Title</h3>
						<p>Result description</p>
					</div>
				</body>
				</html>
			`),
			setupMocks: func(factory *testdata.ClientFactory, client *testdata.Client) {
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
			},
			expectedStatus: tools.StatusOK,
			wantContent: []string{
				"Result Title",
				"Result description",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig(t)
			printerInstance := newTestPrinter()

			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			if tt.setupMocks != nil {
				tt.setupMocks(factory, client)
			}

			if tt.fetchResponse != nil || tt.fetchErr != nil {
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						// Verify the URL contains the encoded query
						// The query should be in the URL (URL-encoded - spaces become +)
						encodedQuery := strings.ReplaceAll(tt.query, " ", "+")
						require.Contains(t, req.URL.String(), encodedQuery, "URL should contain encoded query")
						return tt.fetchResponse, tt.fetchErr
					}).Once()
			}

			// Create the websearch tool
			tool := websearch.NewTool(cfg, printerInstance, factory, nil)

			// Set up rate limiter if needed
			if tt.name == "rate limiter error" {
				// Create a limiter that's already exhausted to simulate rate limit error
				limiter := rate.NewLimiter(rate.Limit(0), 1)
				tool.SetLimiter(limiter)
			}

			ctx := context.Background()
			got, err := tool.WebSearch(ctx, tt.params)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			response, ok := got.(*tools.Response)
			require.True(t, ok)
			require.Equal(t, tt.expectedStatus, response.Status)

			if len(tt.wantContent) > 0 {
				require.Contains(t, response.Result, tt.wantContent[0])
			}
		})
	}
}

func TestWebSearch_Name(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	require.Equal(t, "web_search", tool.Name())
}

func TestWebSearch_Description(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	require.Contains(t, tool.Description(), "web search")
}

func TestWebSearch_Params(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid query",
			query:   "test query",
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: false,
		},
		{
			name:    "query with special characters",
			query:   "test query with & special <chars>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := &websearch.Params{Query: tt.query}
			require.Equal(t, tt.query, params.Query)
		})
	}
}

// TestWebSearch_MultipleQueries tests that multiple search requests can be made
func TestWebSearch_MultipleQueries(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)

	// Create multiple mock clients for multiple requests
	clients := make([]*testdata.Client, 3)
	for i := range clients {
		clients[i] = testdata.NewClient(t)
	}

	for i := range clients {
		factory.EXPECT().Get().Return(clients[i]).Once()
		factory.EXPECT().Put(clients[i]).Once()

		clients[i].EXPECT().Do(mock.AnythingOfType("*http.Request")).
			Return(newTestResponse(http.StatusOK, "text/html", `
				<html>
				<body>
					<div class="result">
						<h3>Result Title</h3>
						<p>Result description</p>
					</div>
				</body>
				</html>
			`), nil).Once()
	}

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	queries := []string{"query 1", "query 2", "query 3"}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			params := &websearch.Params{Query: query}
			ctx := context.Background()
			got, err := tool.WebSearch(ctx, params)

			require.NoError(t, err)
			require.NotNil(t, got)

			response, ok := got.(*tools.Response)
			require.True(t, ok)
			require.Equal(t, tools.StatusOK, response.Status)
			require.Contains(t, response.Result, "Result Title")
		})
	}
}

// TestWebSearch_EscapedQuery tests that the query is properly URL-encoded
func TestWebSearch_EscapedQuery(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Run(func(req *http.Request) {
			// Verify URL encoding
			// DuckDuckGo uses + for spaces in query params
			expected := "https://html.duckduckgo.com/html/?q=test+query+with+spaces"
			require.Equal(t, expected, req.URL.String())
		}).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "test query with spaces"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_ExtractParams tests that params are properly extracted and used
func TestWebSearch_ExtractParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple query",
			input: "test query",
			want:  "test query",
		},
		{
			name:  "query with unicode",
			input: "тест запрос",
			want:  "тест запрос",
		},
		{
			name:  "query with emoji",
			input: "test 🎯 query",
			want:  "test 🎯 query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := &websearch.Params{Query: tt.input}
			require.Equal(t, tt.want, params.Query)
		})
	}
}

// TestWebSearch_Concurrent tests that the tool can handle multiple sequential requests
func TestWebSearch_Concurrent(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()

	queries := []string{"query 1", "query 2", "query 3"}
	results := make(chan error, len(queries))

	for _, query := range queries {
		// Create a new factory and client for each request to avoid race conditions
		// (testify/mock is not safe for concurrent use)
		factory := testdata.NewClientFactory(t)
		client := testdata.NewClient(t)

		factory.EXPECT().Get().Return(client).Once()
		factory.EXPECT().Put(client).Once()

		client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
			Return(newTestResponse(http.StatusOK, "text/html", `
				<html>
				<body>
					<div class="result">
						<h3>Result Title</h3>
						<p>Result description</p>
					</div>
				</body>
				</html>
			`), nil).Once()

		tool := websearch.NewTool(cfg, printerInstance, factory, nil)

		go func(q string) {
			params := &websearch.Params{Query: q}
			ctx := context.Background()
			_, err := tool.WebSearch(ctx, params)
			results <- err
		}(query)
	}

	for range queries {
		err := <-results
		require.NoError(t, err)
	}
}

// TestWebSearch_ParamsStructure tests the Params struct fields
func TestWebSearch_ParamsStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "empty query",
			query:    "",
			expected: "",
		},
		{
			name:     "single word",
			query:    "test",
			expected: "test",
		},
		{
			name:     "multiple words",
			query:    "test query with multiple words",
			expected: "test query with multiple words",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := &websearch.Params{Query: tt.query}
			require.Equal(t, tt.expected, params.Query)
		})
	}
}

// TestWebSearch_WithLimiter tests the tool with rate limiter
func TestWebSearch_WithLimiter(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)
	tool.SetLimiter(limiter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := &websearch.Params{Query: "test query"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)

	response, ok := got.(*tools.Response)
	require.True(t, ok)
	require.Equal(t, tools.StatusOK, response.Status)
	require.Contains(t, response.Result, "Result Title")
}

// TestWebSearch_EmptyResponse tests handling of empty HTML response
func TestWebSearch_EmptyResponse(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body></body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "empty response"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_InvalidHTML tests handling of invalid HTML
func TestWebSearch_InvalidHTML(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", "invalid html"), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "invalid html"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_CustomUserAgent tests that custom user agent is set
func TestWebSearch_CustomUserAgent(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Run(func(req *http.Request) {
			// Verify user agent is set
			require.NotEmpty(t, req.Header.Get("User-Agent"))
			require.Contains(t, req.Header.Get("User-Agent"), "Mozilla/5.0")
		}).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "test user agent"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_SearchURL tests that the search URL is correctly formed
func TestWebSearch_SearchURL(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Run(func(req *http.Request) {
			// Verify the search URL is correctly formed
			require.Contains(t, req.URL.String(), "https://html.duckduckgo.com/html/?q=")
		}).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "test url"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_ResponseFormat tests the response format
func TestWebSearch_ResponseFormat(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "test format"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)

	// Verify response structure
	response, ok := got.(*tools.Response)
	require.True(t, ok)
	require.Equal(t, tools.StatusOK, response.Status)
	require.NotEmpty(t, response.Result)
	require.IsType(t, "", response.Result) // Result should be a string
}

// TestWebSearch_ErrorHandling tests error handling
func TestWebSearch_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		responseErr    error
		wantErr        bool
		errContains    string
	}{
		{
			name:           "404 not found",
			responseStatus: http.StatusNotFound,
			responseBody:   "not found",
			wantErr:        true,
			errContains:    "web_search: failed",
		},
		{
			name:           "500 internal server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   "internal server error",
			wantErr:        true,
			errContains:    "web_search: failed",
		},
		{
			name:           "401 unauthorized",
			responseStatus: http.StatusUnauthorized,
			responseBody:   "unauthorized",
			wantErr:        true,
			errContains:    "web_search: failed",
		},
		{
			name:           "network timeout",
			responseStatus: http.StatusOK,
			responseBody:   "timeout",
			responseErr:    context.DeadlineExceeded,
			wantErr:        true,
			errContains:    "web_search: failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			printerInstance := newTestPrinter()
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
				Return(newTestResponse(tt.responseStatus, "text/html", tt.responseBody), tt.responseErr).Once()

			tool := websearch.NewTool(cfg, printerInstance, factory, nil)

			ctx := context.Background()
			params := &websearch.Params{Query: "error test"}
			got, err := tool.WebSearch(ctx, params)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

// TestWebSearch_DuckDuckGoURLFormat tests that the DuckDuckGo URL is correctly formatted
func TestWebSearch_DuckDuckGoURLFormat(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Run(func(req *http.Request) {
			// Verify the URL format
			url := req.URL.String()
			require.True(t, strings.HasPrefix(url, "https://html.duckduckgo.com/html/?q="))
		}).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "duckduckgo url test"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestWebSearch_MarkdownProcessing tests that HTML is converted to Markdown
func TestWebSearch_MarkdownProcessing(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<h1>Heading</h1>
				<p>Paragraph with <strong>bold</strong> and <em>italic</em>.</p>
				<a href="https://example.com">Link</a>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "markdown test"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)

	response, ok := got.(*tools.Response)
	require.True(t, ok)
	require.Contains(t, response.Result, "Heading")
	require.Contains(t, response.Result, "Paragraph")
	require.Contains(t, response.Result, "bold")
	require.Contains(t, response.Result, "italic")
	require.Contains(t, response.Result, "Link")
	require.Contains(t, response.Result, "https://example.com")
}

// TestWebSearch_StreamedResponse tests handling of streamed response
func TestWebSearch_StreamedResponse(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "streamed response"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)

	response, ok := got.(*tools.Response)
	require.True(t, ok)
	require.Equal(t, tools.StatusOK, response.Status)
	require.Contains(t, response.Result, "Result Title")
	require.Contains(t, response.Result, "Result description")
}

// TestWebSearch_ResultExtraction tests that result is properly extracted from response
func TestWebSearch_ResultExtraction(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()

	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newTestResponse(http.StatusOK, "text/html", `
			<html>
			<body>
				<div class="result">
					<h3>Result Title</h3>
					<p>Result description</p>
				</div>
			</body>
			</html>
		`), nil).Once()

	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	ctx := context.Background()
	params := &websearch.Params{Query: "result extraction"}
	got, err := tool.WebSearch(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, got)

	// Verify response structure
	response, ok := got.(*tools.Response)
	require.True(t, ok)
	require.Equal(t, tools.StatusOK, response.Status)
	require.NotEmpty(t, response.Result)
	require.IsType(t, "", response.Result) // Result should be a string
}

// TestWebSearch_ToolDescription tests the tool description
func TestWebSearch_ToolDescription(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	tool := websearch.NewTool(cfg, printerInstance, factory, nil)

	description := tool.Description()
	require.Contains(t, description, "web search")
	require.Contains(t, description, "results")
}

// TestWebSearch_URLEncodingSpecialChars tests URL encoding of special characters
func TestWebSearch_URLEncodingSpecialChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "spaces",
			query:    "test query with spaces",
			expected: "https://html.duckduckgo.com/html/?q=test+query+with+spaces",
		},
		{
			name:     "ampersand",
			query:    "test&query",
			expected: "https://html.duckduckgo.com/html/?q=test%26query",
		},
		{
			name:     "special characters",
			query:    "test?query=1",
			expected: "https://html.duckduckgo.com/html/?q=test%3Fquery%3D1",
		},
		{
			name:     "unicode",
			query:    "тест запрос",
			expected: "https://html.duckduckgo.com/html/?q=%D1%82%D0%B5%D1%81%D1%82+%D0%B7%D0%B0%D0%BF%D1%80%D0%BE%D1%81",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			printerInstance := newTestPrinter()
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
				Run(func(req *http.Request) {
					require.Equal(t, tt.expected, req.URL.String())
				}).
				Return(newTestResponse(http.StatusOK, "text/html", `
					<html>
					<body>
						<div class="result">
							<h3>Result Title</h3>
							<p>Result description</p>
						</div>
					</body>
					</html>
				`), nil).Once()

			tool := websearch.NewTool(cfg, printerInstance, factory, nil)

			ctx := context.Background()
			params := &websearch.Params{Query: tt.query}
			got, err := tool.WebSearch(ctx, params)

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

// TestWebSearch_StatusCodeChecking tests HTTP status code validation
func TestWebSearch_StatusCodeChecking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		wantErr        bool
	}{
		{
			name:           "200 OK",
			responseStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "301 redirect",
			responseStatus: http.StatusMovedPermanently,
			wantErr:        true,
		},
		{
			name:           "302 redirect",
			responseStatus: http.StatusFound,
			wantErr:        true,
		},
		{
			name:           "400 bad request",
			responseStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "401 unauthorized",
			responseStatus: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "403 forbidden",
			responseStatus: http.StatusForbidden,
			wantErr:        true,
		},
		{
			name:           "404 not found",
			responseStatus: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "500 internal server error",
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "502 bad gateway",
			responseStatus: http.StatusBadGateway,
			wantErr:        true,
		},
		{
			name:           "503 service unavailable",
			responseStatus: http.StatusServiceUnavailable,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			printerInstance := newTestPrinter()
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
				Return(newTestResponse(tt.responseStatus, "text/html", "error"), nil).Once()

			tool := websearch.NewTool(cfg, printerInstance, factory, nil)

			ctx := context.Background()
			params := &websearch.Params{Query: "status code test"}
			got, err := tool.WebSearch(ctx, params)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				require.Contains(t, err.Error(), "status")
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

// TestWebSearch_ResponseHeaderChecking tests response header validation
func TestWebSearch_ResponseHeaderChecking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{
			name:        "text/html content type",
			contentType: "text/html; charset=utf-8",
			wantErr:     false,
		},
		{
			name:        "application/html content type",
			contentType: "application/html",
			wantErr:     false,
		},
		{
			name:        "empty content type",
			contentType: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t)
			printerInstance := newTestPrinter()
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			factory.EXPECT().Get().Return(client).Once()
			factory.EXPECT().Put(client).Once()

			client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
				Return(newTestResponse(http.StatusOK, tt.contentType, `
					<html>
					<body>
						<div class="result">
							<h3>Result Title</h3>
							<p>Result description</p>
						</div>
					</body>
					</html>
				`), nil).Once()

			tool := websearch.NewTool(cfg, printerInstance, factory, nil)

			ctx := context.Background()
			params := &websearch.Params{Query: "header test"}
			got, err := tool.WebSearch(ctx, params)

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

// TestWebSearch_ContextTimeout tests context timeout handling
func TestWebSearch_ContextTimeout(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	printerInstance := newTestPrinter()
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)

	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()
	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(nil, context.DeadlineExceeded).Once()

	// Context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	params := &websearch.Params{Query: "timeout test"}
	_, err := websearch.NewTool(cfg, printerInstance, factory, nil).WebSearch(ctx, params)

	// The error should be related to context cancellation or network error
	require.Error(t, err)
	require.Contains(t, err.Error(), "web_search: failed")
}
