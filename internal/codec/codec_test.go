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
