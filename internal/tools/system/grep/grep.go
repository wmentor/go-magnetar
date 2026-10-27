package grep

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// SystemGrepParams represents parameters for the system_grep tool.
type SystemGrepParams struct {
	Filename string `json:"filename" jsonschema_description:"Path to the file or directory to search"`
	Pattern  string `json:"pattern" jsonschema_description:"Regular expression pattern to search for"`
}

// Tool provides system_grep operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	root          *os.Root
	state         *plugin.State
	useRg         bool // whether to use rg (ripgrep) instead of grep
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, root *os.Root, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		root:          root,
		state:         state,
		useRg:         isRgAvailable(),
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "system_grep"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Execute system grep command with safe parameters: -n (always), -i (case-insensitive), and -r (recursive)"
}

// Exec executes the system_grep tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *SystemGrepParams) (any, error) {
	if params == nil {
		return nil, ErrInvalidParams
	}

	extra := []any{"pattern", params.Pattern}
	if t.useRg {
		extra = append(extra, "ripgrep", "on")
	}
	t.printer.ToolCall(printer.IconTool, "system_grep", extra...)

	// Build the full path inside the sandboxed root
	fullPath := params.Filename
	if t.root != nil {
		fullPath = filepath.Join(t.root.Name(), params.Filename)
	}

	var cmd []string
	if t.useRg {
		// Use rg (ripgrep) with same flags as grep: -n (line numbers), -i (case-insensitive)
		// rg is recursive by default and uses extended regex by default
		cmd = []string{"rg", "-n", "-i", params.Pattern, fullPath}
	} else {
		// Use grep with -n (line numbers), -i (case-insensitive), -r (recursive), and -E (extended regex)
		cmd = []string{"grep", "-n", "-i", "-r", "-E", params.Pattern, fullPath}
	}

	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput() //nolint:gosec // it's ok.
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// grep returns exit code 1 when no match is found
			// rg returns exit code 1 when no match is found
			if code := exitErr.ExitCode(); code == 1 {
				return &tools.Response{
					Status: tools.StatusOK,
					Result: "No match found",
				}, nil
			}
		}
		t.printer.Error("system_grep error", "err", err)
		return nil, fmt.Errorf("system_grep error: %w", err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: string(output),
	}, nil
}

// isRgAvailable checks whether rg (ripgrep) is installed and available in PATH.
func isRgAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}
