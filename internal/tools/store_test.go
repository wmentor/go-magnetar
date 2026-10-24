package tools

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestNewStore(t *testing.T) {
	t.Parallel()

	store := NewStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.tools)
	assert.Len(t, store.Tools(), 0)
}

func TestStore_Register_Success(t *testing.T) {
	t.Parallel()

	store := NewStore()

	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	store.Register(
		"test_tool",
		"Test tool description",
		func(ctx context.Context, req *TestRequest) (any, error) {
			return map[string]any{
				"message": "Hello " + req.Name,
				"age":     req.Age,
			}, nil
		},
	)

	assert.Len(t, store.Tools(), 1)

	tool, err := store.GetTool("test_tool")
	assert.NoError(t, err)
	assert.Equal(t, "test_tool", tool.Name)
	assert.NotEmpty(t, tool.Definition)
	assert.NotNil(t, tool.Execute)
}

func TestStore_Register_DuplicateToolName(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"test_tool",
		"First description",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "first", nil
		},
	)

	// Second registration with the same name should overwrite
	store.Register(
		"test_tool",
		"Second description",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "second", nil
		},
	)

	assert.Len(t, store.Tools(), 1)

	tool, err := store.GetTool("test_tool")
	assert.NoError(t, err)
	assert.Equal(t, "test_tool", tool.Name)
	assert.NotNil(t, tool.Definition)
}

func TestStore_GetTool_Success(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"existing_tool",
		"Description",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return nil, nil
		},
	)

	tool, err := store.GetTool("existing_tool")
	assert.NoError(t, err)
	assert.Equal(t, "existing_tool", tool.Name)
	assert.NotEmpty(t, tool.Definition)
}

func TestStore_GetTool_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	tool, err := store.GetTool("nonexistent_tool")
	assert.ErrorIs(t, err, ErrToolNotFound)
	assert.Equal(t, Tool{}, tool)
}

func TestStore_CloneWith_Success(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"tool1",
		"Tool 1",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool1", nil
		},
	)

	store.Register(
		"tool2",
		"Tool 2",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool2", nil
		},
	)

	store.Register(
		"tool3",
		"Tool 3",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool3", nil
		},
	)

	// Clone with specific tools
	cloned, err := store.CloneWith("tool1", "tool2")
	assert.NoError(t, err)
	assert.NotNil(t, cloned)

	// Check cloned store has only the specified tools
	clonedTools := cloned.Tools()
	assert.Len(t, clonedTools, 2)
	for _, tool := range clonedTools {
		assert.NotEmpty(t, tool.Name)
	}

	// Verify original store is unchanged
	originalTools := store.Tools()
	assert.Len(t, originalTools, 3)
}

func TestStore_CloneWith_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"tool1",
		"Tool 1",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool1", nil
		},
	)

	// Try to clone with non-existent tool
	cloned, err := store.CloneWith("tool1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tools not found")
	assert.Nil(t, cloned)

	// Try with all non-existent tools
	cloned, err = store.CloneWith("nonexistent1", "nonexistent2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tools not found")
	assert.Nil(t, cloned)
}

func TestStore_CloneWith_EmptyList(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"tool1",
		"Tool 1",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool1", nil
		},
	)

	// Clone with empty list should return empty store
	cloned, err := store.CloneWith()
	assert.NoError(t, err)
	assert.NotNil(t, cloned)

	clonedTools := cloned.Tools()
	assert.Len(t, clonedTools, 0)
}

func TestTool_GetName(t *testing.T) {
	t.Parallel()

	tool := Tool{
		Name: "test_tool",
		Definition: func() openai.Tool {
			return openai.Tool{}
		},
		Execute: func(ctx context.Context, rawJSON []byte) string {
			return "{}"
		},
	}

	assert.Equal(t, "test_tool", tool.GetName())
}

func TestStore_CloneWith_Isolation(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"tool1",
		"Tool 1",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool1", nil
		},
	)

	cloned, err := store.CloneWith("tool1")
	assert.NoError(t, err)

	// Register a tool in cloned store
	cloned.Register(
		"cloned_tool",
		"Cloned tool",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "cloned", nil
		},
	)

	// Original store should not have the cloned tool
	_, err = store.GetTool("cloned_tool")
	assert.ErrorIs(t, err, ErrToolNotFound)

	// Clone should not have original-only tools
	cloned2, err := cloned.CloneWith("tool1")
	assert.NoError(t, err)
	cloned2Tools := cloned2.Tools()
	assert.Len(t, cloned2Tools, 1)
}

func TestStore_Tools(t *testing.T) {
	t.Parallel()

	store := NewStore()

	store.Register(
		"tool1",
		"Tool 1",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool1", nil
		},
	)

	store.Register(
		"tool2",
		"Tool 2",
		func(ctx context.Context, req *map[string]any) (any, error) {
			return "tool2", nil
		},
	)

	allTools := store.Tools()
	assert.Len(t, allTools, 2)
}

func TestStore_Tools_Concurrent(t *testing.T) {
	t.Parallel()

	store := NewStore()

	// Register some tools
	for i := range 10 {
		store.Register(
			"tool"+string(rune('0'+i)),
			"Tool",
			func(ctx context.Context, req *map[string]any) (any, error) {
				return i, nil
			},
		)
	}

	// Concurrent reads
	done := make(chan bool)
	for range 10 {
		go func() {
			tools := store.Tools()
			assert.Len(t, tools, 10)
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := NewStore()

	// Concurrent register and read
	done := make(chan bool, 100)

	for i := range 50 {
		go func() {
			store.Register(
				"tool"+string(rune('0'+i)),
				"Tool",
				func(ctx context.Context, req *map[string]any) (any, error) {
					return i, nil
				},
			)
			done <- true
		}()
	}

	for range 50 {
		go func() {
			_ = store.Tools()
			done <- true
		}()
	}

	for range 100 {
		<-done
	}

	assert.Len(t, store.Tools(), 50)
}
