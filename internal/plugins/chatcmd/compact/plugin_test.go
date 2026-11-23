package compactplugin

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
	printerdata "github.com/wmentor/go-magnetar/internal/printer/testdata"
)

type mockCompactHub struct {
	cmd plugin.ChatCommand
}

func (h *mockCompactHub) RegisterChatCommand(c plugin.ChatCommand)     { h.cmd = c }
func (h *mockCompactHub) RegisterPreprocessor(plugin.PreprocessorFunc) {}
func (h *mockCompactHub) Go(func(ctx context.Context))                 {}
func (h *mockCompactHub) Stop()                                        {}

func TestCompactCommandModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       string
		wantMode   []string
		wantStdout string
	}{
		{name: "default mode", args: "", wantMode: nil, wantStdout: "Context compacted: 2 → 2 messages.\n\n"},
		{name: "soft mode", args: "soft", wantMode: []string{"soft"}, wantStdout: "Context compacted: 2 → 2 messages.\n\n"},
		{name: "full mode", args: "full", wantMode: []string{"full"}, wantStdout: "Context compacted: 2 → 2 messages.\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hub := &mockCompactHub{}
			require.NoError(t, (&Plugin{}).Init(&plugin.State{}, hub))

			handler := testdata.NewAgentHandle(t)
			handler.EXPECT().Messages().Return([]openai.ChatCompletionMessage{{Role: "system"}, {Role: "user"}}).Twice()
			var gotMode []string
			if len(tt.wantMode) == 0 {
				handler.EXPECT().Compact().RunAndReturn(func(mode ...string) error {
					gotMode = mode
					return nil
				})
			} else {
				handler.EXPECT().Compact(tt.wantMode[0]).RunAndReturn(func(mode ...string) error {
					gotMode = mode
					return nil
				})
			}

			resp, err := hub.cmd.Execute(context.Background(), &plugin.ChatCommandInput{
				AgentHandler: handler,
				Args:         tt.args,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantMode, gotMode)
			assert.Equal(t, tt.wantStdout, resp.Stdout)
		})
	}
}

func TestCompactCommandUsage(t *testing.T) {
	t.Parallel()

	hub := &mockCompactHub{}
	require.NoError(t, (&Plugin{}).Init(&plugin.State{}, hub))

	resp, err := hub.cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		Args: "nope",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Usage: /compact [soft|full]\n", resp.Stdout)
}

func TestCompactCommandError(t *testing.T) {
	t.Parallel()

	hub := &mockCompactHub{}
	pr := printerdata.NewPrinter(t)
	compactErr := assert.AnError
	pr.EXPECT().Error("chat: manual compaction failed", "err", compactErr).Once()

	require.NoError(t, (&Plugin{}).Init(&plugin.State{Printer: pr}, hub))

	handler := testdata.NewAgentHandle(t)
	handler.EXPECT().Messages().Return([]openai.ChatCompletionMessage{{Role: "system"}}).Once()
	handler.EXPECT().Compact().Return(compactErr).Once()

	resp, err := hub.cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		AgentHandler: handler,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Error: compaction failed — see logs for details.\n", resp.Stdout)
}
