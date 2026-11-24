package slack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// fakeClient answers the request without a network and records what was sent.
type fakeClient struct {
	status int
	body   string
	err    error
	seen   *http.Request
	sent   []byte
}

func (c *fakeClient) Do(req *http.Request) (*http.Response, error) {
	c.seen = req
	if req.Body != nil {
		c.sent, _ = io.ReadAll(req.Body)
	}
	if c.err != nil {
		return nil, c.err
	}
	return &http.Response{
		StatusCode: c.status,
		Body:       io.NopCloser(strings.NewReader(c.body)),
		Header:     make(http.Header),
	}, nil
}

type factory struct{ client *fakeClient }

func (f factory) Get() plugin.Client { return f.client }
func (f factory) Put(plugin.Client)  {}

type settings struct {
	enable  bool
	token   string
	allowed []string
	def     string
}

func newTool(t *testing.T, s settings, client *fakeClient) *Tool {
	t.Helper()
	cfg := testdata.NewConfig(t)
	// Maybe: a rejected call may stop before reading every key.
	cfg.On("Bool", "slack.enable").Return(s.enable).Maybe()
	cfg.On("String", "slack.token").Return(s.token).Maybe()
	cfg.On("String", "slack.default_channel_id").Return(s.def).Maybe()
	cfg.On("Strings", "slack.allowed_channel_ids").Return(s.allowed).Maybe()
	state := &plugin.State{Config: cfg, Printer: printer.GetDefault()}
	if client != nil {
		state.RequesterFactory = factory{client: client}
	}
	return New(cfg, printer.GetDefault(), state)
}

var on = settings{enable: true, token: "xoxb-test"}

func TestExec_NilParams(t *testing.T) {
	t.Parallel()
	tool := newTool(t, on, nil)
	_, err := tool.Exec(context.Background(), nil)
	require.ErrorContains(t, err, "invalid params")
}

func TestExec_Disabled(t *testing.T) {
	t.Parallel()
	tool := newTool(t, settings{}, nil)
	_, err := tool.Exec(context.Background(), &Params{Text: "hi"})
	require.ErrorContains(t, err, "integration is disabled")
}

func TestExec_MissingToken(t *testing.T) {
	t.Parallel()
	tool := newTool(t, settings{enable: true}, nil)
	_, err := tool.Exec(context.Background(), &Params{Text: "hi"})
	require.ErrorContains(t, err, "token is not configured")
}

func TestExec_MissingChannelAndText(t *testing.T) {
	t.Parallel()

	noDefault := newTool(t, on, nil)
	_, err := noDefault.Exec(context.Background(), &Params{Text: "hi"})
	require.ErrorContains(t, err, "channel is not provided")

	s := on
	s.allowed = []string{"C1"}
	noText := newTool(t, s, nil)
	_, err = noText.Exec(context.Background(), &Params{Channel: "C1"})
	require.ErrorContains(t, err, "text is required")
}

func TestExec_AllowedChannels(t *testing.T) {
	t.Parallel()

	s := on
	s.allowed = []string{"C1"}
	tool := newTool(t, s, nil)
	_, err := tool.Exec(context.Background(), &Params{Channel: "C2", Text: "hi"})
	require.ErrorContains(t, err, "not in the allowed list")
}

func TestExec_SendsSlackShape(t *testing.T) {
	t.Parallel()

	client := &fakeClient{status: http.StatusOK, body: `{"ok":true,"channel":"C42","ts":"1700000000.000100"}`}
	s := on
	s.def = "C42"
	tool := newTool(t, s, client)

	result, err := tool.Exec(context.Background(), &Params{Text: "*hello*", ThreadTS: "1.2", UnfurlLinks: boolPtr(false)})
	require.NoError(t, err)

	require.Equal(t, "https://slack.com/api/chat.postMessage", client.seen.URL.String())
	require.Equal(t, http.MethodPost, client.seen.Method)
	require.Equal(t, "Bearer xoxb-test", client.seen.Header.Get("Authorization"))
	require.Equal(t, "application/json; charset=utf-8", client.seen.Header.Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(client.sent, &body))
	require.Equal(t, "C42", body["channel"])
	require.Equal(t, "*hello*", body["text"])
	require.Equal(t, true, body["mrkdwn"])
	require.Equal(t, "1.2", body["thread_ts"])
	require.Equal(t, false, body["unfurl_links"])
	require.NotContains(t, body, "unfurl_media")

	response, ok := result.(*tools.Response)
	require.True(t, ok)
	require.Equal(t, tools.StatusOK, response.Status)
	require.Equal(t, "Message sent successfully to C42 (ts 1700000000.000100)", response.Result)
}

func TestExec_SlackErrorInBody(t *testing.T) {
	t.Parallel()

	client := &fakeClient{status: http.StatusOK, body: `{"ok":false,"error":"channel_not_found"}`}
	tool := newTool(t, on, client)
	_, err := tool.Exec(context.Background(), &Params{Channel: "C1", Text: "hi"})
	require.ErrorContains(t, err, "slack: API error: channel_not_found")
}

func TestExec_HTTPStatusError(t *testing.T) {
	t.Parallel()

	client := &fakeClient{status: http.StatusTooManyRequests, body: "slow down"}
	tool := newTool(t, on, client)
	_, err := tool.Exec(context.Background(), &Params{Channel: "C1", Text: "hi"})
	require.ErrorContains(t, err, "status 429: slow down")
}

func TestExec_RejectsOverlongText(t *testing.T) {
	t.Parallel()

	client := &fakeClient{status: http.StatusOK, body: `{"ok":true}`}
	tool := newTool(t, on, client)
	_, err := tool.Exec(context.Background(), &Params{Channel: "C1", Text: strings.Repeat("x", maxTextLength+1)})
	require.ErrorContains(t, err, "longer than")
	require.Nil(t, client.seen)
}

func boolPtr(v bool) *bool { return &v }
