package csv_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/csv"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/table1.csv"

	codec := &csv.Codec{}

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
