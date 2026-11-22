package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestDotenvLoading(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")
	content := "MAGNETAR_TEST_VAR=autoloaded_value\n"

	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test .env file: %v", err)
	}

	envMap, err := godotenv.Read(envFile)
	if err != nil {
		t.Fatalf("failed to read .env file via godotenv: %v", err)
	}

	if envMap["MAGNETAR_TEST_VAR"] != "autoloaded_value" {
		t.Errorf("expected 'autoloaded_value', got %q", envMap["MAGNETAR_TEST_VAR"])
	}
}
