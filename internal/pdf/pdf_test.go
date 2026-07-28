package pdf_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/pdf"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/test.pdf"

	data, err := pdf.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	expect, err := os.ReadFile("testdata/test.txt")
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	if string(expect) != data {
		t.Fatalf("invalid result")
	}
}
