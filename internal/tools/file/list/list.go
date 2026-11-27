package list

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// FileListParams represents parameters for the file_list tool.
type FileListParams struct {
	Filter string `json:"filter,omitempty" jsonschema_description:"Glob pattern to match file paths (e.g., '*.go', 'internal/**/*.go', 'src/*.py'). Pattern is matched against relative path from root."` //nolint:lll // ok.
}

// Tool provides file_list operation as LLM tool.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
	root    *os.Root
	state   *plugin.State
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, root *os.Root, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
		root:    root,
		state:   state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "file_list"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "List all files in the current directory tree, optionally filtered by a glob pattern."
}

// Exec executes the file_list tool and returns the list of files.
func (t *Tool) Exec(ctx context.Context, params *FileListParams) (any, error) {
	if t.root == nil {
		t.printer.Error("file_list: root filesystem is not set")
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: root filesystem is not set",
		}, nil
	}

	args := []any{}
	if params != nil && params.Filter != "" {
		args = append(args, "filter", params.Filter)
	}

	t.printer.ToolCall(printer.IconTool, "file_list", args...)

	var results []string

	err := filepath.WalkDir(t.root.Name(), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(t.root.Name(), p)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		base := filepath.Base(p)

		if d.IsDir() {
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if params != nil && params.Filter != "" {
			// Normalize path separators for cross-platform compatibility
			normalizedPath := filepath.ToSlash(relPath)
			matched, err := doublestar.Match(params.Filter, normalizedPath)
			if err != nil {
				t.printer.Error("file_list: invalid filter pattern", "pattern", params.Filter, "err", err)
				return nil
			}
			if !matched {
				return nil
			}
		}

		results = append(results, relPath)
		return nil
	})
	if err != nil {
		t.printer.Error("file_list: failed to walk directory", "path", t.root.Name(), "err", err)
		return nil, fmt.Errorf("file_list: failed to walk directory: %w", err)
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: strings.Join(results, "\n"),
	}, nil
}
