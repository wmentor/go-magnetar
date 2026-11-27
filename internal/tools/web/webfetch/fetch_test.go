package webfetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type fakeReadCodec struct {
	out string
	err error
}

func (f fakeReadCodec) ReadFile(filename string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.out, nil
}

func newHTTPResponse(status int, contentType, body string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	if contentType != "" {
		resp.Header.Set("Content-Type", contentType)
	}
	return resp
}

func TestToolMetadata(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	factory := testdata.NewClientFactory(t)
	p := printer.New(io.Discard, false)
	tool := NewTool(cfg, p, factory, &plugin.State{})

	require.NotNil(t, tool)
	assert.Equal(t, "web_fetch", tool.Name())
	assert.Equal(t, "web_fetch", tool.NameMCP())
	assert.Contains(t, tool.Description(), "Markdown content")
}

func TestWebFetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		params      *Params
		setupMocks  func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client)
		wantErr     string
		wantContain []string
	}{
		{
			name:    "nil params",
			params:  nil,
			wantErr: "invalid params",
		},
		{
			name:   "handled by web processor",
			params: &Params{URL: "https://example.com/issue/1"},
			setupMocks: func(store *testdata.WebProcessorStore, _ *testdata.ClientFactory, _ *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/issue/1").
					Return("# Issue Title", nil).Once()
			},
			wantContain: []string{"# Issue Title"},
		},
		{
			name:   "web processor returns non-fallback error",
			params: &Params{URL: "https://example.com/issue/2"},
			setupMocks: func(store *testdata.WebProcessorStore, _ *testdata.ClientFactory, _ *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/issue/2").
					Return("", errors.New("api failure")).Once()
			},
			wantErr: "web fetch enigine",
		},
		{
			name:   "URL without scheme gets https prefix and HTML response",
			params: &Params{URL: "example.com/page"},
			setupMocks: func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/page").
					Return("", plugin.ErrNoWebProcessor).Once()
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					RunAndReturn(func(req *http.Request) (*http.Response, error) {
						require.Equal(t, "https://example.com/page", req.URL.String())
						return newHTTPResponse(
							http.StatusOK,
							"text/html; charset=utf-8",
							"<html><body><h1>Hello</h1><p>World</p></body></html>",
						), nil
					}).Once()
			},
			wantContain: []string{"Hello", "World"},
		},
		{
			name:   "plain text response",
			params: &Params{URL: "https://example.com/notes.txt"},
			setupMocks: func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/notes.txt").
					Return("", plugin.ErrNoWebProcessor).Once()
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newHTTPResponse(http.StatusOK, "text/plain; charset=utf-8", "plain body text"), nil).Once()
			},
			wantContain: []string{"plain body text"},
		},
		{
			name:   "RSS XML feed response",
			params: &Params{URL: "https://example.com/feed.xml"},
			setupMocks: func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/feed.xml").
					Return("", plugin.ErrNoWebProcessor).Once()
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
				rssBody := `<?xml version="1.0" encoding="UTF-8"?>` +
					`<rss version="2.0"><channel><title>Feed</title>` +
					`<item><title>First Post</title><link>https://example.com/1</link></item>` +
					`</channel></rss>`
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newHTTPResponse(http.StatusOK, "application/rss+xml", rssBody), nil).Once()
			},
			wantContain: []string{"First Post", "https://example.com/1"},
		},
		{
			name:   "HTTP 404 error",
			params: &Params{URL: "https://example.com/missing"},
			setupMocks: func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/missing").
					Return("", plugin.ErrNoWebProcessor).Once()
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return(newHTTPResponse(http.StatusNotFound, "text/plain", "not found"), nil).Once()
			},
			wantErr: "failed to fetch URL",
		},
		{
			name:   "HTTP transport error",
			params: &Params{URL: "https://example.com/down"},
			setupMocks: func(store *testdata.WebProcessorStore, factory *testdata.ClientFactory, client *testdata.Client) {
				store.EXPECT().Process(mock.Anything, "https://example.com/down").
					Return("", plugin.ErrNoWebProcessor).Once()
				factory.EXPECT().Get().Return(client).Once()
				factory.EXPECT().Put(client).Once()
				client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
					Return((*http.Response)(nil), errors.New("connection refused")).Once()
			},
			wantErr: "failed to fetch URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)
			store := testdata.NewWebProcessorStore(t)
			if tt.setupMocks != nil {
				tt.setupMocks(store, factory, client)
			}

			state := &plugin.State{
				WebProcStore: store,
			}
			tool := NewTool(cfg, printer.New(io.Discard, false), factory, state)

			got, err := tool.WebFetch(context.Background(), tt.params)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			resp, ok := got.(*tools.Response)
			require.True(t, ok)
			assert.Equal(t, tools.StatusOK, resp.Status)
			for _, want := range tt.wantContain {
				assert.Contains(t, resp.Result, want)
			}
		})
	}
}

