package read

import (
	"bufio"
	"compress/gzip"
	"context"
	"io"
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
	"github.com/wmentor/go-magnetar/internal/types"
)

const maxDecompressedSize = 50 * 1024 * 1024 // 50MB limit to prevent decompression bomb DoS (gosec G110)

// FileReadParams represents parameters for the file_read tool.
type FileReadParams struct {
	Filename string             `json:"filename" jsonschema_description:"Path to the file to read"`
	Limit    *types.FlexibleInt `json:"limit,omitempty" jsonschema_description:"Maximum number of lines to read (optional, 0 = read all)"`
	Offset   *types.FlexibleInt `json:"offset,omitempty" jsonschema_description:"Number of lines to skip from the beginning (optional, 0 = start from beginning)"` //nolint:lll //ok.
}

// Tool provides file_read operation as LLM tool.
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
	return "file_read"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Read the contents of a file by its path. Supports transparent gzip decompression (no manual decompression needed)."
}

// recoverableExtension returns the file extension to use for codec lookup.
// If the filename ends with .gz (case-insensitive), it inspects the extension before .gz.
// Otherwise, it returns the extension of the filename.
func recoverableExtension(filename string) string {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".gz") {
		base := filename[:len(filename)-3]
		return strings.ToLower(filepath.Ext(base))
	}
	return strings.ToLower(filepath.Ext(filename))
}

// isGzip reports whether the reader or filename indicates gzip-compressed content
// by checking for gzip magic bytes (0x1f 0x8b) or a .gz extension.
func isGzip(filename string, br *bufio.Reader) bool {
	header, err := br.Peek(2)
	hasMagic := err == nil && len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b
	hasExt := strings.HasSuffix(strings.ToLower(filename), ".gz")
	return hasMagic || hasExt
}

// readGzip handles reading and processing decompressed gzip content from br.
func (t *Tool) readGzip(filename string, br *bufio.Reader, limit int, offset int) (any, error) {
	ext := recoverableExtension(filename)
	if reader, err := codec.GetCodec(ext); err == nil {
		tmpFile, err := os.CreateTemp("", "magnetar-*"+ext)
		if err != nil {
			t.printer.Error("file_read: failed to create temporary file", "file", filename, "err", err)
			return "", err
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		limitedReader := io.LimitReader(br, maxDecompressedSize+1)
		written, copyErr := io.Copy(tmpFile, limitedReader)
		if closeErr := tmpFile.Close(); closeErr != nil {
			t.printer.Error("file_read: failed to close temporary file", "file", filename, "err", closeErr)
			return "", copyErr
		}

		if copyErr != nil {
			t.printer.Error("file_read: failed to decompress gzip content", "file", filename, "err", copyErr)
			return "", copyErr
		}
		if written > maxDecompressedSize {
			t.printer.Error("file_read: decompressed content exceeds size limit", "file", filename, "max_size", maxDecompressedSize)
			return "", errors.New("decompressed content exceeds size limit")
		}

		text, err := reader.ReadFile(tmpPath)
		if err != nil {
			t.printer.Error("file_read: failed to read file", "file", filename, "err", err)
			return "", err
		}
		return text, nil
	}

	if limit <= 0 && offset <= 0 {
		limitedReader := io.LimitReader(br, maxDecompressedSize+1)
		data, readErr := io.ReadAll(limitedReader)
		if readErr != nil {
			t.printer.Error("file_read: failed to decompress gzip content", "file", filename, "err", readErr)
			return "", readErr
		}
		if int64(len(data)) > maxDecompressedSize {
			t.printer.Error("file_read: decompressed content exceeds size limit", "file", filename, "max_size", maxDecompressedSize)
			return "", errors.New("decompressed content exceeds size limit")
		}
		return tools.Response{
			Status: tools.StatusOK,
			Result: string(data),
		}, nil
	}

	limitedReader := &io.LimitedReader{R: br, N: maxDecompressedSize + 1}
	scanner := bufio.NewScanner(limitedReader)
	var sb strings.Builder
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		if lineCount <= offset {
			continue
		}
		if limit > 0 && lineCount > offset+limit {
			break
		}
		if lineCount > offset+1 {
			sb.WriteByte('\n')
		}
		sb.WriteString(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.printer.Error("file_read: failed to scan file", "file", filename, "err", err)
		return "", err
	}
	if limitedReader.N <= 0 {
		t.printer.Error("file_read: decompressed content exceeds size limit", "file", filename, "max_size", maxDecompressedSize)
		return "", errors.New("decompressed content exceeds size limit")
	}

	return tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}

// Exec executes the file_read tool and returns the file content.
func (t *Tool) Exec(ctx context.Context, params *FileReadParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	args := []any{"filename", params.Filename}

	limit := 0
	offset := 0
	if params.Limit != nil {
		limit = params.Limit.Int()
		args = append(args, "limit", limit)
	}
	if params.Offset != nil {
		offset = params.Offset.Int()
		args = append(args, "offset", offset)
	}

	t.printer.ToolCall(printer.IconTool, "file_read", args...)

	if t.root == nil {
		t.printer.Error("file_read: root filesystem is not set", "file", params.Filename)
		return nil, errors.New("root filesystem is not set")
	}

	f, err := t.root.Open(params.Filename)
	if err != nil {
		t.printer.Error("file_read: failed to open file", "file", params.Filename, "err", err)
		return nil, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	if isGzip(params.Filename, br) {
		gzReader, err := gzip.NewReader(br)
		if err != nil {
			t.printer.Error("file_read: failed to initialize gzip reader", "file", params.Filename, "err", err)
			return nil, err
		}
		defer gzReader.Close()

		br = bufio.NewReader(gzReader)
		return t.readGzip(params.Filename, br, limit, offset)
	}

	ext := strings.ToLower(filepath.Ext(params.Filename))

	if reader, err := codec.GetCodec(ext); err == nil {
		text, err := reader.ReadFile(f.Name())
		if err != nil {
			t.printer.Error("file_read: failed to read file", "file", params.Filename, "err", err)
			return nil, err
		}
		return text, nil
	}

	if limit <= 0 && offset <= 0 {
		data, err := io.ReadAll(br)
		if err != nil {
			t.printer.Error("file_read: failed to read file", "file", params.Filename, "err", err)
			return nil, err
		}
		return tools.Response{
			Status: tools.StatusOK,
			Result: string(data),
		}, nil
	}

	scanner := bufio.NewScanner(br)
	var sb strings.Builder
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		if lineCount <= offset {
			continue
		}
		if limit > 0 && lineCount > offset+limit {
			break
		}
		if lineCount > offset+1 {
			sb.WriteByte('\n')
		}
		sb.WriteString(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.printer.Error("file_read: failed to scan file", "file", params.Filename, "err", err)
		return nil, err
	}

	return tools.Response{
		Status: tools.StatusOK,
		Result: sb.String(),
	}, nil
}
