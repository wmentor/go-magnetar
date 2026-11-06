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

// mockAgentStore is a mock implementation of plugin.AgentStore for testing
type mockAgentStore struct {
	agents map[string]*plugin.Agent
}

func (m *mockAgentStore) Get(agentName string) (*plugin.Agent, error) {
	if agent, ok := m.agents[agentName]; ok {
		return agent, nil
	}
	return nil, plugin.ErrAgentNotFound
}

func (m *mockAgentStore) List() []*plugin.Agent {
	result := make([]*plugin.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		result = append(result, agent)
	}
	return result
}

func (m *mockAgentStore) Filter(agents []string) (plugin.AgentStore, error) {
	filtered := &mockAgentStore{agents: make(map[string]*plugin.Agent)}
	for _, name := range agents {
		if agent, ok := m.agents[name]; ok {
			filtered.agents[name] = agent
		}
	}
	return filtered, nil
}

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

	// Create a mock agent store
	agentStore := &mockAgentStore{
		agents: map[string]*plugin.Agent{
			"test-agent": {
				Name:        "test-agent",
				Description: "Test agent for unit tests",
				Disable:     false,
			},
		},
	}

	// Create a mock state with minimum required fields
	state := &plugin.State{
		Config:           cfg,
		ReadOnly:         false,
		Printer:          p,
		RequesterFactory: clientFactory,
		AgentStore:       agentStore,
	}

	// Create the tool
	tool := NewTool("test-agent", state)

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
			name:        "empty message",
			params:      &AgentParams{Message: ""},
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

func TestTool_Name(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/config.yml"
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	agentStore := &mockAgentStore{agents: make(map[string]*plugin.Agent)}

	state := &plugin.State{
		Config:           cfg,
		Printer:          p,
		RequesterFactory: clientFactory,
		AgentStore:       agentStore,
	}

	tests := []struct {
		name      string
		agentName string
		want      string
	}{
		{
			name:      "simple name",
			agentName: "reviewer",
			want:      "agent_reviewer",
		},
		{
			name:      "name with hyphens",
			agentName: "code-reviewer",
			want:      "agent_code_reviewer",
		},
		{
			name:      "name with multiple hyphens",
			agentName: "my-complex-agent-name",
			want:      "agent_my_complex_agent_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			tool := NewTool(tt.agentName, state)
			assert.Equal(t, tt.want, tool.Name())
		})
	}
}

func TestTool_Description(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/config.yml"
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	agentStore := &mockAgentStore{
		agents: map[string]*plugin.Agent{
			"test-agent": {
				Name:        "test-agent",
				Description: "This is a test agent description",
				Disable:     false,
			},
		},
	}

	state := &plugin.State{
		Config:           cfg,
		Printer:          p,
		RequesterFactory: clientFactory,
		AgentStore:       agentStore,
	}

	tool := NewTool("test-agent", state)
	desc := tool.Description()
	assert.Equal(t, "This is a test agent description", desc)
}
