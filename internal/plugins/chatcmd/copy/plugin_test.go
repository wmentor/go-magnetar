package copyplugin

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/response"
)

func TestCopyCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messages   []openai.ChatCompletionMessage
		args       string
		wantOutput string
		wantClip   string
	}{
		{
			name: "copy last assistant message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "Hi there!",
		},
		{
			name: "copy index 0 explicitly",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First"},
				{Role: "user", Content: "Again"},
				{Role: "assistant", Content: "Second"},
			},
			args:       "0",
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "Second",
		},
		{
			name: "copy previous assistant with index 1",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First"},
				{Role: "user", Content: "Again"},
				{Role: "assistant", Content: "Second"},
			},
			args:       "1",
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "First",
		},
		{
			name: "copy when last message is user still uses last assistant as index 0",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Response"},
				{Role: "user", Content: "Another user message"},
			},
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "Response",
		},
		{
			name: "skips assistant tool call message after the answer",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Answer"},
				{Role: "user", Content: "Search something"},
				{Role: "assistant", ToolCalls: []openai.ToolCall{toolCall("call_1")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
			},
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "Answer",
		},
		{
			name: "tool call messages do not count towards index",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First"},
				{Role: "user", Content: "Search something"},
				{Role: "assistant", ToolCalls: []openai.ToolCall{toolCall("call_1")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
				{Role: "assistant", Content: "Let me check more", ToolCalls: []openai.ToolCall{toolCall("call_2")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_2"},
				{Role: "assistant", Content: "Second"},
			},
			args:       "1",
			wantOutput: "Answer copied to clipboard\n",
			wantClip:   "First",
		},
		{
			name: "only tool call assistant messages",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", ToolCalls: []openai.ToolCall{toolCall("call_1")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
			},
			wantOutput: "Nothing to copy: no conversation history\n",
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
			wantOutput: "Nothing to copy: selected assistant message is empty\n",
		},
		{
			name: "index out of range",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Only one"},
			},
			args:       "1",
			wantOutput: "Nothing to copy: only 1 assistant answer(s) in history\n",
		},
		{
			name: "invalid index",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			args:       "nope",
			wantOutput: "usage: /copy [index] — index must be a non-negative integer (0 = last answer)\n",
		},
	}

	// Subtests run sequentially because they swap the package-level out and writeAll.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var clipped string
			oldOut, oldWrite := out, writeAll
			out = &buf
			writeAll = func(text string) error {
				clipped = text
				return nil
			}
			t.Cleanup(func() {
				out = oldOut
				writeAll = oldWrite
			})

			p := &Plugin{}
			responseStore := response.New(plugin.DefaultResponseStoreSize)
			mockHub := &mockHub{}
			if err := p.Init(&plugin.State{ResponseStore: responseStore}, mockHub); err != nil {
				t.Fatalf("Init failed: %v", err)
			}
			cmd := mockHub.lastCommand
			if cmd == nil {
				t.Fatal("no command registered")
			}

			_, err := cmd.Execute(context.Background(), &plugin.ChatCommandInput{
				AgentHandler: &mockAgent{messages: tt.messages},
				Args:         tt.args,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got := buf.String(); got != tt.wantOutput {
				t.Errorf("output = %q, want %q", got, tt.wantOutput)
			}
			if tt.wantClip != "" && clipped != tt.wantClip {
				t.Errorf("clipboard = %q, want %q", clipped, tt.wantClip)
			}
		})
	}
}

func TestParseCopyIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args    string
		want    int
		wantErr error
	}{
		{args: "", want: 0},
		{args: "  ", want: 0},
		{args: "0", want: 0},
		{args: " 2 ", want: 2},
		{args: "nope", wantErr: ErrInvalidIndex},
		{args: "-1", wantErr: ErrInvalidIndex},
		{args: "1.5", wantErr: ErrInvalidIndex},
	}

	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			t.Parallel()

			got, err := parseCopyIndex(tt.args)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseCopyIndex(%q) error = %v, want %v", tt.args, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseCopyIndex(%q) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func toolCall(id string) openai.ToolCall {
	return openai.ToolCall{
		ID:   id,
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "search",
			Arguments: `{"query":"something"}`,
		},
	}
}

type mockAgent struct {
	messages []openai.ChatCompletionMessage
}

func (a *mockAgent) Messages() []openai.ChatCompletionMessage { return a.messages }
func (a *mockAgent) SetMessages(msgs []openai.ChatCompletionMessage) {
	a.messages = msgs
}
func (a *mockAgent) Config() *config.Config { return nil }
func (a *mockAgent) Compact() error         { return nil }
func (a *mockAgent) Reset()                 {}
func (a *mockAgent) Reconfigure()           {}

type mockHub struct {
	lastCommand *plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand)      { h.lastCommand = &cmd }
func (h *mockHub) RegisterTool(tool plugin.LLMTool)                {}
func (h *mockHub) RegisterCLICommand(cmd any)                      {}
func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {}
func (h *mockHub) Go(f func(context.Context))                      {}
func (h *mockHub) Stop()                                           {}
