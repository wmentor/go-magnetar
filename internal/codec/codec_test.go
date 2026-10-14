package codec_test

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/codec"
)

func TestGetCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ext     string
		wantErr bool
	}{
		{name: "csv extension with dot", ext: ".csv", wantErr: false},
		{name: "csv extension without dot", ext: "csv", wantErr: false},
		{name: "docx extension with dot", ext: ".docx", wantErr: false},
		{name: "docx extension without dot", ext: "docx", wantErr: false},
		{name: "pdf extension with dot", ext: ".pdf", wantErr: false},
		{name: "pdf extension without dot", ext: "pdf", wantErr: false},
		{name: "odt extension with dot", ext: ".odt", wantErr: false},
		{name: "odt extension without dot", ext: "odt", wantErr: false},
		{name: "pptx extension with dot", ext: ".pptx", wantErr: false},
		{name: "pptx extension without dot", ext: "pptx", wantErr: false},
		{name: "xlsx extension with dot", ext: ".xlsx", wantErr: false},
		{name: "xlsx extension without dot", ext: "xlsx", wantErr: false},
		{name: "unknown extension", ext: ".unknown", wantErr: true},
		{name: "empty extension", ext: "", wantErr: true},
		{name: "uppercase CSV", ext: ".CSV", wantErr: false},
		{name: "uppercase docx", ext: ".DOCX", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.GetCodec(tt.ext)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetCodec(%q) expected error, got nil", tt.ext)
				}
				if !errors.Is(err, codec.ErrCodecNotFound) {
					t.Errorf("GetCodec(%q) expected ErrCodecNotFound, got %v", tt.ext, err)
				}
			} else {
				if err != nil {
					t.Errorf("GetCodec(%q) unexpected error: %v", tt.ext, err)
				}
			}
		})
	}
}

func TestGetCodecReturnsCodec(t *testing.T) {
	t.Parallel()

	codec, err := codec.GetCodec(".csv")
	if err != nil {
		t.Fatalf("GetCodec(\".csv\") unexpected error: %v", err)
	}

	if codec == nil {
		t.Fatal("GetCodec(\".csv\") returned nil codec")
	}
}

func TestGetCodecByMIME(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mime    string
		wantErr bool
	}{
		{name: "csv MIME", mime: "text/csv", wantErr: false},
		{name: "tsv MIME", mime: "text/tab-separated-values", wantErr: false},
		{name: "pdf MIME", mime: "application/pdf", wantErr: false},
		{name: "docx MIME", mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", wantErr: false},
		{name: "xlsx MIME", mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", wantErr: false},
		{name: "odt MIME", mime: "application/vnd.oasis.opendocument.text", wantErr: false},
		{name: "pptx MIME", mime: "application/vnd.openxmlformats-officedocument.presentationml.presentation", wantErr: false},
		{name: "unknown MIME", mime: "application/unknown", wantErr: true},
		{name: "empty MIME", mime: "", wantErr: true},
		{name: "uppercase CSV MIME", mime: "TEXT/CSV", wantErr: false},
		{name: "uppercase PDF MIME", mime: "APPLICATION/PDF", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.GetCodecByMIME(tt.mime)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetCodecByMIME(%q) expected error, got nil", tt.mime)
				}
				if !errors.Is(err, codec.ErrCodecNotFound) {
					t.Errorf("GetCodecByMIME(%q) expected ErrCodecNotFound, got %v", tt.mime, err)
				}
			} else {
				if err != nil {
					t.Errorf("GetCodecByMIME(%q) unexpected error: %v", tt.mime, err)
				}
			}
		})
	}
}

func TestGetCodecByMIMEReturnsCodec(t *testing.T) {
	t.Parallel()

	codec, err := codec.GetCodecByMIME("text/csv")
	if err != nil {
		t.Fatalf("GetCodecByMIME(\"text/csv\") unexpected error: %v", err)
	}

	if codec == nil {
		t.Fatal("GetCodecByMIME(\"text/csv\") returned nil codec")
	}
}

func TestIsBinaryFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mime string
		want bool
	}{
		{name: "PDF", mime: "application/pdf", want: true},
		{name: "DOCX", mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", want: true},
		{name: "XLSX", mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", want: true},
		{name: "ODT", mime: "application/vnd.oasis.opendocument.text", want: true},
		{name: "PPTX", mime: "application/vnd.openxmlformats-officedocument.presentationml.presentation", want: true},
		{name: "CSV text", mime: "text/csv", want: false},
		{name: "TSV text", mime: "text/tab-separated-values", want: false},
		{name: "HTML", mime: "text/html", want: false},
		{name: "unknown binary", mime: "application/unknown", want: false},
		{name: "empty string", mime: "", want: false},
		{name: "uppercase PDF", mime: "APPLICATION/PDF", want: false},
		{name: "mixed case PDF", mime: "Application/Pdf", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := codec.IsBinaryFormat(tt.mime)
			if got != tt.want {
				t.Errorf("IsBinaryFormat(%q) = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}
