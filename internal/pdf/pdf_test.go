package pdf_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/pdf"
)

func TestReadPDF1(t *testing.T) {
	t.Parallel()

	data, err := pdf.ReadPDF("testdata/1.pdf")
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 617 {
		t.Fatalf("invalid output size get=%d wait=%d", len(data), 617)
	}
}

func TestReadPDF2(t *testing.T) {
	t.Parallel()

	data, err := pdf.ReadPDF("testdata/2.pdf")
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 5226 {
		t.Fatalf("invalid output size get=%d wait=%d", len(data), 5226)
	}
}
