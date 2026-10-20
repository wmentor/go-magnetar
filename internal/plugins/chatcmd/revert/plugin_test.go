package revertplugin

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestRevertCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messages   []openai.ChatCompletionMessage
		wantOutput string
		wantErr    bool
	}{
		{
			name: "revert with simple conversation",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
				{Role: "user", Content: "How are you?"},
			},
			wantOutput: "Reverted to previous answer:\nHi there!",
		},
		{
			name: "revert with multiple exchanges",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First answer"},
				{Role: "user", Content: "Question 1"},
				{Role: "assistant", Content: "Second answer"},
				{Role: "user", Content: "Question 2"},
			},
			wantOutput: "Reverted to previous answer:\nSecond answer",
		},
		{
			name: "revert with tool calls in between",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First answer"},
				{Role: "user", Content: "Search something"},
				{Role: "assistant", ToolCalls: []openai.ToolCall{toolCall("call_1")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
				{Role: "assistant", Content: "Second answer with tool"},
				{Role: "user", Content: "Another question"},
			},
			wantOutput: "Reverted to previous answer:\nSecond answer with tool",
		},
		{
			name: "nothing to revert - only user message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
			},
			wantOutput: "Nothing to revert: not enough conversation history",
		},
		{
			name: "nothing to revert - only system and user",
			messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "System prompt"},
				{Role: "user", Content: "Hello"},
			},
			wantOutput: "Nothing to revert: not enough conversation history",
		},
		{
			name: "nothing to revert - assistant with only tool calls",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", ToolCalls: []openai.ToolCall{toolCall("call_1")}},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
				{Role: "user", Content: "Another question"},
			},
			wantOutput: "Nothing to revert: no assistant answer found",
		},
		{
			name: "revert to first answer",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "First answer"},
				{Role: "user", Content: "Question 1"},
				{Role: "assistant", Content: "Second answer"},
				{Role: "user", Content: "Question 2"},
				{Role: "assistant", Content: "Third answer"},
				{Role: "user", Content: "Question 3"},
			},
			wantOutput: "Reverted to previous answer:\nThird answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &Plugin{}
			mockHub := &mockHub{}
			if err := p.Init(&plugin.State{}, mockHub); err != nil {
				t.Fatalf("Init failed: %v", err)
			}
			cmd := mockHub.lastCommand
			if cmd == nil {
				t.Fatal("no command registered")
			}

			response, err := cmd.Execute(context.Background(), &plugin.ChatCommandInput{
				AgentHandler: &mockAgent{messages: tt.messages},
				Args:         "",
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if response.Stdout != tt.wantOutput {
				t.Errorf("Stdout = %q, want %q", response.Stdout, tt.wantOutput)
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
