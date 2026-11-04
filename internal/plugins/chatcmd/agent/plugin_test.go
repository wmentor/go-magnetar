package agentplugin

import (
	"context"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestSplitArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		n     int
		want  []string
	}{
		{
			name:  "single word",
			input: "hello",
			n:     2,
			want:  []string{"hello"}, // Single element when no space found
		},
		{
			name:  "two words",
			input: "hello world",
			n:     2,
			want:  []string{"hello", "world"},
		},
		{
			name:  "three words with n=2",
			input: "hello world foo",
			n:     2,
			want:  []string{"hello", "world foo"},
		},
		{
			name:  "empty string",
			input: "",
			n:     2,
			want:  []string{""},
		},
		{
			name:  "multiple spaces",
			input: "hello   world",
			n:     2,
			want:  []string{"hello", "  world"},
		},
		{
			name:  "n=3 with 3 words",
			input: "one two three",
			n:     3,
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "n=3 with 4 words",
			input: "one two three four",
			n:     3,
			want:  []string{"one", "two", "three four"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			result := splitArgs(tt.input, tt.n)
			if len(result) != len(tt.want) {
				t.Errorf("splitArgs() = %v, want %v", result, tt.want)
				return
			}
			for i := range result {
				if result[i] != tt.want[i] {
					t.Errorf("splitArgs()[%d] = %q, want %q", i, result[i], tt.want[i])
				}
			}
		})
	}
}

func TestAgentPlugin_Init(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	hub := &mockHub{}

	err := p.Init(&plugin.State{}, hub)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if len(hub.commands) != 1 {
		t.Errorf("expected 1 command registered, got %d", len(hub.commands))
	}

	if hub.commands[0].Name != "agent" {
		t.Errorf("expected command name 'agent', got '%s'", hub.commands[0].Name)
	}

	if len(hub.commands[0].Aliases) != 0 {
		t.Errorf("expected no aliases, got %v", hub.commands[0].Aliases)
	}
}

type mockHub struct {
	commands []plugin.ChatCommand
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.commands = append(h.commands, cmd)
}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	// no-op for testing
}

func (h *mockHub) Go(f func(ctx context.Context)) {
	// no-op for testing
}

func (h *mockHub) Stop() {
	// no-op for testing
}
