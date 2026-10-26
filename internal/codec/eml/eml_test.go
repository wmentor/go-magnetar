package eml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodec_ReadFile_SimpleEmail(t *testing.T) {
	t.Parallel()

	filename := "./testdata/simple.eml"

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	// Check that required headers are present
	expectedHeaders := []string{"Subject:", "From:", "To:", "Date:", "Raw Body:"}
	for _, header := range expectedHeaders {
		if !strings.Contains(data, header) {
			t.Errorf("ReadFile(%q) missing header: %s", filename, header)
		}
	}

	// Check that body content is present
	expectedBody := "This is a simple test email body."
	if !strings.Contains(data, expectedBody) {
		t.Errorf("ReadFile(%q) missing expected body content", filename)
	}
}

func TestCodec_ReadFile_HTMLEmail(t *testing.T) {
	t.Parallel()

	filename := "./testdata/html.eml"

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	// Check that headers are present
	expectedHeaders := []string{"Subject:", "From:", "To:", "Date:", "Raw Body:"}
	for _, header := range expectedHeaders {
		if !strings.Contains(data, header) {
			t.Errorf("ReadFile(%q) missing header: %s", filename, header)
		}
	}

	// Check that HTML content is present (as raw body)
	if !strings.Contains(data, "<html>") {
		t.Errorf("ReadFile(%q) missing HTML content", filename)
	}
}

func TestCodec_ReadFile_MultipartEmail(t *testing.T) {
	t.Parallel()

	filename := "./testdata/multipart.eml"

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	// Check that headers are present
	expectedHeaders := []string{"Subject:", "From:", "To:", "Date:", "Raw Body:"}
	for _, header := range expectedHeaders {
		if !strings.Contains(data, header) {
			t.Errorf("ReadFile(%q) missing header: %s", filename, header)
		}
	}
}

func TestCodec_ReadFile_EmptyEmail(t *testing.T) {
	t.Parallel()

	filename := "./testdata/empty.eml"

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	// Check that headers are present (even with empty subject)
	expectedHeaders := []string{"From:", "To:", "Date:", "Raw Body:"}
	for _, header := range expectedHeaders {
		if !strings.Contains(data, header) {
			t.Errorf("ReadFile(%q) missing header: %s", filename, header)
		}
	}

	// Check that body content is present
	if !strings.Contains(data, "Empty email body") {
		t.Errorf("ReadFile(%q) missing body content", filename)
	}
}

func TestCodec_ReadFile_NonExistentFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/nonexistent.eml"

	codec := &Codec{}

	_, err := codec.ReadFile(filename)
	if err == nil {
		t.Errorf("ReadFile(%q) expected error for non-existent file, got nil", filename)
	}
}

func TestCodec_ReadFile_TempFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "temp.eml")

	emlContent := `From: Test User <test@example.com>
To: Recipient <recipient@example.com>
Subject: Temp File Test
Date: Thu, 29 Oct 2026 15:00:00 +0000

This is a temporary file test.
`

	if err := os.WriteFile(filename, []byte(emlContent), 0o644); err != nil {
		t.Fatalf("write temp file error: %v", err)
	}

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read temp file %q error: %v", filename, err)
	}

	// Verify content
	expected := "This is a temporary file test."
	if !strings.Contains(data, expected) {
		t.Errorf("ReadFile(%q) missing expected content", filename)
	}
}

func TestCodec_ReadFile_HeaderValues(t *testing.T) {
	t.Parallel()

	filename := "./testdata/simple.eml"

	codec := &Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	// Check specific header values
	checks := []struct {
		name     string
		expected string
	}{
		{"Subject", "Test Email"},
		{"From", "John Doe <john@example.com>"},
		{"To", "Jane Smith <jane@example.com>"},
		{"Date", "Sun, 25 Oct 2026 12:00:00 +0000"},
	}

	for _, tt := range checks {
		if !strings.Contains(data, tt.expected) {
			t.Errorf("ReadFile(%q) missing expected %s value: %s", filename, tt.name, tt.expected)
		}
	}
}
