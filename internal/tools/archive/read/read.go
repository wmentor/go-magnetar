package read

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/archive"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// Tool implements the archive_read tool.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
}

// NewTool creates a new archive_read tool.
func NewTool(cfg *config.Config, p printer.Printer) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
	}
}

// Params defines the parameters for the archive_read tool.
type Params struct {
	ArchivePath string `json:"archive_path" jsonschema_description:"Path to the archive file"`
	FilePath    string `json:"file_path" jsonschema_description:"Path to the file within the archive"`
}

// Name returns the name of the tool.
func (t *Tool) Name() string {
	return "archive_read"
}

// Description returns the description of the tool.
func (t *Tool) Description() string {
	return "Read the contents of a specific file within an archive (ZIP, etc.)"
}

// Exec executes the archive_read tool.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconRead, "archive_read", "archive", params.ArchivePath, "file", params.FilePath)

	ext := strings.ToLower(filepath.Ext(params.ArchivePath))

	arch, err := archive.GetArchive(ext)
	if err != nil {
		t.printer.Error("archive_read: unsupported archive format", "ext", ext, "err", err)
		return nil, errors.Wrap(err, "unsupported archive format")
	}

	data, err := arch.ReadFile(params.ArchivePath, params.FilePath)
	if err != nil {
		t.printer.Error("archive_read: failed to read file from archive", "archive", params.ArchivePath, "file", params.FilePath, "err", err)
		return nil, errors.Wrap(err, "failed to read file from archive")
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: string(data),
	}, nil
}
