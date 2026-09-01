package config

import (
	"os"
	"testing"
)

func TestEnvVarSubstitution(t *testing.T) {
	testConfig := `
profile: default

profiles:
  default:
    llm:
      api_key: $env:TEST_API_KEY
      base_url: https://api.example.com
`

	tmpFile := "/tmp/test-env-config.yaml"
	if err := os.WriteFile(tmpFile, []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	os.Setenv("TEST_API_KEY", "test-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	apiKey := cfg.ProfileParamString("llm.api_key")
	if apiKey != "test-key-123" {
		t.Errorf("expected 'test-key-123', got %q", apiKey)
	}

	baseURL := cfg.ProfileParamString("llm.base_url")
	if baseURL != "https://api.example.com" {
		t.Errorf("expected 'https://api.example.com', got %q", baseURL)
	}
}
