package list

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/archive"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// Tool implements the archive_list tool.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
}

// NewTool creates a new archive_list tool.
func NewTool(cfg *config.Config, p printer.Printer) *Tool {
	return &Tool{
		cfg:     cfg,
		printer: p,
	}
}

// Params defines the parameters for the archive_list tool.
type Params struct {
	Filename string `json:"filename" jsonschema_description:"Path to the archive file"`
	Filter   string `json:"filter,omitempty" jsonschema_description:"Optional filter pattern to match file names (supports glob patterns like *.txt, docs/*.md, etc.)"` //nolint:lll // ok.
}

// Name returns the name of the tool.
func (t *Tool) Name() string {
	return "archive_list"
}

// Description returns the description of the tool.
func (t *Tool) Description() string {
	return "List all files within an archive (ZIP, etc.) with their metadata. Supports filtering by file name pattern."
}

// Exec executes the archive_list tool.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	args := []any{"filename", params.Filename}
	if params.Filter != "" {
		args = append(args, "filter", params.Filter)
	}
	t.printer.ToolCall(printer.IconRead, "archive_list", args...)

	ext := strings.ToLower(filepath.Ext(params.Filename))

	arch, err := archive.GetArchive(ext)
	if err != nil {
		t.printer.Error("archive_list: unsupported archive format", "ext", ext, "err", err)
		return nil, errors.Wrap(err, "unsupported archive format")
	}

	files, err := arch.List(params.Filename)
	if err != nil {
		t.printer.Error("archive_list: failed to list archive", "filename", params.Filename, "err", err)
		return nil, errors.Wrap(err, "failed to list archive")
	}

	// Apply filter if specified
	if params.Filter != "" {
		files = filterFiles(files, params.Filter)
	}

	// Convert to JSON for display
	jsonData, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		t.printer.Error("archive_list: failed to marshal file list", "err", err)
		return nil, errors.Wrap(err, "failed to marshal file list")
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: string(jsonData),
	}, nil
}

// filterFiles filters the file list based on the provided pattern.
// The pattern supports glob syntax (*, ?, [abc], [^abc]).
func filterFiles(files []archive.FileInfo, pattern string) []archive.FileInfo {
	var filtered []archive.FileInfo

	for _, file := range files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Match the file name against the pattern
		matched, err := filepath.Match(pattern, file.Name)
		if err != nil {
			// If pattern is invalid, skip this file
			continue
		}

		if matched {
			filtered = append(filtered, file)
			continue
		}

		// Also try matching just the base name (without directory path)
		baseName := filepath.Base(file.Name)
		if matched, _ := filepath.Match(pattern, baseName); matched {
			filtered = append(filtered, file)
		}
	}

	return filtered
}
