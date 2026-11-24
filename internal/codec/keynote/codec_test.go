package keynote_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/keynote"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/1.key"

	codec := &keynote.Codec{}

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
