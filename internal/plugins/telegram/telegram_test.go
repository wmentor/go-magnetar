package telegramplugin

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

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type mockHub struct{}

func (h *mockHub) RegisterChatCommand(plugin.ChatCommand)       {}
func (h *mockHub) RegisterPreprocessor(plugin.PreprocessorFunc) {}
func (h *mockHub) Go(func(ctx context.Context))                 {}
func (h *mockHub) Stop()                                        {}

func newTestState(t *testing.T, cfg plugin.Config, factory plugin.ClientFactory) *plugin.State {
	t.Helper()

	maskService := testdata.NewMaskService(t)
	maskService.EXPECT().Process(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, input string) string { return input }).
		Maybe()

	return &plugin.State{
		Config:           cfg,
		Printer:          printer.GetDefault(),
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: factory,
	}
}

func toolNames(store *tools.Store) []string {
	registered := store.Tools()
	names := make([]string, 0, len(registered))
	for _, tool := range registered {
		names = append(names, tool.Name)
	}
	return names
}

func TestInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		enabled             bool
		nilHub              bool
		wantTool            bool
		wantDescriptionPart string
	}{
		{
			name:     "disabled by default",
			enabled:  false,
			wantTool: false,
		},
		{
			name:     "explicitly disabled",
			enabled:  false,
			wantTool: false,
		},
		{
			name:                "enabled",
			enabled:             true,
			wantTool:            true,
			wantDescriptionPart: "Send a message to Telegram chat",
		},
		{
			name:                "enabled with nil hub",
			enabled:             true,
			nilHub:              true,
			wantTool:            true,
			wantDescriptionPart: "Send a message to Telegram chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			cfg.EXPECT().Bool("telegram.enable").Return(tt.enabled).Once()
			state := newTestState(t, cfg, testdata.NewClientFactory(t))

			var hub plugin.Hub = &mockHub{}
			if tt.nilHub {
				hub = nil
			}

			p := &Plugin{}
			require.NoError(t, p.Init(state, hub))
			assert.Same(t, state, p.state)

			if !tt.wantTool {
				assert.Empty(t, toolNames(state.ToolStore))
				assert.Nil(t, p.tool)
				_, err := state.ToolStore.GetTool("telegram")
				assert.ErrorIs(t, err, tools.ErrToolNotFound)
				return
			}

			assert.Equal(t, []string{"telegram"}, toolNames(state.ToolStore))
			tool, err := state.ToolStore.GetTool("telegram")
			require.NoError(t, err)
			def := tool.Definition()
			assert.Equal(t, "telegram", def.Function.Name)
			assert.Contains(t, def.Function.Description, tt.wantDescriptionPart)
			assert.NotNil(t, def.Function.Parameters)
		})
	}
}

func TestRegisteredToolExecute(t *testing.T) {
	t.Parallel()

	successResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}
	}

	expectHTTP := func(factory *testdata.ClientFactory, client *testdata.Client, resp *http.Response, err error) {
		factory.EXPECT().Get().Return(client).Once()
		factory.EXPECT().Put(client).Once()
		client.EXPECT().Do(mock.Anything).Return(resp, err).Once()
	}

	tests := []struct {
		name         string
		args         string
		execDisabled bool
		decodeFails  bool
		setup        func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client)
		want         []string
	}{
		{
			name: "send with explicit chat ID",
			args: `{"chat_id":"12345","text":"Hello **world**"}`,
			setup: func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return(nil).Once()
				expectHTTP(factory, client, successResponse(), nil)
			},
			want: []string{`"status":"OK"`, "Message sent successfully to chat 12345"},
		},
		{
			name: "send with default chat ID and optional flags",
			args: `{"text":"Hello","disable_preview":true,"disable_notification":true}`,
			setup: func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().String("telegram.default_chat_id").Return("default-chat").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return([]string{"default-chat"}).Once()
				expectHTTP(factory, client, successResponse(), nil)
			},
			want: []string{`"status":"OK"`, "Message sent successfully to chat default-chat"},
		},
		{
			name:         "disabled when executed",
			args:         `{"chat_id":"12345","text":"Hello"}`,
			execDisabled: true,
			setup: func(t *testing.T, _ *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
			},
			want: []string{`"status":"validation_failed"`, "integration is disabled"},
		},
		{
			name: "missing bot token",
			args: `{"chat_id":"12345","text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("").Once()
			},
			want: []string{`"status":"validation_failed"`, "bot_token is not configured"},
		},
		{
			name: "missing chat ID and default chat ID",
			args: `{"text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().String("telegram.default_chat_id").Return("").Once()
			},
			want: []string{`"status":"validation_failed"`, "chat_id is not provided and default_chat_id is not configured"},
		},
		{
			name: "chat ID not in allowed list",
			args: `{"chat_id":"999","text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return([]string{"123", "456"}).Once()
			},
			want: []string{`"status":"validation_failed"`, "not in the allowed list"},
		},
		{
			name: "empty text",
			args: `{"chat_id":"12345"}`,
			setup: func(t *testing.T, cfg *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return(nil).Once()
			},
			want: []string{`"status":"validation_failed"`, "text is required"},
		},
		{
			name: "client fails to send request",
			args: `{"chat_id":"12345","text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return(nil).Once()
				expectHTTP(factory, client, nil, errors.New("network is down"))
			},
			want: []string{`"status":"validation_failed"`, "failed to send message: network is down"},
		},
		{
			name: "API returns error status",
			args: `{"chat_id":"12345","text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return(nil).Once()
				resp := &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("bad request")),
				}
				expectHTTP(factory, client, resp, nil)
			},
			want: []string{`"status":"validation_failed"`, "API returned status 400: bad request"},
		},
		{
			name: "API returns malformed JSON",
			args: `{"chat_id":"12345","text":"Hello"}`,
			setup: func(t *testing.T, cfg *testdata.Config, factory *testdata.ClientFactory, client *testdata.Client) {
				t.Helper()
				cfg.EXPECT().String("telegram.bot_token").Return("test-token").Once()
				cfg.EXPECT().Strings("telegram.allowed_chat_ids").Return(nil).Once()
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not-json")),
				}
				expectHTTP(factory, client, resp, nil)
			},
			want: []string{`"status":"validation_failed"`, "failed to parse response"},
		},
		{
			name:        "malformed tool arguments",
			args:        `{"text":`,
			decodeFails: true,
			setup: func(t *testing.T, _ *testdata.Config, _ *testdata.ClientFactory, _ *testdata.Client) {
				t.Helper()
			},
			want: []string{`"status":"validation_failed"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testdata.NewConfig(t)
			factory := testdata.NewClientFactory(t)
			client := testdata.NewClient(t)

			switch {
			case tt.execDisabled:
				cfg.EXPECT().Bool("telegram.enable").Return(true).Once()
				cfg.EXPECT().Bool("telegram.enable").Return(false).Once()
			case tt.decodeFails:
				cfg.EXPECT().Bool("telegram.enable").Return(true).Once()
			default:
				cfg.EXPECT().Bool("telegram.enable").Return(true).Twice()
			}
			tt.setup(t, cfg, factory, client)

			state := newTestState(t, cfg, factory)
			p := &Plugin{}
			require.NoError(t, p.Init(state, &mockHub{}))
			tool, err := state.ToolStore.GetTool("telegram")
			require.NoError(t, err)

			result := tool.Execute(context.Background(), []byte(tt.args))
			for _, fragment := range tt.want {
				assert.Contains(t, result, fragment)
			}
		})
	}
}
