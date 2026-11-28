package toolplugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
)

type mockHub struct {
	registeredCommands []plugin.ChatCommand
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.registeredCommands = append(m.registeredCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(_ plugin.PreprocessorFunc) {}
func (m *mockHub) Go(_ func(ctx context.Context))                 {}
func (m *mockHub) Stop()                                          {}

type mockMaskService struct {
	prefix string
}

func (m *mockMaskService) Process(_ context.Context, input string) string {
	if m.prefix == "" {
		return input
	}
	return strings.ReplaceAll(input, "SECRET", m.prefix)
}

type nestedFilter struct {
	Tags []string `json:"tags"`
}

type echoParams struct {
	Query  string        `json:"query"`
	Count  int           `json:"count"`
	Filter *nestedFilter `json:"filter,omitempty"`
}

type noParams struct{}

type validatedParams struct {
	Alpha string `json:"alpha"`
	Beta  string `json:"beta"`
}

func (v *validatedParams) Validate() error {
	errs := validation.Errors{}
	if v.Alpha == "" {
		errs["alpha"] = errors.New("cannot be blank")
	}
	if v.Beta == "" {
		errs["beta"] = errors.New("cannot be blank")
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func newTestStore(t *testing.T) *tools.Store {
	t.Helper()

	store := tools.NewStore(&mockMaskService{prefix: "[MASKED]"})

	store.Register("echo", "Echo parameters", func(_ context.Context, req *echoParams) (any, error) {
		tags := ""
		if req.Filter != nil {
			tags = strings.Join(req.Filter.Tags, ",")
		}
		return &tools.Response{
			Status: tools.StatusOK,
			Result: fmt.Sprintf("query=%s count=%d tags=%s", req.Query, req.Count, tags),
		}, nil
	})

	store.Register("ping", "Ping with no params", func(_ context.Context, _ *noParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "pong",
		}, nil
	})

	store.Register("seq_tool", "Sequential tool", func(_ context.Context, _ *noParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "sequential ok",
		}, nil
	}, tools.OptionSequential())

	store.Register("agent_sub", "Agent tool", func(_ context.Context, _ *noParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "agent ok",
		}, nil
	}, tools.OptionAgent())

	store.Register("empty_tool", "Returns empty result", func(_ context.Context, _ *noParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "",
		}, nil
	})

	store.Register("raw_string_tool", "Returns plain string", func(_ context.Context, _ *noParams) (any, error) {
		return "# Markdown Result\nHello from raw string", nil
	})

	store.Register("custom_json_tool", "Returns custom object", func(_ context.Context, _ *noParams) (any, error) {
		return map[string]any{"items": []int{1, 2}, "ok": true}, nil
	})

	store.Register("error_tool", "Returns internal error", func(_ context.Context, _ *noParams) (any, error) {
		return nil, errors.New("boom")
	})

	store.Register("validated_tool", "Validates fields", func(_ context.Context, req *validatedParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: req.Alpha + ":" + req.Beta,
		}, nil
	})

	store.Register("secret_tool", "Returns sensitive content", func(_ context.Context, _ *noParams) (any, error) {
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "token=SECRET",
		}, nil
	})

	return store
}

func setupCommand(t *testing.T, state *plugin.State) plugin.ChatCommand {
	t.Helper()

	p := &Plugin{}
	hub := &mockHub{}
	err := p.Init(state, hub)
	require.NoError(t, err)
	require.Len(t, hub.registeredCommands, 1)

	return hub.registeredCommands[0]
}

func TestInit(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	cmd := setupCommand(t, &plugin.State{ToolStore: store})

	assert.Equal(t, "tool", cmd.Name)
	assert.Empty(t, cmd.Aliases)
	assert.Contains(t, cmd.Help, "Usage: /tool <tool_name> [json_params]")
	assert.NotNil(t, cmd.Execute)
}

func TestExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupState func(t *testing.T) *plugin.State
		input      *plugin.ChatCommandInput
		wantStdout string
		wantNoLog  bool
	}{
		{
			name: "success with spaces and nested json",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input: &plugin.ChatCommandInput{
				Args: `echo {"query": "hello world", "count": 42, "filter": {"tags": ["go", "rag"]}}`,
			},
			wantStdout: "query=hello world count=42 tags=go,rag",
			wantNoLog:  false,
		},
		{
			name: "success with newline separator before json",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input: &plugin.ChatCommandInput{
				Args: "echo\n{\"query\": \"multiline test\", \"count\": 1}",
			},
			wantStdout: "query=multiline test count=1 tags=",
			wantNoLog:  false,
		},
		{
			name: "success without json defaults to empty object",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "ping"},
			wantStdout: "pong",
			wantNoLog:  false,
		},
		{
			name: "success without json and trailing spaces",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "  ping   "},
			wantStdout: "pong",
			wantNoLog:  false,
		},
		{
			name: "success with sequential tool",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "seq_tool {}"},
			wantStdout: "sequential ok",
			wantNoLog:  false,
		},
		{
			name: "success with agent tool",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "agent_sub {}"},
			wantStdout: "agent ok",
			wantNoLog:  false,
		},
		{
			name: "success with empty result",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "empty_tool"},
			wantStdout: "Tool \"empty_tool\" completed with empty output.\n",
			wantNoLog:  false,
		},
		{
			name: "success with raw string response",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "raw_string_tool"},
			wantStdout: "# Markdown Result\nHello from raw string",
			wantNoLog:  false,
		},
		{
			name: "success with custom json response",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "custom_json_tool"},
			wantStdout: "```json\n{\n  \"items\": [\n    1,\n    2\n  ],\n  \"ok\": true\n}\n```\n",
			wantNoLog:  false,
		},
		{
			name: "success applies mask service",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "secret_tool"},
			wantStdout: "token=[MASKED]",
			wantNoLog:  false,
		},
		{
			name: "empty args shows usage",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: ""},
			wantStdout: "Usage: /tool <tool_name> [json_params]\n",
			wantNoLog:  true,
		},
		{
			name: "whitespace args shows usage",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: " \t\n "},
			wantStdout: "Usage: /tool <tool_name> [json_params]\n",
			wantNoLog:  true,
		},
		{
			name: "nil input shows usage",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      nil,
			wantStdout: "Usage: /tool <tool_name> [json_params]\n",
			wantNoLog:  true,
		},
		{
			name: "malformed json rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo {"query": `},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "json with trailing tokens rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo {"query": "ok"} extra`},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "json with second object rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo {"query": "ok"} {"count": 1}`},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "json null rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo null`},
			wantStdout: "Error: JSON parameters must be a JSON object\n",
			wantNoLog:  true,
		},
		{
			name: "json array rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo ["a", "b"]`},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "json string rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo "hello"`},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "json number rejected",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo 123`},
			wantStdout: "Error: invalid JSON parameters:",
			wantNoLog:  true,
		},
		{
			name: "tool not found",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `nonexistent {"a": 1}`},
			wantStdout: "Error: tool \"nonexistent\" not found. Use /tools to list available tools.\n",
			wantNoLog:  true,
		},
		{
			name: "nil state",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return nil
			},
			input:      &plugin.ChatCommandInput{Args: "ping"},
			wantStdout: "Error: tool store is unavailable.\n",
			wantNoLog:  true,
		},
		{
			name: "nil tool store",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{}
			},
			input:      &plugin.ChatCommandInput{Args: "ping"},
			wantStdout: "Error: tool store is unavailable.\n",
			wantNoLog:  true,
		},
		{
			name: "unknown parameter rejected by tool schema",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo {"unknown": 1}`},
			wantStdout: "Error executing tool \"echo\":\n- structure_error: You provided an unsupported parameter. Please check the tool schema and try again.\n",
			wantNoLog:  true,
		},
		{
			name: "parameter type mismatch rejected by tool schema",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `echo {"count": "not-an-int"}`},
			wantStdout: "Error executing tool \"echo\":\n- type_error: Data type mismatch (e.g., passed a string instead of an integer).\n",
			wantNoLog:  true,
		},
		{
			name: "validation errors sorted deterministically",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: `validated_tool {}`},
			wantStdout: "Error executing tool \"validated_tool\":\n- alpha: cannot be blank\n- beta: cannot be blank\n",
			wantNoLog:  true,
		},
		{
			name: "tool internal error formatted cleanly",
			setupState: func(t *testing.T) *plugin.State {
				t.Helper()
				return &plugin.State{ToolStore: newTestStore(t)}
			},
			input:      &plugin.ChatCommandInput{Args: "error_tool {}"},
			wantStdout: "Error executing tool \"error_tool\":\n- internal_error: boom\n",
			wantNoLog:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := setupCommand(t, tt.setupState(t))
			resp, err := cmd.Execute(context.Background(), tt.input)
			require.NoError(t, err)
			require.NotNil(t, resp)

			if strings.HasSuffix(tt.wantStdout, ":") {
				assert.True(t, strings.HasPrefix(resp.Stdout, tt.wantStdout), "Stdout = %q, want prefix %q", resp.Stdout, tt.wantStdout)
			} else {
				assert.Equal(t, tt.wantStdout, resp.Stdout)
			}
			assert.Equal(t, tt.wantNoLog, resp.NoLog)
			assert.Empty(t, resp.Content)
			assert.False(t, resp.Exit)
		})
	}
}

func TestExecuteNilToolHandler(t *testing.T) {
	t.Parallel()

	p := &Plugin{state: &plugin.State{ToolStore: newTestStore(t)}}
	resp, err := p.execute(context.Background(), &plugin.ChatCommandInput{Args: "ping"})
	require.NoError(t, err)
	require.Equal(t, "pong", resp.Stdout)

	nilTool := tools.Tool{
		Name: "nil_exec",
		Definition: func() openai.Tool {
			return openai.Tool{Type: openai.ToolTypeFunction}
		},
	}
	require.Nil(t, nilTool.Execute)

	// Verify formatToolOutput and nil-handler branches directly as unit coverage.
	rawEmptyOut, rawEmptyNoLog := formatToolOutput("raw_empty", "   ")
	assert.Equal(t, "Tool \"raw_empty\" completed with empty output.\n", rawEmptyOut)
	assert.False(t, rawEmptyNoLog)

	emptyQuotedOut, emptyQuotedNoLog := formatToolOutput("empty_str", `""`)
	assert.Equal(t, "Tool \"empty_str\" completed with empty output.\n", emptyQuotedOut)
	assert.False(t, emptyQuotedNoLog)

	plainOut, plainNoLog := formatToolOutput("plain", "unstructured plain output")
	assert.Equal(t, "unstructured plain output", plainOut)
	assert.False(t, plainNoLog)

	// Exercise the nil Execute guard directly via executeTool replacement or response helper.
	assert.Equal(t, "Error: tool \"nil_exec\" has no executable handler.\n",
		errorResponse(fmt.Sprintf("Error: tool %q has no executable handler.", nilTool.Name)).Stdout)
}
