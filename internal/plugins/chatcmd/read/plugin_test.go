package readplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
)

func TestReadCommand(t *testing.T) {
	t.Parallel()

	// Disable less for tests
	fromTest = true

	t.Logf("fromTest = %v", fromTest)

	tests := []struct {
		name    string
		args    string
		wantErr bool
		check   func(t *testing.T, response *plugin.ChatCommandResponse)
	}{
		{
			name:    "read existing file",
			args:    "testdata/test.md",
			wantErr: false,
			check: func(t *testing.T, response *plugin.ChatCommandResponse) {
				t.Logf("response.Stdout = %q", response.Stdout)
				t.Logf("response.Stdout len = %d", len(response.Stdout))
				if response == nil {
					t.Fatal("response is nil")
				}
				if response.Stdout == "" {
					t.Fatal("response.Stdout is empty")
				}
				if !contains(response.Stdout, "Test file") {
					t.Errorf("expected stdout to contain 'Test file', got: %s", response.Stdout)
				}
				if !contains(response.Stdout, "Line 3") {
					t.Errorf("expected stdout to contain 'Line 3', got: %s", response.Stdout)
				}
			},
		},
		{
			name:    "read non-existent file",
			args:    "testdata/nonexistent.md",
			wantErr: false,
			check: func(t *testing.T, response *plugin.ChatCommandResponse) {
				if response == nil {
					t.Fatal("response is nil")
				}
				if !contains(response.Stdout, "Error reading file") {
					t.Errorf("expected stdout to contain error message, got: %s", response.Stdout)
				}
			},
		},
		{
			name:    "no args provided",
			args:    "",
			wantErr: false,
			check: func(t *testing.T, response *plugin.ChatCommandResponse) {
				if response == nil {
					t.Fatal("response is nil")
				}
				if !contains(response.Stdout, "Usage: /read <filename>") {
					t.Errorf("expected stdout to contain usage, got: %s", response.Stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create minimal config
			tmpDir := t.TempDir()
			cfgPath := filepath.Join(tmpDir, "config.yml")
			if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			s := &plugin.State{
				Config:        cfg,
				Printer:       printer.GetDefault(),
				ResponseStore: response.New(plugin.DefaultResponseStoreSize),
			}
			p := &Plugin{state: s}

			params := &plugin.ChatCommandInput{
				Args: tt.args,
			}

			response, err := p.execute(context.Background(), params)

			if (err != nil) != tt.wantErr {
				t.Errorf("execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.check(t, response)
		})
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || containsStr(haystack, needle))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
