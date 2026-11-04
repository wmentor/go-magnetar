package agent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/agent/subagent"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// AgentParams represents parameters for the agent tool.
type AgentParams struct {
	Agent   string `json:"agent" jsonschema_description:"Agent name to invoke"`
	Message string `json:"message" jsonschema_description:"Message to process by the agent"`
}

// Tool provides agent invocation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	state         *plugin.State
}

// NewTool creates a new Tool instance.
func NewTool(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		state:         state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "agent"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Invoke a subagent by name and process a message with it. Use this tool when you need to delegate a task to a specialized agent."
}

// Exec executes the agent tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *AgentParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconRobot, "agent", "agent", params.Agent, "message", params.Message)

	if params.Agent == "" {
		t.printer.Error("agent: agent name is required", "message", params.Message)
		return nil, errors.New("agent name is required")
	}

	if params.Message == "" {
		t.printer.Error("agent: message is required", "agent", params.Agent)
		return nil, errors.New("message is required")
	}

	// Create subagent
	sa, err := subagent.New(params.Agent, t.state)
	if err != nil {
		t.printer.Error("agent: failed to create subagent", "agent", params.Agent, "err", err)
		return nil, fmt.Errorf("failed to create subagent %q: %w", params.Agent, err)
	}

	// Execute request
	response, err := sa.Ask(params.Message)
	if err != nil {
		t.printer.Error("agent: failed to ask subagent", "agent", params.Agent, "err", err)
		return nil, fmt.Errorf("failed to ask subagent %q: %w", params.Agent, err)
	}

	t.printer.ToolCall(printer.IconDone, "agent: completed", "agent", params.Agent)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: response,
	}, nil
}
