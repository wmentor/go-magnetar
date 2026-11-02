package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFullDocumentIntegration(t *testing.T) {
	t.Parallel()

	codec := &Codec{}

	markdownContent := `# Main Heading

Some introductory text.

## Section 1

First list:
* Item 1
* Item 2
  * Nested item

Second list (should restart numbering):
* First item
* Second item

### Subsection

Numbered list:
1. First
2. Second
3. Third

Another numbered list (should restart):
1. One
2. Two

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

Final paragraph.
`

	tmpFile, err := os.CreateTemp("", "test-full-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	tmpFilename := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilename)

	err = codec.WriteFile(tmpFilename, markdownContent)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Verify file structure
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

	// Check that all required files exist
	requiredFiles := map[string]bool{
		"[Content_Types].xml":          false,
		"_rels/.rels":                  false,
		"word/document.xml":            false,
		"word/styles.xml":              false,
		"word/fontTable.xml":           false,
		"word/theme/theme1.xml":        false,
		"word/numbering.xml":           false,
		"word/settings.xml":            false,
		"word/_rels/document.xml.rels": false,
		"docProps/app.xml":             false,
		"docProps/core.xml":            false,
	}

	for _, zipFile := range zipReader.File {
		if _, exists := requiredFiles[zipFile.Name]; exists {
			requiredFiles[zipFile.Name] = true
		}
	}

	for file, found := range requiredFiles {
		if !found {
			t.Errorf("required file missing: %s", file)
		}
	}

	// Verify styles.xml contains heading and table styles
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/styles.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				t.Fatalf("open styles.xml error: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read styles.xml error: %v", err)
			}

			// Check for heading styles
			headingStyles := []string{"Heading1", "Heading2", "Heading3", "Heading4", "Heading5", "Heading6"}
			for _, style := range headingStyles {
				if !strings.Contains(string(content), style) {
					t.Errorf("heading style missing in styles.xml: %s", style)
				}
			}

			// Check for table styles
			tableStyles := []string{"TableGrid", "TableNormal"}
			for _, style := range tableStyles {
				if !strings.Contains(string(content), style) {
					t.Errorf("table style missing in styles.xml: %s", style)
				}
			}
			break
		}
	}

	// Verify numbering.xml exists and contains multiple abstractNum definitions
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/numbering.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				t.Fatalf("open numbering.xml error: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read numbering.xml error: %v", err)
			}

			// Should have two abstractNum definitions (0 for bullet, 1 for numbered)
			if !strings.Contains(string(content), `abstractNumId="0"`) {
				t.Error("numbering.xml missing abstractNumId 0 (bullet)")
			}
			if !strings.Contains(string(content), `abstractNumId="1"`) {
				t.Error("numbering.xml missing abstractNumId 1 (numbered)")
			}
			break
		}
	}

	// Verify relationships include numbering and settings
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/_rels/document.xml.rels" {
			rc, err := zipFile.Open()
			if err != nil {
				t.Fatalf("open document.xml.rels error: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read document.xml.rels error: %v", err)
			}

			if !strings.Contains(string(content), "numbering.xml") {
				t.Error("document.xml.rels missing numbering relationship")
			}
			if !strings.Contains(string(content), "settings.xml") {
				t.Error("document.xml.rels missing settings relationship")
			}
			break
		}
	}
}
