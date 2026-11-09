package changes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// Tool provides local_changes operation as LLM tool.
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
	return "local_changes"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Get local Git repository changes. Returns the result of git status, git diff, and git diff --staged commands."
}

// Exec executes the local_changes tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	t.printer.ToolCall(printer.IconSearch, "local_changes", "directory", t.root.Name())

	var result strings.Builder

	// Run git status
	statusCmd := exec.CommandContext(ctx, "git", "status")
	statusCmd.Dir = t.root.Name()
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		t.printer.Error("local_changes: git status failed", "err", err)
		return nil, fmt.Errorf("local_changes: git status failed: %w", err)
	}
	fmt.Fprintf(&result, "# git status\n\n%s\n\n", string(statusOutput))

	// Run git diff
	diffCmd := exec.CommandContext(ctx, "git", "diff")
	diffCmd.Dir = t.root.Name()
	diffOutput, err := diffCmd.CombinedOutput()
	if err != nil {
		t.printer.Error("local_changes: git diff failed", "err", err)
		return nil, fmt.Errorf("local_changes: git diff failed: %w", err)
	}
	fmt.Fprintf(&result, "# git diff\n\n%s\n\n", string(diffOutput))

	// Run git diff --staged
	stagedCmd := exec.CommandContext(ctx, "git", "diff", "--staged")
	stagedCmd.Dir = t.root.Name()
	stagedOutput, err := stagedCmd.CombinedOutput()
	if err != nil {
		t.printer.Error("local_changes: git diff --staged failed", "err", err)
		return nil, fmt.Errorf("local_changes: git diff --staged failed: %w", err)
	}
	fmt.Fprintf(&result, "# git diff --staged\n\n%s", string(stagedOutput))

	return &tools.Response{
		Status: tools.StatusOK,
		Result: result.String(),
	}, nil
}

// Params represents parameters for the local_changes tool.
type Params struct {
	// No parameters needed - uses current directory
}
