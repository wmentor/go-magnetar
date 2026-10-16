package web

import (
	"fmt"
	"net/url"
	"strings"
)

// extractGitLabRepoURL extracts the project path from a GitLab repository URL.
func extractGitLabRepoURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Get path from URL
	path := u.Path

	// Trim leading/trailing slashes
	path = strings.Trim(path, "/")

	// Check if we have a valid project path (contains at least one slash for namespace/project)
	if path == "" || !strings.Contains(path, "/") {
		return "", fmt.Errorf("invalid GitLab repository URL format")
	}

	return path, nil
}

// extractGitLabFileURL extracts project path, branch, and file from a GitLab file URL.
func extractGitLabFileURL(rawURL string) (string, string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check for /-/blob/ pattern in path
	if !strings.Contains(u.Path, "/-/blob/") {
		return "", "", "", fmt.Errorf("not a GitLab file URL")
	}

	// Split path at /-/blob/
	parts := strings.SplitN(u.Path, "/-/blob/", 2)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid GitLab file URL format")
	}

	// Extract project path from before /-/blob/
	projectPath := strings.TrimPrefix(parts[0], "/")

	// Extract branch and file from after /-/blob/
	pathAfterBlob := strings.TrimPrefix(parts[1], "/")
	pathParts := strings.SplitN(pathAfterBlob, "/", 2)
	if len(pathParts) < 2 {
		return "", "", "", fmt.Errorf("invalid GitLab file URL format")
	}

	branch := pathParts[0]
	file := pathParts[1]

	// File path must not be empty
	if file == "" {
		return "", "", "", fmt.Errorf("file path is empty")
	}

	// Clean up file path (remove query params and fragment)
	if idx := strings.Index(file, "?"); idx != -1 {
		file = file[:idx]
	}
	if idx := strings.Index(file, "#"); idx != -1 {
		file = file[:idx]
	}

	return projectPath, branch, file, nil
}

// extractGitLabTreeURL extracts project path, branch, and path from a GitLab tree URL.
func extractGitLabTreeURL(rawURL string) (string, string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check for /-/tree/ or /-/commits/ pattern in path
	marker := ""
	if strings.Contains(u.Path, "/-/tree/") {
		marker = "/-/tree/"
	} else if strings.Contains(u.Path, "/-/commits/") {
		marker = "/-/commits/"
	} else {
		return "", "", "", fmt.Errorf("not a GitLab tree URL")
	}

	// Split path at marker
	parts := strings.SplitN(u.Path, marker, 2)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid GitLab tree URL format")
	}

	// Extract project path from before marker
	projectPath := strings.TrimPrefix(parts[0], "/")

	// Extract branch and path from after marker
	pathAfterMarker := strings.TrimPrefix(parts[1], "/")
	pathParts := strings.SplitN(pathAfterMarker, "/", 2)
	if len(pathParts) < 1 {
		return "", "", "", fmt.Errorf("invalid GitLab tree URL format")
	}

	branch := pathParts[0]
	treePath := ""
	if len(pathParts) > 1 {
		treePath = pathParts[1]
	}

	// Clean up branch (remove query params and fragment)
	if idx := strings.Index(branch, "?"); idx != -1 {
		branch = branch[:idx]
	}
	if idx := strings.Index(treePath, "?"); idx != -1 {
		treePath = treePath[:idx]
	}

	return projectPath, branch, treePath, nil
}
