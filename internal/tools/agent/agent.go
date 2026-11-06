package agent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	agentcore "github.com/wmentor/go-magnetar/internal/agent"
	"github.com/wmentor/go-magnetar/internal/agent/subagent"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// AgentParams represents parameters for the agent tool.
type AgentParams struct {
	Message string `json:"message" jsonschema_description:"Message to process by the agent"`
}

// Tool provides agent invocation as LLM tool.
type Tool struct {
	agentName string
	state     *plugin.State
}

// NewTool creates a new Tool instance.
func NewTool(agentName string, state *plugin.State) *Tool {
	return &Tool{
		agentName: agentName,
		state:     state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "agent_" + agentcore.NormalizeAgentName(t.agentName)
}

// Description returns the tool description.
func (t *Tool) Description() string {
	agentDef, err := t.state.AgentStore.Get(t.agentName)
	if err != nil {
		return fmt.Sprintf("Invoke agent %s", t.agentName)
	}
	return agentDef.Description
}

// Exec executes the agent tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *AgentParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.state.Printer.ToolCall(printer.IconRobot, "agent", "agent", t.agentName)
	defer t.state.Printer.ToolCall(printer.IconDone, "agent: completed", "agent", t.agentName)

	t.state.Printer.PrintRaw(params.Message)

	if params.Message == "" {
		t.state.Printer.Error("agent: message is required")
		return nil, errors.New("message is required")
	}

	// Create subagent
	sa, err := subagent.New(t.agentName, t.state)
	if err != nil {
		t.state.Printer.Error("agent: failed to create subagent", "agent", t.agentName, "err", err)
		return nil, fmt.Errorf("failed to create subagent %q: %w", t.agentName, err)
	}

	// Execute request
	response, err := sa.Ask(params.Message)
	if err != nil {
		t.state.Printer.Error("agent: failed to ask subagent", "agent", t.agentName, "err", err)
		return nil, fmt.Errorf("failed to ask subagent %q: %w", t.agentName, err)
	}

	t.state.Printer.PrintRaw("agent response: ")
	t.state.Printer.PrintRaw(response)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: response,
	}, nil
}
