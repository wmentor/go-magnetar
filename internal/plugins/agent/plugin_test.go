package agentplugin

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// mockHub is a mock implementation of plugin.Hub interface
type mockHub struct {
	chatCommands  []plugin.ChatCommand
	preprocessors []plugin.PreprocessorFunc
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.chatCommands = append(m.chatCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	m.preprocessors = append(m.preprocessors, fn)
}

func (m *mockHub) Go(f func(ctx context.Context)) {
	// No-op for testing
}

func (m *mockHub) Stop() {
	// No-op for testing
}

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

func createTestState(t *testing.T) *plugin.State {
	t.Helper()

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	// Create mock agent store with test agents
	agentStore := &mockAgentStore{
		agents: map[string]*plugin.Agent{
			"code-reviewer": {
				Name:        "code-reviewer",
				Description: "Performs code reviews",
				Disable:     false,
			},
			"security-analyzer": {
				Name:        "security-analyzer",
				Description: "Analyzes security issues",
				Disable:     false,
			},
			"disabled-agent": {
				Name:        "disabled-agent",
				Description: "This agent is disabled",
				Disable:     true,
			},
		},
	}

	return &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(),
		RequesterFactory: clientFactory,
		AgentStore:       agentStore,
	}
}

func TestPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	// Create plugin and init
	p := &Plugin{}
	err := p.Init(state, hub)
	require.NoError(t, err)

	// Verify tools were registered (2 active agents, 1 disabled)
	tools := state.ToolStore.Tools()
	assert.Equal(t, 2, len(tools), "should register 2 tools for 2 active agents")

	// Verify agent tools are registered with correct names
	_, err = state.ToolStore.GetTool("agent_code_reviewer")
	assert.NoError(t, err, "agent_code_reviewer tool should be registered")

	_, err = state.ToolStore.GetTool("agent_security_analyzer")
	assert.NoError(t, err, "agent_security_analyzer tool should be registered")

	// Verify disabled agent is not registered
	_, err = state.ToolStore.GetTool("agent_disabled_agent")
	assert.Error(t, err, "disabled agent should not be registered")
}

func TestPlugin_InitToolDefinition(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent_code_reviewer")
	require.NoError(t, err)

	// Check tool definition
	def := tool.Definition()
	assert.Equal(t, "agent_code_reviewer", def.Function.Name)

	// Check description
	desc := def.Function.Description
	assert.Equal(t, "Performs code reviews", desc)
}

func TestPlugin_InitToolIsAgent(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent_code_reviewer")
	require.NoError(t, err)

	// Check that IsAgent is set to true
	assert.True(t, tool.IsAgent, "agent tool should be marked as agent")
}

func TestPlugin_InitToolExecute(t *testing.T) {
	t.Parallel()

	state := createTestState(t)
	hub := &mockHub{}

	err := (&Plugin{}).Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err := state.ToolStore.GetTool("agent_code_reviewer")
	require.NoError(t, err)

	// Test with nil params
	result := tool.Execute(context.Background(), nil)
	assert.Contains(t, result, "error")

	// Test with empty message
	result = tool.Execute(context.Background(), []byte(`{"message": ""}`))
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "message is required")
}

func TestPlugin_InitNoAgents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	p := printer.GetDefault()
	clientFactory := requester.NewFactory()

	// Create empty agent store
	agentStore := &mockAgentStore{
		agents: map[string]*plugin.Agent{},
	}

	state := &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(),
		RequesterFactory: clientFactory,
		AgentStore:       agentStore,
	}

	hub := &mockHub{}
	plugin := &Plugin{}
	err = plugin.Init(state, hub)
	require.NoError(t, err)

	// Verify no tools were registered
	tools := state.ToolStore.Tools()
	assert.Equal(t, 0, len(tools), "should register 0 tools when no agents available")
}
