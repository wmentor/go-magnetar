package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsCompatibilityMode(t *testing.T) {
	t.Parallel()

	c := &Codec{}
	md := "# Test\n\nCheck [Go](https://go.dev) for details.\n"

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "test.docx")

	if err := c.WriteFile(outFile, md); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip reader error: %v", err)
	}

	for _, f := range reader.File {
		if f.Name == "word/settings.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open settings.xml error: %v", err)
			}
			xmlData, _ := io.ReadAll(rc)
			rc.Close()

			content := string(xmlData)

			// Проверка на наличие compatibilityMode
			if !strings.Contains(content, "compatibilityMode") {
				t.Error("settings.xml missing compatibilityMode (required for Word 2007 compatibility)")
			}

			// Проверка на правильное значение (12 для Word 2007)
			if !strings.Contains(content, `w:val="12"`) {
				t.Error("settings.xml should have compatibilityMode val=12 for Word 2007")
			}

			// Проверка, что characterSpacingControl удалён (может вызывать проблемы в Word 2007)
			if strings.Contains(content, "characterSpacingControl") {
				t.Error("settings.xml should not contain characterSpacingControl (Word 2007 compatibility)")
			}

			t.Logf("settings.xml content:\n%s", content)
		}
	}
}
