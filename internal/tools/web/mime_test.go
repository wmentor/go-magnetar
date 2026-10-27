package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/pdf"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "HTML content type",
			input:   "text/html; charset=utf-8",
			want:    "text/html",
			wantErr: false,
		},
		{
			name:    "JSON content type",
			input:   "application/json; charset=utf-8",
			want:    "application/json",
			wantErr: false,
		},
		{
			name:    "PDF content type",
			input:   "application/pdf",
			want:    "application/pdf",
			wantErr: false,
		},
		{
			name:    "DOCX content type",
			input:   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			want:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			wantErr: false,
		},
		{
			name:    "XLSX content type",
			input:   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			want:    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			wantErr: false,
		},
		{
			name:    "ODT content type",
			input:   "application/vnd.oasis.opendocument.text",
			want:    "application/vnd.oasis.opendocument.text",
			wantErr: false,
		},
		{
			name:    "PPTX content type",
			input:   "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			want:    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			wantErr: false,
		},
		{
			name:    "CSV content type",
			input:   "text/csv",
			want:    "text/csv",
			wantErr: false,
		},
		{
			name:    "TSV content type",
			input:   "text/tab-separated-values",
			want:    "text/tab-separated-values",
			wantErr: false,
		},
		{
			name:    "UTF-8 HTML content type",
			input:   "text/html; charset=UTF-8",
			want:    "text/html",
			wantErr: false,
		},
		{
			name:    "uppercase content type",
			input:   "TEXT/HTML; CHARSET=UTF-8",
			want:    "text/html",
			wantErr: false,
		},
		{
			name:    "empty content type",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "invalid content type",
			input:   "invalid/; charset=utf-8",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMimeType(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetMimeType() expected error, got nil")
				}
				if err != nil && !strings.Contains(err.Error(), tt.input) {
					// Check if error is related to our test case
					t.Logf("GetMimeType() error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("GetMimeType() unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("GetMimeType() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMimeCodecWork(t *testing.T) {
	t.Run("PDF file decoding", func(t *testing.T) {
		// Read the test PDF file
		pdfPath := "testdata/test_mime_decode.pdf"
		pdfContent, err := os.ReadFile(pdfPath)
		if err != nil {
			t.Fatalf("failed to read test PDF file %s: %v", pdfPath, err)
		}

		// Expected result
		expectedPath := "testdata/test_mime_decode_result.txt"
		expectedContent, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("failed to read expected result file %s: %v", expectedPath, err)
		}
		expected := string(expectedContent)

		// Test mimeCodecWork with PDF codec
		result, err := mimeCodecWork(&pdf.Codec{}, pdfContent)
		if err != nil {
			t.Fatalf("mimeCodecWork() returned error: %v", err)
		}

		if result != expected {
			t.Errorf("mimeCodecWork() = %q, want %q", result, expected)
			t.Logf("Result length: %d, Expected length: %d", len(result), len(expected))
		}

		// Verify the result contains expected text
		if !strings.Contains(result, "Code Review") {
			t.Errorf("Result does not contain expected text 'Code Review'")
			t.Logf("Result: %q", result)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		// Test with empty content
		_, err := mimeCodecWork(&pdf.Codec{}, []byte{})
		if err == nil {
			t.Errorf("mimeCodecWork() expected error for empty content, got nil")
		}
		t.Logf("Expected error for empty PDF content, got: %v", err)
	})

	t.Run("unknown codec", func(t *testing.T) {
		// Test with nil codec
		_, err := mimeCodecWork(nil, []byte("test"))
		if err == nil {
			t.Errorf("mimeCodecWork() expected error for nil codec, got nil")
		}
		if !strings.Contains(err.Error(), "codec is nil") {
			t.Errorf("Expected error about nil codec, got: %v", err)
		}
		t.Logf("Expected error for nil codec, got: %v", err)
	})
}

func TestWebFetch_MimeCodecError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("not a valid pdf"))
	}))
	defer srv.Close()

	cfgPath := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(cfgPath, []byte("version: \"1.0\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error: %v", err)
	}
	wt, err := New(cfg, nil, &plugin.State{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	got, err := wt.webFetch(srv.URL, false)
	if err == nil {
		t.Fatalf("expected decode data error, got nil (result=%q)", got)
	}
	if !strings.Contains(err.Error(), "decode data error:") {
		t.Fatalf("expected decode data error, got: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty result on error, got %q", got)
	}
}
