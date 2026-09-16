package tsv_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/tsv"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/table1.tsv"

	codec := &tsv.Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	expect, err := os.ReadFile("./testdata/table1.md")
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	if string(expect) != data {
		t.Fatalf("invalid result")
	}
}
