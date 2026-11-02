package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestHyperlinks(t *testing.T) {
	t.Parallel()

	codec := &Codec{}
	content := `# Test Document

Check out [Google](https://google.com) and [GitHub](https://github.com).

Also see [docs/architecture.md](./docs/architecture.md) for more info.
`

	tmpFile, err := os.CreateTemp("", "test-hyperlink-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = codec.WriteFile(tmpFile.Name(), content)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Open and inspect the generated docx
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open docx: %v", err)
	}
	defer zipReader.Close()

	// Find document.xml
	var docXML []byte
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml: %v", err)
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml: %v", err)
			}
			break
		}
	}

	if docXML == nil {
		t.Fatal("document.xml not found in docx")
	}

	docStr := string(docXML)

	// Check for hyperlinks
	if !strings.Contains(docStr, "w:hyperlink") {
		t.Error("No hyperlinks found in document.xml")
	}

	if !strings.Contains(docStr, "Google") {
		t.Error("Link text 'Google' not found")
	}

	if !strings.Contains(docStr, "GitHub") {
		t.Error("Link text 'GitHub' not found")
	}

	// Find word/_rels/document.xml.rels
	var relsXML []byte
	for _, file := range zipReader.File {
		if file.Name == "word/_rels/document.xml.rels" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml.rels: %v", err)
			}
			relsXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml.rels: %v", err)
			}
			break
		}
	}

	if relsXML == nil {
		t.Fatal("document.xml.rels not found in docx")
	}

	relsStr := string(relsXML)

	// Check for hyperlink relationships
	if !strings.Contains(relsStr, "https://google.com") {
		t.Error("Hyperlink relationship for google.com not found")
	}

	if !strings.Contains(relsStr, "https://github.com") {
		t.Error("Hyperlink relationship for github.com not found")
	}

	if !strings.Contains(relsStr, "./docs/architecture.md") {
		t.Error("Hyperlink relationship for docs/architecture.md not found")
	}

	t.Logf("✓ Hyperlinks correctly created")
}
