package writeplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	history := []openai.ChatCompletionMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "again"},
		{Role: "assistant", Content: "the latest answer"},
	}

	dir := t.TempDir()
	okFile := filepath.Join(dir, "answer.md")
	dirPath := filepath.Join(dir, "sub")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}

	big := strings.Repeat("line\nwith `ticks` and \"quotes\"\n", 500)

	tests := []struct {
		name      string
		msgs      []openai.ChatCompletionMessage
		args      string
		wantIn    string   // substrings expected in Stdout
		wantFile  string   // content expected in the file, if any
		wantNoLog bool
	}{
		{
			name:   "no filename prints usage",
			msgs:   history,
			args:   "",
			wantIn: "Usage: `/write <filename>`",
		},
		{
			name:   "empty history has nothing to save",
			msgs:   nil,
			args:   okFile,
			wantIn: "Nothing to save: no conversation history",
		},
		{
			name: "single message has nothing to save",
			msgs: []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}},
			args: okFile, wantIn: "Nothing to save: no conversation history",
		},
		{
			name: "empty assistant message has nothing to save",
			msgs: []openai.ChatCompletionMessage{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: ""},
			},
			args:   okFile,
			wantIn: "Nothing to save: last assistant message is empty",
		},
		{
			name:     "saves the last assistant answer",
			msgs:     history,
			args:     okFile,
			wantIn:   "Answer saved to `" + okFile + "`",
			wantFile: "the latest answer",
		},
		{
			name: "last message from user falls back to previous assistant",
			msgs: []openai.ChatCompletionMessage{
				{Role: "user", Content: "q"},
				{Role: "assistant", Content: "saved answer"},
				{Role: "user", Content: "thanks"},
			},
			args:     filepath.Join(dir, "fallback.md"),
			wantIn:   "Answer saved to",
			wantFile: "saved answer",
		},
		{
			name:     "special characters survive",
			msgs:     []openai.ChatCompletionMessage{{Role: "user", Content: "q"}, {Role: "assistant", Content: big}},
			args:     filepath.Join(dir, "big.md"),
			wantIn:   "Answer saved to",
			wantFile: big,
		},
		{
			name:   "directory as target is an error, not a crash",
			msgs:   history,
			args:   dirPath,
			wantIn: "**Error saving file:**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := execute(tt.msgs, tt.args)
			if resp == nil {
				t.Fatal("execute returned nil")
			}
			if resp.Content != "" {
				t.Errorf("Content should stay empty, got %q", resp.Content)
			}
			for _, want := range []string{tt.wantIn} {
				if !strings.Contains(resp.Stdout, want) {
					t.Errorf("Stdout = %q, want it to contain %q", resp.Stdout, want)
				}
			}
			if tt.wantFile != "" {
				got, err := os.ReadFile(tt.args)
				if err != nil {
					t.Fatalf("expected file %s: %v", tt.args, err)
				}
				if string(got) != tt.wantFile {
					t.Errorf("file content mismatch: len %d, want len %d", len(got), len(tt.wantFile))
				}
			}
		})
	}
}
