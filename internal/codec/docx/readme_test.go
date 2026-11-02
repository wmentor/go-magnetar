package docx

import (
	"os"
	"strings"
	"testing"
)

func TestReadmeConversion(t *testing.T) {
	codec := &Codec{}

	// Read README.md
	readmeContent, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	// Create temp docx file
	tmpFile, err := os.CreateTemp("", "readme-test-*.docx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Convert README to docx
	err = codec.WriteFile(tmpFile.Name(), string(readmeContent))
	if err != nil {
		t.Fatalf("Failed to write docx: %v", err)
	}

	// Read back
	readData, err := codec.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read back docx: %v", err)
	}

	// Check that key content is preserved
	checks := []string{
		"go-magnetar",
		"Chat",
		"Index",
		"Fetch",
		"Skills",
		"interactive REPL",
		"multi-turn conversation",
	}

	for _, check := range checks {
		if !strings.Contains(readData, check) {
			t.Errorf("Expected text %q not found in output", check)
		}
	}

	// Check that list items are present
	if !strings.Contains(readData, "* Chat") && !strings.Contains(readData, "• Chat") {
		t.Error("List item 'Chat' not found with proper marker")
	}

	t.Logf("README.md successfully converted to docx")
	t.Logf("First 1000 chars of output:\n%s", readData[:min(1000, len(readData))])
}
