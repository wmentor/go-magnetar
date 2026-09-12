package sessionplugin

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

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

func TestSaveSessionCommand(t *testing.T) {
	t.Parallel()

	out = io.Discard

	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "session.json")

	tests := []struct {
		name     string
		messages []openai.ChatCompletionMessage
		args     string
		wantErr  bool
	}{
		{
			name:     "save session with valid filename",
			messages: []openai.ChatCompletionMessage{},
			args:     sessionFile,
			wantErr:  false,
		},
		{
			name:     "no filename provided",
			messages: []openai.ChatCompletionMessage{},
			args:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}

			hub := &mockHub{}
			if err := p.Init(&plugin.State{}, hub); err != nil {
				t.Fatalf("Init failed: %v", err)
			}

			cmd := hub.lastCommand
			if cmd == nil {
				t.Fatal("no command registered")
			}

			agent := &mockAgent{messages: tt.messages}
			err := cmd.Execute(context.Background(), agent, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.args != "" {
				if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
					t.Errorf("Session file was not created")
				}

				data, err := os.ReadFile(sessionFile)
				if err != nil {
					t.Fatalf("Failed to read session file: %v", err)
				}

				var savedMsgs []openai.ChatCompletionMessage
				if err := json.Unmarshal(data, &savedMsgs); err != nil {
					t.Errorf("Failed to parse session file: %v", err)
				}
			}
		})
	}
}

type mockHub struct {
	lastCommand *plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.lastCommand = &cmd
}

func (h *mockHub) RegisterTool(tool plugin.LLMTool) {}
func (h *mockHub) RegisterCLICommand(cmd any)       {}
func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
}
func (h *mockHub) Go(f func(context.Context)) {}
func (h *mockHub) Stop()                      {}
