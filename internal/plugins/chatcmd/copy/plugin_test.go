package copyplugin

import (
	"context"
	"io"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestCopyCommand(t *testing.T) {
	t.Parallel()

	out = io.Discard

	tests := []struct {
		name       string
		messages   []openai.ChatCompletionMessage
		wantOutput string
		wantErr    bool
	}{
		{
			name: "copy last assistant message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			wantOutput: "Answer copied to clipboard\n",
		},
		{
			name: "copy second to last when last is user message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Response"},
				{Role: "user", Content: "Another user message"},
			},
			wantOutput: "Answer copied to clipboard\n",
		},
		{
			name: "nothing to copy - empty conversation",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
			},
			wantOutput: "Nothing to copy: no conversation history\n",
		},
		{
			name: "nothing to copy - last assistant message is empty",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: ""},
			},
			wantOutput: "Nothing to copy: last assistant message is empty\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}

			mockHub := &mockHub{}
			if err := p.Init(&plugin.State{}, mockHub); err != nil {
				t.Fatalf("Init failed: %v", err)
			}

			cmd := mockHub.lastCommand
			if cmd == nil {
				t.Fatal("no command registered")
			}

			mockAgent := &mockAgent{messages: tt.messages}

			oldOut := out
			defer func() { out = oldOut }()

			err := cmd.Execute(context.Background(), mockAgent, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockAgent struct {
	messages []openai.ChatCompletionMessage
}

func (a *mockAgent) Messages() []openai.ChatCompletionMessage {
	return a.messages
}

func (a *mockAgent) SetMessages(msgs []openai.ChatCompletionMessage) {
	a.messages = msgs
}

func (a *mockAgent) Config() *config.Config {
	return nil
}

func (a *mockAgent) Compact() error {
	return nil
}

func (a *mockAgent) Reset() {
}

func (a *mockAgent) Reconfigure() {
}

type mockHub struct {
	lastCommand *plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.lastCommand = &cmd
}

func (h *mockHub) RegisterTool(tool plugin.LLMTool) {}

func (h *mockHub) RegisterCLICommand(cmd any) {}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {}

func (h *mockHub) Go(f func(context.Context)) {}

func (h *mockHub) Stop() {}
