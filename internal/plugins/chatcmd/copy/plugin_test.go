package copyplugin

import (
	"context"
	"errors"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/response"
)

func TestCopyCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		responses  []string
		args       string
		writeErr   error
		wantOutput string
		wantClip   string
	}{
		{
			name:       "copy latest response",
			responses:  []string{"First answer", "Latest answer"},
			wantOutput: "Response copied to clipboard\n",
			wantClip:   "Latest answer",
		},
		{
			name:       "copy command output",
			responses:  []string{"Model answer", "command stdout"},
			wantOutput: "Response copied to clipboard\n",
			wantClip:   "command stdout",
		},
		{
			name:       "copy index zero explicitly",
			responses:  []string{"First", "Second"},
			args:       "0",
			wantOutput: "Response copied to clipboard\n",
			wantClip:   "Second",
		},
		{
			name:       "copy previous response with index one",
			responses:  []string{"First", "Second"},
			args:       "1",
			wantOutput: "Response copied to clipboard\n",
			wantClip:   "First",
		},
		{
			name:       "empty response store",
			wantOutput: "Nothing to copy: no response found at index 0\n",
		},
		{
			name:       "selected response is empty",
			responses:  []string{""},
			wantOutput: "Nothing to copy: selected response is empty\n",
		},
		{
			name:       "index out of range",
			responses:  []string{"Only response"},
			args:       "1",
			wantOutput: "Nothing to copy: no response found at index 1\n",
		},
		{
			name:       "invalid index",
			args:       "nope",
			wantOutput: "usage: /copy [index] — index must be a non-negative integer (0 = most recent response)\n",
		},
		{
			name:       "clipboard error",
			responses:  []string{"Response"},
			writeErr:   errors.New("clipboard unavailable"),
			wantOutput: "Error copying to clipboard: clipboard unavailable\n",
		},
	}

	// Subtests run sequentially because they replace the package-level clipboard writer.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var clipped string
			oldWrite := writeAll
			writeAll = func(text string) error {
				clipped = text
				return tt.writeErr
			}
			t.Cleanup(func() { writeAll = oldWrite })

			p := &Plugin{}
			responseStore := response.New(plugin.DefaultResponseStoreSize)
			for _, item := range tt.responses {
				responseStore.Push(item)
			}
			mockHub := &mockHub{}
			if err := p.Init(&plugin.State{ResponseStore: responseStore}, mockHub); err != nil {
				t.Fatalf("Init failed: %v", err)
			}
			cmd := mockHub.lastCommand
			if cmd == nil {
				t.Fatal("no command registered")
			}

			got, err := cmd.Execute(context.Background(), &plugin.ChatCommandInput{Args: tt.args})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got == nil {
				t.Fatal("Execute() returned a nil response")
			}
			if got.Stdout != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", got.Stdout, tt.wantOutput)
			}
			if tt.wantClip != "" && clipped != tt.wantClip {
				t.Errorf("clipboard = %q, want %q", clipped, tt.wantClip)
			}
		})
	}
}

func TestCopyCommandReturnsUnexpectedResponseStoreErrors(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("response store failed")
	p := &Plugin{}
	hub := &mockHub{}
	if err := p.Init(&plugin.State{ResponseStore: failingResponseStore{err: storeErr}}, hub); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	got, err := hub.lastCommand.Execute(context.Background(), &plugin.ChatCommandInput{})
	if !errors.Is(err, storeErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, storeErr)
	}
	if got != nil {
		t.Errorf("Execute() response = %#v, want nil when the store fails", got)
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

type mockHub struct {
	lastCommand *plugin.ChatCommand
}

type failingResponseStore struct {
	err error
}

func (s failingResponseStore) Push(string) {}

func (s failingResponseStore) Last(int) (string, error) { return "", s.err }

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand)      { h.lastCommand = &cmd }
func (h *mockHub) RegisterTool(tool plugin.LLMTool)                {}
func (h *mockHub) RegisterCLICommand(cmd any)                      {}
func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {}
func (h *mockHub) Go(f func(context.Context))                      {}
func (h *mockHub) Stop()                                           {}
