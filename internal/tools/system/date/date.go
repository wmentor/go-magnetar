package date

import (
	"context"
	"time"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// SystemDateParams represents parameters for the system_date tool.
type SystemDateParams struct {
	// No parameters needed - returns current date
}

// Tool provides system_date operation as LLM tool.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
	state   *plugin.State
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
		state:   state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "system_date"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Return the current date in the \"Mon Jan _2 15:04:05 MST 2006\" format."
}

// Exec executes the system_date tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *SystemDateParams) (any, error) {
	t.printer.ToolCall(printer.IconClock, "system_date")

	dateStr := time.Now().Format(time.UnixDate)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: dateStr,
	}, nil
}
