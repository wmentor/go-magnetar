package docx_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/docx"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/1.docx"

	data, err := docx.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	expect, err := os.ReadFile("testdata/1.md")
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	if string(expect) != data {
		t.Fatalf("invalid result")
	}
}
