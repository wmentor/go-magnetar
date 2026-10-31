package compactplugin

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
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
		name     string
		args     string
		wantMode []string
	}{
		{name: "default mode", args: "", wantMode: nil},
		{name: "soft mode", args: "soft", wantMode: []string{"soft"}},
		{name: "full mode", args: "full", wantMode: []string{"full"}},
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

			_, err := hub.cmd.Execute(context.Background(), &plugin.ChatCommandInput{
				AgentHandler: handler,
				Args:         tt.args,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, gotMode)
		})
	}
}
