package template

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var templatesFS embed.FS

// EnsureConfigDir checks if ~/.go-magnetar/ exists and creates it with templates if needed.
func EnsureConfigDir(homeDir string) error {
	configDir := filepath.Join(homeDir, ".go-magnetar")

	// Check if config directory exists
	if _, err := os.Stat(configDir); err == nil {
		// Directory exists, sync templates
		return SyncTemplates(configDir)
	}

	// Directory doesn't exist, create it
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Copy all templates (excluding hidden files/directories)
	return copyDir(configDir, "templates")
}

// SyncTemplates copies new template files from embed.FS to config directory.
func SyncTemplates(configDir string) error {
	// Copy all templates (excluding hidden files/directories)
	return copyDir(configDir, "templates")
}

// copyDir recursively copies files from embed.FS to local filesystem.
func copyDir(baseDir string, embedPrefix string) error {
	return fs.WalkDir(templatesFS, embedPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories (starting with .)
		if isHidden(path) {
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Calculate relative path from embedPrefix
		relativePath := strings.TrimPrefix(path, embedPrefix+"/")
		localPath := filepath.Join(baseDir, relativePath)

		// Check if file already exists
		if _, err := os.Stat(localPath); err == nil {
			return nil
		}

		// Copy file
		return extractFile(baseDir, path, relativePath)
	})
}

// isHidden checks if a path contains hidden components (starting with .)
func isHidden(path string) bool {
	// Skip if path contains /.
	if strings.Contains(path, "/.") {
		return true
	}
	// Skip if path starts with .
	if strings.HasPrefix(path, ".") {
		return true
	}
	return false
}

// extractFile extracts a single file from embed.FS to the local filesystem.
func extractFile(baseDir string, embedPath string, localName string) error {
	data, err := templatesFS.ReadFile(embedPath)
	if err != nil {
		return err
	}

	localPath := filepath.Join(baseDir, localName)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(localPath, data, 0644)
}
