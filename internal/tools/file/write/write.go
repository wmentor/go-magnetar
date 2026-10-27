package write

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// FileWriteParams represents parameters for the file_write tool.
type FileWriteParams struct {
	Filename string `json:"filename" jsonschema_description:"Path where to write the file"`
	Content  string `json:"content" jsonschema_description:"Content to write to the file"`
}

// Tool provides file_write operation as LLM tool.
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
	return "file_write"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Write content to a file. Supports creating directories recursively if they don't exist."
}

// Exec executes the file_write tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *FileWriteParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSave, "file_write", "file", params.Filename)

	if t.root == nil {
		t.printer.Error("file_write: root filesystem is not set", "file", params.Filename)
		return nil, errors.New("root filesystem is not set")
	}

	if t.state.ReadOnly {
		t.printer.Error("file_write: blocked by read-only mode", "file", params.Filename)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: read-only mode is enabled",
		}, nil
	}

	// Create directories if they don't exist
	dir := filepath.Dir(params.Filename)
	if err := t.root.MkdirAll(dir, 0o755); err != nil {
		t.printer.Error("file_write: failed to create directories", "file", params.Filename, "dir", dir, "err", err)
		return nil, fmt.Errorf("file_write: failed to create directories: %w", err)
	}

	// Check if the file extension is handled by a codec (binary/structured formats)
	ext := strings.ToLower(filepath.Ext(params.Filename))
	if _, err := codec.GetCodec(ext); err == nil {
		// For codec-handled formats, write directly without validation
		// The codec will handle the file during read operations
		if err := t.root.WriteFile(params.Filename, []byte(params.Content), 0o644); err != nil {
			t.printer.Error("file_write: failed to write file", "file", params.Filename, "err", err)
			return nil, fmt.Errorf("file_write: failed to write file: %w", err)
		}
	} else {
		// For plain text files, write directly
		if err := t.root.WriteFile(params.Filename, []byte(params.Content), 0o644); err != nil {
			t.printer.Error("file_write: failed to write file", "file", params.Filename, "err", err)
			return nil, fmt.Errorf("file_write: failed to write file: %w", err)
		}
	}

	t.printer.ToolCall(printer.IconDone, "file_write: file written successfully", "file", params.Filename)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: "File written successfully.",
	}, nil
}
