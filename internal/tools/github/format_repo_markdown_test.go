package github

import (
	"os"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
)

// Test formatRepoMarkdown with minimal repo data
func TestFormatRepoMarkdown(t *testing.T) {
	t.Parallel()

	// Create a minimal valid config
	cfg := createTestConfig(t)
	ghTools := New(cfg, nil, nil, nil)

	repoData := struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
		License       *struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"license"`
		Language        string   `json:"language"`
		ForksCount      int      `json:"forks_count"`
		Archived        bool     `json:"archived"`
		StargazersCount int      `json:"stargazers_count"`
		OpenIssuesCount int      `json:"open_issues_count"`
		Topics          []string `json:"topics"`
		Description     string   `json:"description"`
		HTMLURL         string   `json:"html_url"`
	}{
		FullName:      "test-owner/test-repo",
		DefaultBranch: "main",
	}

	result := ghTools.formatRepoMarkdown(repoData, "")

	// Verify key elements are present
	if !containsString(result, "# test-owner/test-repo") {
		t.Errorf("result missing repo name: %q", result)
	}
	if !containsString(result, "**Default Branch:** main") {
		t.Errorf("result missing default branch: %q", result)
	}
	if !containsString(result, "**Forks:** 0 | **Stars:** 0 | **Open Issues:** 0") {
		t.Errorf("result missing stats: %q", result)
	}
}

// containsString checks if needle is in haystack
func containsString(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && findSubstring(haystack, needle))
}

// findSubstring finds needle in haystack
func findSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// createTestConfig creates a minimal test config
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	cfg := `version: "1.0"`
	if err := writeStringToFile(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	cfgLoaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	return cfgLoaded
}

// writeStringToFile writes a string to a file
func writeStringToFile(path string, content string) error {
	return writeStringToFileWithMode(path, content, 0o644)
}

// writeStringToFileWithMode writes a string to a file with specific mode
func writeStringToFileWithMode(path string, content string, mode int) error {
	return os.WriteFile(path, []byte(content), os.FileMode(mode))
}
