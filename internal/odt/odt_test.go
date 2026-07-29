package odt_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/odt"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/test.odt"

	data, err := odt.ReadFile(filename)
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
