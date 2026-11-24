package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const slackDefaultTimeout = 30 * time.Second

// Slack rejects messages above 40,000 characters; stop well before that.
const maxTextLength = 39_000

// apiURL is a variable so tests can point the tool at a local server.
var apiURL = "https://slack.com/api/chat.postMessage"

// Params represents parameters for the slack tool.
type Params struct {
	Channel     string `json:"channel,omitempty" jsonschema_description:"Slack channel ID or name (#general or C01234567). If not provided, uses default_channel_id from config."` //nolint:lll // ok.
	Text        string `json:"text" jsonschema_description:"Message text in Slack markdown (*bold*, _italic_, <https://example.com|link>, 'code')."`
	ThreadTS    string `json:"thread_ts,omitempty" jsonschema_description:"Reply in this thread (the ts of the parent message)."` //nolint:lll // ok.
	UnfurlLinks *bool  `json:"unfurl_links,omitempty" jsonschema_description:"Let Slack expand link previews for this message."`  //nolint:lll // ok.
	UnfurlMedia *bool  `json:"unfurl_media,omitempty" jsonschema_description:"Let Slack expand media previews for this message."` //nolint:lll // ok.
}

// Tool provides slack send operation as LLM tool.
type Tool struct {
	cfg     plugin.Config
	printer printer.Printer
	state   *plugin.State
}

// New creates a new Tool instance.
func New(cfg plugin.Config, p printer.Printer, state *plugin.State) *Tool {
	return &Tool{cfg: cfg, printer: p, state: state}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "slack"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Send a message to a Slack channel using Slack's own markdown. " +
		"Requires slack.enable=true and slack.token; optional: slack.default_channel_id, slack.allowed_channel_ids."
}

type apiResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

// Exec executes the slack tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	ctx, cancel := context.WithTimeout(ctx, slackDefaultTimeout)
	defer cancel()

	if !t.cfg.Bool("slack.enable") {
		return nil, errors.New("slack: integration is disabled. Set slack.enable=true in config")
	}

	token := t.cfg.String("slack.token")
	if token == "" {
		return nil, errors.New("slack: token is not configured")
	}

	channel := params.Channel
	if channel == "" {
		channel = t.cfg.String("slack.default_channel_id")
		if channel == "" {
			return nil, errors.New("slack: channel is not provided and default_channel_id is not configured")
		}
	}

	if allowed := t.cfg.Strings("slack.allowed_channel_ids"); len(allowed) > 0 && !slices.Contains(allowed, channel) {
		return nil, errors.New("slack: channel is not in the allowed list")
	}

	if params.Text == "" {
		return nil, errors.New("slack: text is required")
	}
	if len(params.Text) > maxTextLength {
		return nil, fmt.Errorf("slack: text is longer than %d characters", maxTextLength)
	}

	t.printer.ToolCall(printer.IconRobot, "slack", "channel", channel)

	body := map[string]any{"channel": channel, "text": params.Text, "mrkdwn": true}
	if params.ThreadTS != "" {
		body["thread_ts"] = params.ThreadTS
	}
	if params.UnfurlLinks != nil {
		body["unfurl_links"] = *params.UnfurlLinks
	}
	if params.UnfurlMedia != nil {
		body["unfurl_media"] = *params.UnfurlMedia
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("slack: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("slack: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	client := t.state.RequesterFactory.Get()
	defer t.state.RequesterFactory.Put(client)

	resp, err := client.Do(req)
	if err != nil {
		t.printer.Error("slack: failed to send message", "err", err)
		return nil, fmt.Errorf("slack: failed to send message: %w", err)
	}
	defer resp.Body.Close()

	// Slack answers 200 even for a rejected message, so the body decides.
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slack: API returned status %d: %s", resp.StatusCode, string(raw))
	}
	var result apiResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("slack: failed to parse response: %w", err)
	}
	if !result.OK {
		message := result.Error
		if message == "" {
			message = "unknown error"
		}
		return nil, fmt.Errorf("slack: API error: %s", message)
	}

	t.printer.ToolCall(printer.IconDone, "slack: message sent successfully", "channel", channel)

	where := result.Channel
	if where == "" {
		where = channel
	}
	ts := ""
	if result.TS != "" {
		ts = fmt.Sprintf(" (ts %s)", result.TS)
	}
	return &tools.Response{
		Status: tools.StatusOK,
		Result: fmt.Sprintf("Message sent successfully to %s%s", where, ts),
	}, nil
}
