package grep

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
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
	cfg     *config.Config
	printer printer.Printer
	state   *plugin.State
	useRg   bool // whether to use rg (ripgrep) instead of grep
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
		state:   state,
		useRg:   isRgAvailable(),
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "system_grep"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	if t.useRg {
		return "Execute ripgrep (rg) with PCRE2 regex support: -n (line numbers), -i (case-insensitive), recursive search. " +
			"Supports advanced regex features: lookahead (?=), lookbehind (?<=), backreferences."
	}
	return "Execute system grep command with safe parameters: -n (line numbers), -i (case-insensitive), -r (recursive), -E (extended regex)"
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
	t.printer.ToolCall(printer.IconSearch, "system_grep", extra...)

	// Build the full path inside the sandboxed root
	fullPath := params.Filename
	if t.state != nil && t.state.Root != nil {
		fullPath = filepath.Join(t.state.Root.Name(), params.Filename)
	}

	var cmd []string
	if t.useRg {
		// Use rg (ripgrep) with --pcre2 for better regex compatibility (supports lookahead/lookbehind/backreferences)
		// Flags: -n (line numbers), -i (case-insensitive), --pcre2 (PCRE2 regex syntax)
		// rg is recursive by default
		cmd = []string{"rg", "-n", "-i", "--pcre2", params.Pattern, fullPath}
	} else {
		// Use grep with -n (line numbers), -i (case-insensitive), -r (recursive), and -E (extended regex)
		cmd = []string{"grep", "-n", "-i", "-r", "-E", params.Pattern, fullPath}
	}

	output, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput() //nolint:gosec // it's ok.
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			// grep returns exit code 1 when no match is found
			// rg returns exit code 1 when no match is found
			if code == 1 {
				return &tools.Response{
					Status: tools.StatusOK,
					Result: "No match found",
				}, nil
			}
			// Exit code 2 means syntax error in pattern, file not found, or other error
			// Return stderr output to help diagnose the issue
			if code == 2 {
				return &tools.Response{
					Status: tools.StatusOK,
					Result: fmt.Sprintf("Error (exit code 2): %s", string(output)),
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
