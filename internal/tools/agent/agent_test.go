package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

func TestTool_Exec(t *testing.T) {
	t.Parallel()

	// Create a temporary config file for tests
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/config.yml"
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	// Create a mock state with minimum required fields
	state := &plugin.State{
		Config:           cfg,
		ReadOnly:         false,
		Printer:          p,
		RequesterFactory: clientFactory,
	}

	// Create the tool
	tool := NewTool(cfg, p, clientFactory, state)

	tests := []struct {
		name        string
		params      *AgentParams
		wantErr     bool
		checkResult bool
		checkError  string
	}{
		{
			name:        "nil params",
			params:      nil,
			wantErr:     true,
			checkResult: false,
		},
		{
			name:        "empty agent name",
			params:      &AgentParams{Agent: "", Message: "test"},
			wantErr:     true,
			checkResult: false,
			checkError:  "agent name is required",
		},
		{
			name:        "empty message",
			params:      &AgentParams{Agent: "default", Message: ""},
			wantErr:     true,
			checkResult: false,
			checkError:  "message is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			result, err := tool.Exec(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)

				if tt.checkError != "" {
					assert.Contains(t, err.Error(), tt.checkError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Parse result as tools.Response
				var resp tools.Response
				switch v := result.(type) {
				case string:
					// Try to unmarshal from JSON string
					// This handles the case when result is returned as JSON string
					if err := json.Unmarshal([]byte(v), &resp); err != nil {
						t.Fatalf("failed to parse response: %v", err)
					}
				case *tools.Response:
					resp = *v
				default:
					t.Fatalf("unexpected result type: %T", result)
				}

				assert.Equal(t, tools.StatusOK, resp.Status)
			}
		})
	}
}
