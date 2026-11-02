package docx

import (
	"os"
	"strings"
	"testing"
)

func TestCyrillicText(t *testing.T) {
	codec := &Codec{}
	content := "Таинственный производитель чипов"

	tmpFile, err := os.CreateTemp("", "test-cyrillic-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = codec.WriteFile(tmpFile.Name(), content)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	readData, err := codec.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}

	t.Logf("Original: %s", content)
	t.Logf("Read: %s", readData)

	if !strings.Contains(readData, content) {
		t.Errorf("Expected to find %q in read data, but got %q", content, readData)
	}
}
