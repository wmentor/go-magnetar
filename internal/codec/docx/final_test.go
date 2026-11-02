package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFinalReadmeConversion(t *testing.T) {
	codec := &Codec{}

	// Read README.md
	readmeContent, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	// Create temp docx file
	tmpFile, err := os.CreateTemp("", "readme-final-*.docx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Convert README to docx
	err = codec.WriteFile(tmpFile.Name(), string(readmeContent))
	if err != nil {
		t.Fatalf("Failed to write docx: %v", err)
	}

	// Open and verify the generated docx
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open docx: %v", err)
	}
	defer zipReader.Close()

	// Check document.xml
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
		t.Fatal("document.xml not found")
	}

	docStr := string(docXML)

	// Check for list items with proper numPr
	if !strings.Contains(docStr, "w:numPr") {
		t.Error("No list items found (w:numPr missing)")
	}

	// Check for bold text
	if !strings.Contains(docStr, "<w:b>") {
		t.Error("No bold text found")
	}

	// Check for Cyrillic text
	if !strings.Contains(docStr, "go-magnetar") {
		t.Error("Main title not found")
	}

	// Check numbering.xml
	var numXML []byte
	for _, file := range zipReader.File {
		if file.Name == "word/numbering.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open numbering.xml: %v", err)
			}
			numXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read numbering.xml: %v", err)
			}
			break
		}
	}

	if numXML == nil {
		t.Fatal("numbering.xml not found")
	}

	numStr := string(numXML)

	// Check for widowControl
	if !strings.Contains(numStr, "widowControl") {
		t.Error("widowControl not found in numbering.xml")
	}

	// Check for proper bullet format
	if !strings.Contains(numStr, `<w:lvlText w:val="&#x2022;"/>`) {
		t.Error("Proper bullet character not found in numbering.xml")
	}

	t.Logf("✓ Document generated successfully with proper list formatting")
}
