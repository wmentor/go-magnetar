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

func TestFontTableNoChineseFonts(t *testing.T) {
	t.Parallel()

	c := &Codec{}
	md := "# Test\n\nSome text with code:\n\n```go\nfmt.Println(\"hello\")\n```\n"

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
		if f.Name == "word/fontTable.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open fontTable.xml error: %v", err)
			}
			xmlData, _ := io.ReadAll(rc)
			rc.Close()

			content := string(xmlData)

			// Проверка, что используются стандартные шрифты
			expectedFonts := []string{"Times New Roman", "Courier New", "Arial", "Cambria"}
			for _, font := range expectedFonts {
				if !strings.Contains(content, font) {
					t.Errorf("fontTable.xml should contain %s", font)
				}
			}

			// Проверка, что НЕ используются китайские шрифты
			chineseFonts := []string{"宋体", "等线", "新細明體", "游ゴシック"}
			for _, font := range chineseFonts {
				if strings.Contains(content, font) {
					t.Errorf("fontTable.xml should not contain Chinese font: %s", font)
				}
			}

			t.Logf("fontTable.xml contains only standard fonts")
		}
	}
}
