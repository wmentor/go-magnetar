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

func TestHyperlinksNotInDocumentXML(t *testing.T) {
	t.Parallel()

	c := &Codec{}
	md := "# Test\n\nCheck [Go](https://go.dev) and [GitHub](https://github.com).\n"

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
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open document.xml error: %v", err)
			}
			xmlData, _ := io.ReadAll(rc)
			rc.Close()

			content := string(xmlData)
			if strings.Contains(content, "<Hyperlinks>") {
				t.Error("document.xml contains invalid <Hyperlinks> elements that break Word compatibility")
			}
			if !strings.Contains(content, "w:hyperlink") {
				t.Error("document.xml missing w:hyperlink elements")
			}
			t.Logf("document.xml length: %d bytes", len(xmlData))
		}
	}
}
