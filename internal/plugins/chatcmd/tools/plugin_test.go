package toolsplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type mockHub struct {
	RegisteredCommands []plugin.ChatCommand
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.RegisteredCommands = append(m.RegisteredCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {}
func (m *mockHub) Go(f func(ctx context.Context))                  {}
func (m *mockHub) Stop()                                           {}

type dummyRequest struct{}

func TestListTools(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		got := listTools(nil)
		assert.Equal(t, "Available tools:\n", got)
	})

	t.Run("empty store", func(t *testing.T) {
		store := tools.NewStore(nil)
		got := listTools(store)
		assert.Equal(t, "Available tools:\n", got)
	})

	t.Run("tools with descriptions and sorting", func(t *testing.T) {
		store := tools.NewStore(nil)
		store.Register("web_search", "Search the web for current information", func(_ context.Context, _ *dummyRequest) (any, error) {
			return nil, nil
		})
		store.Register("file_read", "Read contents from local file", func(_ context.Context, _ *dummyRequest) (any, error) {
			return nil, nil
		})
		store.Register("alpha_tool", "", func(_ context.Context, _ *dummyRequest) (any, error) {
			return nil, nil
		})

		got := listTools(store)
		expected := `Available tools:
  * alpha_tool
  * file_read - Read contents from local file
  * web_search - Search the web for current information
`
		assert.Equal(t, expected, got)
	})
}

func TestExecute(t *testing.T) {
	store := tools.NewStore(nil)
	store.Register("test_tool", "Description for test tool", func(_ context.Context, _ *dummyRequest) (any, error) {
		return nil, nil
	})

	state := &plugin.State{
		ToolStore: store,
	}

	p := &Plugin{}
	hub := &mockHub{}

	err := p.Init(state, hub)
	assert.NoError(t, err)

	assert.Len(t, hub.RegisteredCommands, 1)
	cmd := hub.RegisteredCommands[0]

	assert.Equal(t, "tools", cmd.Name)
	assert.Equal(t, []string{"t"}, cmd.Aliases)
	assert.NotEmpty(t, cmd.Help)

	input := &plugin.ChatCommandInput{}
	resp, err := cmd.Execute(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	expectedStdout := `Available tools:
  * test_tool - Description for test tool
`
	assert.Equal(t, expectedStdout, resp.Stdout)
	assert.Empty(t, resp.Content)
	assert.False(t, resp.Exit)
}
