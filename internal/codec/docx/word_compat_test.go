package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestWordCompatibility(t *testing.T) {
	t.Parallel()

	// Create a test markdown content with various elements
	markdownContent := `# Heading 1

This is a paragraph under heading 1.

## Heading 2

This is a paragraph under heading 2.

### Heading 3

This is a paragraph under heading 3.

* List item 1
* List item 2
  * Nested item

| Column 1 | Column 2 |
|----------|----------|
| Cell 1   | Cell 2   |

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

Some text with ` + "`inline code`" + ` in it.
`

	codec := &Codec{}
	tmpFile, err := os.CreateTemp("", "test-word-compat-*.docx")
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

	// Open the docx file and verify structure
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
	requiredFiles := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/document.xml",
		"word/styles.xml",
		"word/fontTable.xml",
		"word/theme/theme1.xml",
		"docProps/app.xml",
		"docProps/core.xml",
	}

	foundFiles := make(map[string]bool)
	for _, zipFile := range zipReader.File {
		foundFiles[zipFile.Name] = true
	}

	for _, required := range requiredFiles {
		if !foundFiles[required] {
			t.Errorf("required file missing: %s", required)
		}
	}

	// Find and verify styles.xml contains heading styles
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
			styles := []string{"Heading1", "Heading2", "Heading3", "Heading4", "Heading5", "Heading6"}
			for _, style := range styles {
				if !strings.Contains(string(content), style) {
					t.Errorf("heading style missing in styles.xml: %s", style)
				}
			}
			break
		}
	}
}
