package numbers_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/numbers"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/1.numbers"

	codec := &numbers.Codec{}

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
