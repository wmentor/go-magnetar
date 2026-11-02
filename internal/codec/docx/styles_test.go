package docx

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestInjectRequiredStyles(t *testing.T) {
	t.Parallel()

	codec := &Codec{}
	markdownContent := "# Heading 1\n\nSome text\n\n## Heading 2\n\nMore text\n"

	tmpFile, err := os.CreateTemp("", "test-styles-*.docx")
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

	// Open the docx file and check styles.xml
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

	// Find and read styles.xml
	foundStyles := false
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/styles.xml" {
			foundStyles = true
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
					t.Errorf("missing style: %s", style)
				}
			}
			break
		}
	}

	if !foundStyles {
		t.Error("styles.xml not found in docx")
	}
}
