package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestListsAndCodeBlocks(t *testing.T) {
	t.Parallel()

	// Read the test markdown file
	content, err := os.ReadFile("testdata/test_lists_and_code.md")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	codec := &Codec{}
	tmpFile, err := os.CreateTemp("", "test-lists-code-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	tmpFilename := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilename)

	err = codec.WriteFile(tmpFilename, string(content))
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Open and inspect the generated docx
	readFile, err := os.Open(tmpFilename)
	if err != nil {
		t.Fatalf("open file error: %v", err)
	}
	defer readFile.Close()

	fileInfo, err := readFile.Stat()
	if err != nil {
		t.Fatalf("stat file error: %v", err)
	}

	zipReader, err := zip.NewReader(readFile, fileInfo.Size())
	if err != nil {
		t.Fatalf("create zip reader error: %v", err)
	}

	// Find and check document.xml
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				t.Fatalf("open document.xml error: %v", err)
			}
			defer rc.Close()

			docContent, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read document.xml error: %v", err)
			}

			docStr := string(docContent)

			// Check that code blocks don't have ``` markers
			if strings.Contains(docStr, "```") {
				t.Error("document.xml should not contain ``` markers")
			}

			// Check that we have list items with numPr
			if !strings.Contains(docStr, "<w:numPr") {
				t.Error("document.xml should contain list items with numPr")
			}

			// Check that we have different numId values for different lists
			if !strings.Contains(docStr, `<w:numId w:val="1">`) {
				t.Error("document.xml should contain numId=1 for first bullet list")
			}
			if !strings.Contains(docStr, `<w:numId w:val="3">`) {
				t.Error("document.xml should contain numId=3 for second bullet list")
			}

			t.Logf("Document XML snippet (first 2000 chars):\n%s", docStr[:min(2000, len(docStr))])
			break
		}
	}

	// Find and check numbering.xml
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/numbering.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				t.Fatalf("open numbering.xml error: %v", err)
			}
			defer rc.Close()

			numContent, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read numbering.xml error: %v", err)
			}

			numStr := string(numContent)

			// Check that we have abstractNum definitions
			if !strings.Contains(numStr, `abstractNumId="0"`) {
				t.Error("numbering.xml should contain abstractNumId=0 for bullet lists")
			}
			if !strings.Contains(numStr, `abstractNumId="1"`) {
				t.Error("numbering.xml should contain abstractNumId=1 for numbered lists")
			}

			// Check that we have num references
			if !strings.Contains(numStr, `w:numId="1"`) {
				t.Error("numbering.xml should contain numId=1")
			}
			if !strings.Contains(numStr, `w:numId="2"`) {
				t.Error("numbering.xml should contain numId=2")
			}

			t.Logf("Numbering XML:\n%s", numStr)
			break
		}
	}
}