func TestWebFetchEngineSearchModeError(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	factory := testdata.NewClientFactory(t)
	client := testdata.NewClient(t)
	store := testdata.NewWebProcessorStore(t)

	store.EXPECT().Process(mock.Anything, "https://example.com/fail").
		Return("", plugin.ErrNoWebProcessor).Once()
	factory.EXPECT().Get().Return(client).Once()
	factory.EXPECT().Put(client).Once()
	client.EXPECT().Do(mock.AnythingOfType("*http.Request")).
		Return(newHTTPResponse(http.StatusInternalServerError, "text/plain", "err"), nil).Once()

	tool := NewTool(cfg, printer.New(io.Discard, false), factory, &plugin.State{WebProcStore: store})
	got, err := tool.WebFetchEngine(context.Background(), "https://example.com/fail", false)
	require.EqualError(t, err, "web_search: failed")
	assert.Empty(t, got)
}

func TestMimeCodecWork(t *testing.T) {
	t.Parallel()

	tool := NewTool(testdata.NewConfig(t), printer.New(io.Discard, false), testdata.NewClientFactory(t), &plugin.State{})

	tests := []struct {
		name    string
		codec   codec.ReadCodec
		content []byte
		want    string
		wantErr string
	}{
		{
			name:    "nil codec",
			codec:   nil,
			content: []byte("abc"),
			wantErr: "codec is nil",
		},
		{
			name:    "codec succeeds",
			codec:   fakeReadCodec{out: "decoded markdown"},
			content: []byte("raw binary"),
			want:    "decoded markdown",
		},
		{
			name:    "codec returns error",
			codec:   fakeReadCodec{err: errors.New("corrupt data")},
			content: []byte("bad"),
			wantErr: "corrupt data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tool.mimeCodecWork(tt.codec, tt.content)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebFetchString(t *testing.T) {
	t.Parallel()

	t.Run("invalid json arguments", func(t *testing.T) {
		t.Parallel()
		tool := NewTool(testdata.NewConfig(t), printer.New(io.Discard, false), testdata.NewClientFactory(t), &plugin.State{})
		got, err := tool.WebFetchString("{bad json")
		require.NoError(t, err)
		assert.Contains(t, got, "error: failed to parse arguments")
	})

	t.Run("fetch error returned in string", func(t *testing.T) {
		t.Parallel()
		store := testdata.NewWebProcessorStore(t)
		store.EXPECT().Process(mock.Anything, "https://example.com/err").
			Return("", errors.New("boom")).Once()
		tool := NewTool(testdata.NewConfig(t), printer.New(io.Discard, false), testdata.NewClientFactory(t), &plugin.State{WebProcStore: store})
		got, err := tool.WebFetchString(`{"url":"https://example.com/err"}`)
		require.NoError(t, err)
		assert.Contains(t, got, "error:")
		assert.Contains(t, got, "boom")
	})

	t.Run("valid fetch stringified", func(t *testing.T) {
		t.Parallel()
		store := testdata.NewWebProcessorStore(t)
		store.EXPECT().Process(mock.Anything, "https://example.com/ok").
			Return("fetched content", nil).Once()
		tool := NewTool(testdata.NewConfig(t), printer.New(io.Discard, false), testdata.NewClientFactory(t), &plugin.State{WebProcStore: store})
		got, err := tool.WebFetchString(`{"url":"https://example.com/ok"}`)
		require.NoError(t, err)
		assert.Contains(t, got, "fetched content")
	})
}
