package read

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/types"
)

func newTestTools(t *testing.T, yaml string, readOnly bool) *Tool {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	responseStore := response.New(plugin.DefaultResponseStoreSize)

	return New(cfg, nil, nil, nil, &plugin.State{Config: cfg, ReadOnly: readOnly, ResponseStore: responseStore})
}

func gzipCompress(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFileReadGzipCSV(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	csvContent := "name,age\nAlice,30\nBob,25\n"
	gzData := gzipCompress(t, []byte(csvContent))

	if err := rootFS.WriteFile("data.csv.gz", gzData, 0o644); err != nil {
		t.Fatal(err)
	}

	params := &FileReadParams{Filename: "data.csv.gz"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec(data.csv.gz) failed: %v", err)
	}

	var content string
	if resp, ok := result.(tools.Response); ok {
		content = resp.Result
	} else {
		content = result.(string)
	}

	if !strings.Contains(content, "name") || !strings.Contains(content, "Alice") || !strings.Contains(content, "Bob") {
		t.Errorf("Exec(data.csv.gz) unexpected output: %s", content)
	}
}

func TestFileReadGzipPDF(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	pdfPath := filepath.Join("..", "..", "..", "codec", "pdf", "testdata", "test.pdf")
	rawPDF, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("failed to read test pdf fixture: %v", err)
	}

	expectedTxtPath := filepath.Join("..", "..", "..", "codec", "pdf", "testdata", "test.txt")
	expectedTxt, err := os.ReadFile(expectedTxtPath)
	if err != nil {
		t.Fatalf("failed to read expected txt fixture: %v", err)
	}

	gzPDF := gzipCompress(t, rawPDF)
	if err := rootFS.WriteFile("report.pdf.gz", gzPDF, 0o644); err != nil {
		t.Fatal(err)
	}

	params := &FileReadParams{Filename: "report.pdf.gz"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec(report.pdf.gz) failed: %v", err)
	}

	var content string
	if resp, ok := result.(tools.Response); ok {
		content = resp.Result
	} else {
		content = result.(string)
	}

	expected := strings.ReplaceAll(string(expectedTxt), "\r\n", "\n")
	actual := strings.ReplaceAll(content, "\r\n", "\n")
	if actual != expected {
		t.Errorf("Exec(report.pdf.gz) = %q, want %q", actual, expected)
	}
}

