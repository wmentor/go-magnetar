package lastresponse

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/response"
)

type mockHub struct {
	tools      []plugin.LLMTool
	commands   []plugin.ChatCommand
	preproc    []plugin.PreprocessorFunc
	registered bool
}

func (h *mockHub) RegisterTool(tool plugin.LLMTool) {
	h.tools = append(h.tools, tool)
	h.registered = true
}

func (h *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	h.commands = append(h.commands, cmd)
}

func (h *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	h.preproc = append(h.preproc, fn)
}

func (h *mockHub) Go(f func(ctx context.Context)) {
}

func (h *mockHub) Stop() {
}

func TestPluginInit(t *testing.T) {
	t.Parallel()

	s := &plugin.State{
		ResponseStore: response.New(plugin.DefaultResponseStoreSize),
	}
	hub := &mockHub{}

	p := &Plugin{}
	err := p.Init(s, hub)
	assert.NoError(t, err)
	assert.True(t, hub.registered, "tool should be registered")

	// Verify tool is registered
	assert.NotEmpty(t, hub.tools)
	def := hub.tools[0].Definition()
	assert.Equal(t, openai.ToolTypeFunction, def.Type)
	assert.Equal(t, "last_response", def.Function.Name)
}

func TestPluginToolDefinition(t *testing.T) {
	t.Parallel()

	s := &plugin.State{
		ResponseStore: response.New(plugin.DefaultResponseStoreSize),
	}
	hub := &mockHub{}

	p := &Plugin{}
	err := p.Init(s, hub)
	assert.NoError(t, err)

	// Check tool definition
	def := hub.tools[0].Definition()
	assert.Equal(t, openai.ToolTypeFunction, def.Type)
	assert.Equal(t, "last_response", def.Function.Name)

	// Check description mentions user explicitly
	desc := def.Function.Description
	assert.Contains(t, desc, "explicitly requested")
	assert.Contains(t, desc, "What was your last answer?")
}

func TestPluginExecute(t *testing.T) {
	t.Parallel()

	s := &plugin.State{
		ResponseStore: response.New(plugin.DefaultResponseStoreSize),
	}
	hub := &mockHub{}

	p := &Plugin{}
	err := p.Init(s, hub)
	assert.NoError(t, err)

	// Push some responses
	s.ResponseStore.Push("First response")
	s.ResponseStore.Push("Second response")
	s.ResponseStore.Push("Third response")

	// Test index 0 (most recent)
	result, err := hub.tools[0].Execute(context.Background(), `{"index": 0}`)
	assert.NoError(t, err)
	assert.Contains(t, result, "Third response")
	assert.Contains(t, result, "index 0")

	// Test index 1 (second most recent)
	result, err = hub.tools[0].Execute(context.Background(), `{"index": 1}`)
	assert.NoError(t, err)
	assert.Contains(t, result, "Second response")
	assert.Contains(t, result, "index 1")

	// Test index 2 (third most recent)
	result, err = hub.tools[0].Execute(context.Background(), `{"index": 2}`)
	assert.NoError(t, err)
	assert.Contains(t, result, "First response")
	assert.Contains(t, result, "index 2")
}

func TestPluginExecuteInvalidIndex(t *testing.T) {
	t.Parallel()

	s := &plugin.State{
		ResponseStore: response.New(plugin.DefaultResponseStoreSize),
	}
	hub := &mockHub{}

	p := &Plugin{}
	err := p.Init(s, hub)
	assert.NoError(t, err)

	// Test negative index
	result, err := hub.tools[0].Execute(context.Background(), `{"index": -1}`)
	assert.Error(t, err)
	assert.Equal(t, "", result)

	// Test index out of bounds (empty store)
	result, err = hub.tools[0].Execute(context.Background(), `{"index": 0}`)
	assert.Error(t, err)
	assert.Equal(t, "", result)
}

func TestPluginExecuteInvalidJSON(t *testing.T) {
	t.Parallel()

	s := &plugin.State{
		ResponseStore: response.New(plugin.DefaultResponseStoreSize),
	}
	hub := &mockHub{}

	p := &Plugin{}
	err := p.Init(s, hub)
	assert.NoError(t, err)

	// Test invalid JSON
	result, err := hub.tools[0].Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Equal(t, "", result)
}
