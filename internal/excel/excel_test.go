package excel_test

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/excel"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/table1.xlsx"

	data, err := excel.ReadFile(filename)
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
