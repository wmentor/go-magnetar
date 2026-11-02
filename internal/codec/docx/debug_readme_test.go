package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDebugReadmeLists(t *testing.T) {
	codec := &Codec{}

	// Read README.md
	readmeContent, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	// Create temp docx file
	tmpFile, err := os.CreateTemp("", "readme-debug-*.docx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Convert README to docx
	err = codec.WriteFile(tmpFile.Name(), string(readmeContent))
	if err != nil {
		t.Fatalf("Failed to write docx: %v", err)
	}

	// Open and inspect the generated docx
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open docx: %v", err)
	}
	defer zipReader.Close()

	// Check document.xml for list items
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml: %v", err)
			}
			docXML, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml: %v", err)
			}

			docStr := string(docXML)

			// Find first few list items
			lines := strings.Split(docStr, "\n")
			listItemCount := 0
			for i, line := range lines {
				if strings.Contains(line, "w:numPr") {
					listItemCount++
					if listItemCount <= 5 {
						// Print context around list item
						start := max(0, i-5)
						end := min(len(lines), i+10)
						t.Logf("List item #%d found at line %d:", listItemCount, i)
						for j := start; j < end; j++ {
							t.Logf("  %s", lines[j])
						}
						t.Log("---")
					}
				}
			}
			t.Logf("Total list items found: %d", listItemCount)
			break
		}
	}

	// Check numbering.xml
	for _, file := range zipReader.File {
		if file.Name == "word/numbering.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open numbering.xml: %v", err)
			}
			numXML, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read numbering.xml: %v", err)
			}

			t.Logf("numbering.xml content:\n%s", string(numXML))
			break
		}
	}
}