func TestFileReadUncompressedSupported(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	csvContent := "product,price\nApple,1.5\nBanana,0.8\n"
	if err := rootFS.WriteFile("data.csv", []byte(csvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	params := &FileReadParams{Filename: "data.csv"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec(data.csv) failed: %v", err)
	}

	var content string
	if resp, ok := result.(tools.Response); ok {
		content = resp.Result
	} else {
		content = result.(string)
	}

	if !strings.Contains(content, "product") || !strings.Contains(content, "Apple") {
		t.Errorf("Exec(data.csv) unexpected output: %s", content)
	}
}

func TestFileReadNonGzFilenameWithGzipContent(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	csvContent := "city,country\nParis,France\nTokyo,Japan\n"
	gzData := gzipCompress(t, []byte(csvContent))

	// Note: filename is "table.csv" without .gz extension, but content is gzip compressed
	if err := rootFS.WriteFile("table.csv", gzData, 0o644); err != nil {
		t.Fatal(err)
	}

	params := &FileReadParams{Filename: "table.csv"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec(table.csv with gzip data) failed: %v", err)
	}

	var content string
	if resp, ok := result.(tools.Response); ok {
		content = resp.Result
	} else {
		content = result.(string)
	}

	if !strings.Contains(content, "Paris") || !strings.Contains(content, "France") {
		t.Errorf("Exec(table.csv with gzip data) unexpected output: %s", content)
	}
}

func TestFileReadMalformedGzip(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	// Case 1: .gz extension with missing/invalid gzip header (plain text pretending to be .gz)
	if err := rootFS.WriteFile("report.pdf.gz", []byte("plain text with no gzip magic"), 0o644); err != nil {
		t.Fatal(err)
	}
	params := &FileReadParams{Filename: "report.pdf.gz"}
	_, err := g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(report.pdf.gz with plain text) expected failure")
	}

	// Case 2: .gz extension with empty file (0 bytes)
	if err := rootFS.WriteFile("empty.csv.gz", []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	params = &FileReadParams{Filename: "empty.csv.gz"}
	_, err = g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(empty.csv.gz) expected failure")
	}

	// Case 3: gzip magic bytes with invalid header
	if err := rootFS.WriteFile("invalid_header.csv.gz", []byte{0x1f, 0x8b, 0xff, 0xff, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	params = &FileReadParams{Filename: "invalid_header.csv.gz"}
	_, err = g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(invalid_header.csv.gz) expected failure")
	}

	// Case 4: valid header with corrupt body
	validGz := gzipCompress(t, []byte("col1,col2\nval1,val2\n"))
	corruptGz := append(validGz[:10], []byte("not valid compressed body")...)
	if err := rootFS.WriteFile("corrupt_body.csv.gz", corruptGz, 0o644); err != nil {
		t.Fatal(err)
	}
	params = &FileReadParams{Filename: "corrupt_body.csv.gz"}
	_, err = g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(corrupt_body.csv.gz) expected failure")
	}

	// Case 5: plain text file with corrupt gzip
	if err := rootFS.WriteFile("malformed.txt.gz", []byte{0x1f, 0x8b, 0x00, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	params = &FileReadParams{Filename: "malformed.txt.gz"}
	_, err = g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(malformed.txt.gz) expected failure")
	}
}

func TestFileReadPlainTextLimitAndOffset(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	text := "line 1\nline 2\nline 3\nline 4\nline 5"

	// Uncompressed test
	if err := rootFS.WriteFile("lines.txt", []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		limit  *types.FlexibleInt
		offset *types.FlexibleInt
		want   string
	}{
		{name: "all lines", limit: nil, offset: nil, want: "line 1\nline 2\nline 3\nline 4\nline 5"},
		{name: "limit 2 offset 0", limit: (*types.FlexibleInt)(&[]int{2}[0]), offset: nil, want: "line 1\nline 2"},
		{name: "limit 2 offset 2", limit: (*types.FlexibleInt)(&[]int{2}[0]), offset: (*types.FlexibleInt)(&[]int{2}[0]), want: "line 3\nline 4"},
		{name: "limit 0 offset 3", limit: (*types.FlexibleInt)(&[]int{0}[0]), offset: (*types.FlexibleInt)(&[]int{3}[0]), want: "line 4\nline 5"},
		{name: "limit 10 offset 1", limit: (*types.FlexibleInt)(&[]int{10}[0]), offset: (*types.FlexibleInt)(&[]int{1}[0]), want: "line 2\nline 3\nline 4\nline 5"},
	}

	for _, tt := range tests {
		t.Run("uncompressed_"+tt.name, func(t *testing.T) {
			t.Parallel()
			params := &FileReadParams{
				Filename: "lines.txt",
				Limit:    tt.limit,
				Offset:   tt.offset,
			}
			result, err := g.Exec(t.Context(), params)
			if err != nil {
				t.Fatalf("Exec failed: %v", err)
			}

			var got string
			if resp, ok := result.(tools.Response); ok {
				got = resp.Result
			} else {
				got = result.(string)
			}
			if got != tt.want {
				t.Errorf("Exec() = %q, want %q", got, tt.want)
			}
		})
	}

	// Compressed plain-text test
	gzText := gzipCompress(t, []byte(text))
	if err := rootFS.WriteFile("lines.txt.gz", gzText, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run("compressed_"+tt.name, func(t *testing.T) {
			t.Parallel()
			params := &FileReadParams{
				Filename: "lines.txt.gz",
				Limit:    tt.limit,
				Offset:   tt.offset,
			}
			result, err := g.Exec(t.Context(), params)
			if err != nil {
				t.Fatalf("Exec failed: %v", err)
			}

			var got string
			if resp, ok := result.(tools.Response); ok {
				got = resp.Result
			} else {
				got = result.(string)
			}
			if got != tt.want {
				t.Errorf("Exec() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileReadUnrecoverableExtension(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)

	text := "plain text inside abc.gz"
	gzData := gzipCompress(t, []byte(text))

	if err := rootFS.WriteFile("abc.gz", gzData, 0o644); err != nil {
		t.Fatal(err)
	}

	params := &FileReadParams{Filename: "abc.gz"}
	result, err := g.Exec(t.Context(), params)
	if err != nil {
		t.Fatalf("Exec(abc.gz) failed: %v", err)
	}

	var content string
	if resp, ok := result.(tools.Response); ok {
		content = resp.Result
	} else {
		content = result.(string)
	}
	if content != text {
		t.Errorf("Exec(abc.gz) = %q, want %q", content, text)
	}
}

func TestFileReadErrors(t *testing.T) {
	t.Parallel()

	g, rootFS := newTestToolsWithRoot(t)
	tmpDir := rootFS.Name()

	// Non-existent file
	params := &FileReadParams{Filename: "does_not_exist.txt"}
	_, err := g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(does_not_exist.txt) expected failure")
	}

	// Directory
	subDir := filepath.Join(tmpDir, "somedir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	params = &FileReadParams{Filename: "somedir"}
	_, err = g.Exec(t.Context(), params)
	if err == nil {
		t.Errorf("Exec(somedir) expected failure")
	}
}

func newTestToolsWithRoot(t *testing.T) (*Tool, *os.Root) {
	t.Helper()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	rootFS, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	responseStore := response.New(plugin.DefaultResponseStoreSize)
	// Create a valid printer and request factory for tests
	p := printer.GetDefault()
	clientFactory := requester.NewFactory()
	return New(cfg, p, clientFactory, rootFS, &plugin.State{Config: cfg, ReadOnly: false, ResponseStore: responseStore}), rootFS
}
