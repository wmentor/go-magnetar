package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestListWithBold(t *testing.T) {
	codec := &Codec{}
	content := `- **Chat** — interactive REPL with multi-turn conversation
- **Index** — adds documents to your RAG knowledge base
- Regular item without bold`

	tmpFile, err := os.CreateTemp("", "test-list-bold-*.docx")
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

	// Check that list items have numPr (are actual list items)
	if !strings.Contains(docStr, "w:numPr") {
		t.Error("No list items found (w:numPr missing)")
	}

	// Check that "Chat" is bold (should have w:b before "Chat")
	if !strings.Contains(docStr, "Chat") {
		t.Error("Text 'Chat' not found in document")
	}

	// Check that bold formatting is applied to "Chat"
	// The structure should be: <w:r><w:rPr><w:b/></w:rPr><w:t>Chat</w:t></w:r>
	if !strings.Contains(docStr, "<w:b>") && !strings.Contains(docStr, "<w:b/>") {
		t.Error("Bold formatting not found in document")
	}

	// Read back and verify content
	readData, err := codec.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read back docx: %v", err)
	}

	t.Logf("Read data:\n%s", readData)

	// Verify that "Chat" appears in the output (not stripped)
	if !strings.Contains(readData, "Chat") {
		t.Error("'Chat' text was stripped from list item")
	}

	// Verify that list markers are present
	if !strings.Contains(readData, "*") && !strings.Contains(readData, "•") {
		t.Error("List markers not found in output")
	}
}
