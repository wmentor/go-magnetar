package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMarkdownFeatures(t *testing.T) {
	// Read test markdown
	content, err := os.ReadFile("testdata/test_markdown.md")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	codec := &Codec{}
	tmpFile, err := os.CreateTemp("", "test-markdown-*.docx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = codec.WriteFile(tmpFile.Name(), string(content))
	if err != nil {
		t.Fatalf("Failed to write docx: %v", err)
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

	// Check for list items (should have w:numPr)
	if !strings.Contains(docStr, "w:numPr") {
		t.Error("No list items found (w:numPr missing)")
	}

	// Check for bold (should have w:b)
	if !strings.Contains(docStr, "w:b") {
		t.Error("No bold text found (w:b missing)")
	}

	// Check for italic (should have w:i)
	if !strings.Contains(docStr, "w:i") {
		t.Error("No italic text found (w:i missing)")
	}

	// Check for headings
	if !strings.Contains(docStr, "Heading1") {
		t.Error("No Heading1 found")
	}
	if !strings.Contains(docStr, "Heading2") {
		t.Error("No Heading2 found")
	}

	// Check for table
	if !strings.Contains(docStr, "w:tbl") {
		t.Error("No table found (w:tbl missing)")
	}

	// Save for manual inspection
	t.Logf("Generated document.xml:\n%s", docStr)
}
