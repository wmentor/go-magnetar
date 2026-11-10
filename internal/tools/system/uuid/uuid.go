package uuid

import (
	"context"
	"os"

	stduuid "uuid"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// SystemUUIDParams represents parameters for the system_uuid tool.
type SystemUUIDParams struct {
	// No parameters needed - generates a UUID v4.
}

// Tool provides system_uuid operation as an LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	root          *os.Root
	state         *plugin.State
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, root *os.Root, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		root:          root,
		state:         state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "system_uuid"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Generate a cryptographically secure UUID version 4 identifier."
}

// Exec executes the system_uuid tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *SystemUUIDParams) (any, error) {
	t.printer.ToolCall(printer.IconTool, "system_uuid")

	return &tools.Response{
		Status: tools.StatusOK,
		Result: stduuid.NewV4().String(),
	}, nil
}
