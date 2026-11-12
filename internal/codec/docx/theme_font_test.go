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

func TestThemeUsesTimesNewRoman(t *testing.T) {
	t.Parallel()

	c := &Codec{}
	md := "# Test Heading\n\nSome text.\n"

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
		if f.Name == "word/theme/theme1.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open theme1.xml error: %v", err)
			}
			xmlData, _ := io.ReadAll(rc)
			rc.Close()

			content := string(xmlData)

			// Проверка, что используется Times New Roman
			if !strings.Contains(content, `typeface="Times New Roman"`) {
				t.Error("theme1.xml should use Times New Roman font")
			}

			// Проверка, что НЕ используются китайские шрифты
			if strings.Contains(content, "等线") {
				t.Error("theme1.xml should not contain Chinese fonts (等线)")
			}

			t.Logf("theme1.xml uses Times New Roman correctly")
		}
	}
}
