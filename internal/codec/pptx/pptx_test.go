package pptx_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/pptx"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	codec := &pptx.Codec{}

	filename := "./testdata/1.pptx"

	data, err := codec.ReadFile(filename)
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
