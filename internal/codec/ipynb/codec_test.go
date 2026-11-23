package ipynb_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/ipynb"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/test1.ipynb"

	codec := &ipynb.Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	expect, err := os.ReadFile("./testdata/test1.md")
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	if string(expect) != data {
		t.Fatalf("invalid result")
	}
}
