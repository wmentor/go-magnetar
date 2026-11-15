package ragplugin

import (
	"context"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester/testdata"
	"github.com/wmentor/go-magnetar/internal/mask"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

// mockStore implements rag.Store interface for testing
type mockStore struct {
	nameFunc   func() string
	searchFunc func(query string) string
	saveFunc   func(content string, prepend string, part int) bool
}

func (m *mockStore) Name() string {
	if m.nameFunc != nil {
		return m.nameFunc()
	}
	return "mock"
}

func (m *mockStore) RagSearch(query string) string {
	if m.searchFunc != nil {
		return m.searchFunc(query)
	}
	return ""
}

func (m *mockStore) RagSave(content string, prepend string, part int) bool {
	if m.saveFunc != nil {
		return m.saveFunc(content, prepend, part)
	}
	return true
}

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

// createTestState creates a test state with default configuration
func createTestState(t *testing.T, ragEnabled bool) *plugin.State {
	t.Helper()

	// Create temp file for config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"

	ragEnableStr := "false"
	if ragEnabled {
		ragEnableStr = "true"
	}

	cfg := `version: "1.0"
rag:
  enable: ` + ragEnableStr + `
  store:
    type: chromem
  llm:
    base_url: "https://api.openai.com/v1"
    api_key: "sk-test-key"
    model: "text-embedding-3-small"
    vector_size: 1536
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)

	return &plugin.State{
		Config:           cfgLoaded,
		Printer:          p,
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}
}

// createMockStore creates a mock store for testing
func createMockStore() rag.Store {
	return &mockStore{}
}

func TestPlugin(t *testing.T) {
	t.Parallel()

	// Create test state with RAG enabled
	state := createTestState(t, true)
	hub := &mockHub{}

	// Create plugin instance
	p := &Plugin{}

	// Initialize the plugin
	err := p.Init(state, hub)
	require.NoError(t, err)

	// Verify the tool was registered
	tool, err := state.ToolStore.GetTool("rag_search")
	require.NoError(t, err, "rag_search tool should be registered when RAG is enabled")

	// Verify tool name
	assert.Equal(t, "rag_search", tool.Definition().Function.Name)

	// Verify tool description
	desc := tool.Definition().Function.Description
	assert.Contains(t, desc, "Search")
	assert.Contains(t, desc, "knowledge base")

	// Create test state with RAG disabled
	state = createTestState(t, false)

	// Create plugin instance
	p = &Plugin{}

	// Initialize the plugin
	err = p.Init(state, hub)
	require.NoError(t, err)

	// Verify no tool was registered when RAG is disabled
	_, err = state.ToolStore.GetTool("rag_search")
	require.Error(t, err, "rag_search tool should not be registered when RAG is disabled")

	// Create temp file for config with empty LLM base URL
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"

	cfg := `version: "1.0"
rag:
  enable: true
  llm:
    base_url: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	cfgLoaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Create printer and request factory for tests
	clientFactory := testdata.NewClientFactory(t)
	maskService := mask.NewService(cfgLoaded)

	state = &plugin.State{
		Config:           cfgLoaded,
		Printer:          printer.GetDefault(),
		ToolStore:        tools.NewStore(maskService),
		RequesterFactory: clientFactory,
	}

	hub = &mockHub{}

	// Create plugin instance
	pPlugin := &Plugin{}

	// Initialize the plugin
	err = pPlugin.Init(state, hub)
	require.NoError(t, err)

	// Verify no tool was registered when LLM base URL is empty
	_, err = state.ToolStore.GetTool("rag_search")
	require.Error(t, err, "rag_search tool should not be registered when LLM base URL is empty")

	// Create test state with valid LLM base URL
	state = createTestState(t, true)

	// Create plugin instance
	p = &Plugin{}

	// Initialize the plugin
	err = p.Init(state, hub)
	require.NoError(t, err)

	// Verify the tool was registered
	tool, err = state.ToolStore.GetTool("rag_search")
	require.NoError(t, err, "rag_search tool should be registered when LLM base URL is valid")

	// Verify tool details
	assert.Equal(t, "rag_search", tool.Definition().Function.Name)
	assert.Contains(t, tool.Definition().Function.Description, "Search")

	// Create test state with RAG enabled
	state = createTestState(t, true)

	// Create plugin instance
	p = &Plugin{}

	// Initialize the plugin
	err = p.Init(state, hub)
	require.NoError(t, err)

	// Verify the tool was registered
	tool, err = state.ToolStore.GetTool("rag_search")
	require.NoError(t, err)

	// Verify tool can be executed (basic test)
	// Note: This won't actually execute the tool since it requires a real store
	// but it verifies the tool is properly registered
	assert.NotNil(t, tool)

	// Create test state with RAG enabled
	state = createTestState(t, true)
	hub = &mockHub{}

	// Create plugin instance
	p = &Plugin{}

	// Initialize the plugin
	err = p.Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err = state.ToolStore.GetTool("rag_search")
	require.NoError(t, err)

	// Check tool definition
	def := tool.Definition()
	assert.Equal(t, "rag_search", def.Function.Name)

	// Check description
	desc = def.Function.Description
	assert.Contains(t, desc, "Search")
	assert.Contains(t, desc, "knowledge base")

	// Create test state with RAG enabled
	state = createTestState(t, true)

	// Create plugin instance
	p = &Plugin{}

	// Initialize the plugin
	err = p.Init(state, hub)
	require.NoError(t, err)

	// Get the registered tool
	tool, err = state.ToolStore.GetTool("rag_search")
	require.NoError(t, err)

	// Check tool parameters
	def = tool.Definition()
	assert.NotNil(t, def.Function.Parameters)

	// Parameters is a *jsonschema.Schema
	schema := def.Function.Parameters.(*jsonschema.Schema)
	assert.NotNil(t, schema.Properties)

	// Verify parameters contain 'query' field
	queryParam, ok := schema.Properties.Get("query")
	assert.True(t, ok, "schema should have 'query' property")
	assert.NotNil(t, queryParam, "query parameter should not be nil")
}

func TestRagSearchCommand(t *testing.T) {
	t.Parallel()

	// 1. When RAG is enabled, command should be registered in hub
	state := createTestState(t, true)
	store := &mockStore{
		searchFunc: func(query string) string {
			if query == "test query" {
				return "search result for test query"
			}
			return ""
		},
	}
	state.VectorStore = store
	hub := &mockHub{}
	p := &Plugin{}

	err := p.Init(state, hub)
	require.NoError(t, err)

	require.Len(t, hub.chatCommands, 1, "rag.search command should be registered")
	cmd := hub.chatCommands[0]
	assert.Equal(t, "rag.search", cmd.Name)
	assert.Empty(t, cmd.Aliases)
	assert.NotEmpty(t, cmd.Help)

	// 2. Command execution with empty args returns error
	resp, err := cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		Args: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage: /rag.search <search_text>")
	assert.Nil(t, resp)

	// 3. Command execution with whitespace returns error
	resp, err = cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		Args: "   ",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage: /rag.search <search_text>")
	assert.Nil(t, resp)

	// 4. Command execution with valid query returns search results in Stdout
	resp, err = cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		Args: "test query",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "search result for test query", resp.Stdout)
	assert.Empty(t, resp.Content)

	// 5. Command execution with query yielding no results
	resp, err = cmd.Execute(context.Background(), &plugin.ChatCommandInput{
		Args: "nonexistent",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "no relevant information found", resp.Stdout)

	// 6. When RAG is disabled, command should NOT be registered
	disabledState := createTestState(t, false)
	disabledHub := &mockHub{}
	disabledP := &Plugin{}
	err = disabledP.Init(disabledState, disabledHub)
	require.NoError(t, err)
	assert.Empty(t, disabledHub.chatCommands, "rag.search command should not be registered when RAG is disabled")

	// 7. When tool is not initialized, executeSearch returns error
	uninitP := &Plugin{}
	resp, err = uninitP.executeSearch(context.Background(), &plugin.ChatCommandInput{Args: "query"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool is not initialized")
	assert.Nil(t, resp)
}
