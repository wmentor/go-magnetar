package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/types"
)

const telegramDefaultTimeout = 30 * time.Second

// Params represents parameters for the telegram tool.
type Params struct {
	ChatID              *types.FlexibleString `json:"chat_id,omitempty" jsonschema_description:"Telegram chat ID where the message will be sent. If not provided, uses default_chat_id from config."` //nolint:lll // ok.
	Text                string                `json:"text" jsonschema_description:"Message text in Markdown format. Will be automatically converted to HTML."`                                        //nolint:lll // ok.
	DisablePreview      *bool                 `json:"disable_preview,omitempty" jsonschema_description:"Disable URL preview for links in the message."`                                               //nolint:lll // ok.
	DisableNotification *bool                 `json:"disable_notification,omitempty" jsonschema_description:"Disable notification for the message (silent push)."`                                    //nolint:lll // ok.
}

// Tool provides telegram send operation as LLM tool.
type Tool struct {
	cfg     plugin.Config
	printer printer.Printer
	state   *plugin.State
}

// New creates a new Tool instance.
func New(cfg plugin.Config, p printer.Printer, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
		state:   state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "telegram"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Send a message to Telegram chat. Accepts text in Markdown format, which is automatically converted to HTML. " +
		"Requires telegram.enable=true, telegram.bot_token, and optionally telegram.default_chat_id in config."
}

// Exec executes the telegram tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	// Set default timeout
	ctx, cancel := context.WithTimeout(ctx, telegramDefaultTimeout)
	defer cancel()

	// Check if telegram is enabled
	if !t.cfg.Bool("telegram.enable") {
		return nil, errors.New("telegram: integration is disabled. Set telegram.enable=true in config")
	}

	// Get bot token from config
	botToken := t.cfg.String("telegram.bot_token")
	if botToken == "" {
		return nil, errors.New("telegram: bot_token is not configured")
	}

	// Determine chat ID
	chatID := ""
	if params.ChatID != nil {
		chatID = params.ChatID.String()
	}
	if chatID == "" {
		chatID = t.cfg.String("telegram.default_chat_id")
		if chatID == "" {
			return nil, errors.New("telegram: chat_id is not provided and default_chat_id is not configured")
		}
	}

	// Validate chat_id against allowed list if configured
	allowedChatIDs := t.cfg.Strings("telegram.allowed_chat_ids")
	if len(allowedChatIDs) > 0 {
		found := slices.Contains(allowedChatIDs, chatID)
		if !found {
			return nil, errors.New("telegram: chat_id is not in the allowed list")
		}
	}

	t.printer.ToolCall(printer.IconRobot, "telegram", "chat_id", chatID)

	if params.Text == "" {
		return nil, errors.New("telegram: text is required")
	}

	// Convert markdown to HTML
	htmlText, err := convertMarkdownToHTML(params.Text)
	if err != nil {
		t.printer.Error("telegram: failed to convert markdown to HTML", "err", err)
		return nil, fmt.Errorf("telegram: failed to convert markdown to HTML: %w", err)
	}

	// Prepare form data
	form := url.Values{}
	form.Add("chat_id", chatID)
	form.Add("text", htmlText)
	form.Add("parse_mode", "HTML")

	// Only add disable_web_page_preview if set to true (value "1")
	if params.DisablePreview != nil && *params.DisablePreview {
		form.Add("disable_web_page_preview", "1")
	}

	// Only add disable_notification if set to true (value "1")
	if params.DisableNotification != nil && *params.DisableNotification {
		form.Add("disable_notification", "1")
	}

	// Build the API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	// Create HTTP request with form data
	client := t.state.RequesterFactory.Get()
	defer t.state.RequesterFactory.Put(client)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.printer.Error("telegram: failed to create request", "err", err)
		return nil, fmt.Errorf("telegram: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		t.printer.Error("telegram: failed to send message", "err", err)
		return nil, fmt.Errorf("telegram: failed to send message: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("telegram: API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.printer.Error("telegram: failed to parse response", "err", err)
		return nil, fmt.Errorf("telegram: failed to parse response: %w", err)
	}

	t.printer.ToolCall(printer.IconDone, "telegram: message sent successfully", "chat_id", chatID)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: fmt.Sprintf("Message sent successfully to chat %s", chatID),
	}, nil
}

// convertMarkdownToHTML converts markdown content to HTML using goldmark.
func convertMarkdownToHTML(markdownContent string) (string, error) {
	var buf strings.Builder
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	if err := markdown.Convert([]byte(markdownContent), &buf); err != nil {
		return "", fmt.Errorf("convert markdown to HTML: %w", err)
	}
	return buf.String(), nil
}
