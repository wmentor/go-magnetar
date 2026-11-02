package docx

import (
	"os"
	"testing"
)

func TestGenerateReadmeDocx(t *testing.T) {
	codec := &Codec{}

	// Read README.md
	readmeContent, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatalf("Error reading README.md: %v", err)
	}

	// Write to docx
	outputPath := "/tmp/readme_final.docx"
	err = codec.WriteFile(outputPath, string(readmeContent))
	if err != nil {
		t.Fatalf("Error writing docx: %v", err)
	}

	// Show file size
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Error getting file info: %v", err)
	}

	t.Logf("✓ README.md converted to %s", outputPath)
	t.Logf("✓ File size: %d bytes", info.Size())
}
