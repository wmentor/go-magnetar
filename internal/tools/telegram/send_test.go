package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/types"
)

func TestExec_NilParams(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	_, err := tool.Exec(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid params")
}

func TestExec_Disabled(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	cfg.On("Bool", "telegram.enable").Return(false)

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	_, err := tool.Exec(context.Background(), &Params{Text: "Hello"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "integration is disabled")
}

func TestExec_MissingBotToken(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	cfg.On("Bool", "telegram.enable").Return(true)
	cfg.On("String", "telegram.bot_token").Return("")

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	_, err := tool.Exec(context.Background(), &Params{Text: "Hello"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bot_token is not configured")
}

func TestExec_MissingChatID(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	cfg.On("Bool", "telegram.enable").Return(true)
	cfg.On("String", "telegram.bot_token").Return("test_token")
	cfg.On("String", "telegram.default_chat_id").Return("")

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	_, err := tool.Exec(context.Background(), &Params{Text: "Hello"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "chat_id is not provided")
}

func TestExec_MissingText(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	cfg.On("Bool", "telegram.enable").Return(true)
	cfg.On("String", "telegram.bot_token").Return("test_token")
	cfg.On("Strings", "telegram.allowed_chat_ids").Return([]string{})

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	fs := types.FlexibleString("123")
	_, err := tool.Exec(context.Background(), &Params{ChatID: &fs})
	require.Error(t, err)
	require.Contains(t, err.Error(), "text is required")
}

func TestExec_AllowedChatIDs(t *testing.T) {
	t.Parallel()

	cfg := testdata.NewConfig(t)
	cfg.On("Bool", "telegram.enable").Return(true)
	cfg.On("String", "telegram.bot_token").Return("test_token")
	cfg.On("Strings", "telegram.allowed_chat_ids").Return([]string{"123", "456"})

	tool := New(cfg, printer.GetDefault(), &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		RequesterFactory: nil,
	})

	fs := types.FlexibleString("999")
	_, err := tool.Exec(context.Background(), &Params{ChatID: &fs, Text: "Hello"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in the allowed list")
}

func TestConvertMarkdownToHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple paragraph",
			input:    "Hello, World!",
			contains: "<p>Hello, World!</p>",
		},
		{
			name:     "heading",
			input:    "# Title",
			contains: "<h1>Title</h1>",
		},
		{
			name:     "bold text",
			input:    "**bold**",
			contains: "<strong>bold</strong>",
		},
		{
			name:     "italic text",
			input:    "*italic*",
			contains: "<em>italic</em>",
		},
		{
			name:     "link",
			input:    "[link](https://example.com)",
			contains: `<a href="https://example.com">link</a>`,
		},
		{
			name:     "unordered list",
			input:    "- item1\n- item2",
			contains: "<li>item1</li>",
		},
		{
			name:     "code block",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			contains: "<code",
		},
		{
			name:     "table (GFM)",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: "<table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			result, err := convertMarkdownToHTML(tt.input)
			require.NoError(t, err)
			require.Contains(t, result, tt.contains)
		})
	}
}

//go:fix inline
func stringPtr(s string) *string {
	return new(s)
}

//go:fix inline
func boolPtr(b bool) *bool {
	return new(b)
}
