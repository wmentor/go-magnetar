package web

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	errNotGitHubURL          = errors.New("not a GitHub URL")
	errNotGitHubTreeURL      = errors.New("not a GitHub tree/commits URL")
	errNotGitHubBlobURL      = errors.New("not a GitHub blob URL")
	errNotGitHubIssueURL     = errors.New("not a GitHub issue URL")
	errNotGitHubMilestoneURL = errors.New("not a GitHub milestone URL")
	errNotGitHubPRURL        = errors.New("not a GitHub pull request URL")
	errNotGitHubReleaseURL   = errors.New("not a GitHub release URL")
)

// Patterns for the path of a github.com URL (see githubURLPath).
var (
	// Matches /{owner}/{repo}/(tree|commits)/{branch}[/path].
	githubTreePathRe = regexp.MustCompile(`^/([^/]+)/([^/]+)/(tree|commits)/([^/]+)(?:/(.+))?$`)
	// Matches /{owner}/{repo}/blob/{branch}/{file}.
	githubBlobPathRe = regexp.MustCompile(`^/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
	// Matches /{owner}/{repo}/issues/{number}.
	githubIssuePathRe = regexp.MustCompile(`^/([^/]+)/([^/]+)/issues/(\d+)$`)
	// Matches /{owner}/{repo}/milestone/{number}.
	githubMilestonePathRe = regexp.MustCompile(`^/([^/]+)/([^/]+)/milestone/(\d+)$`)
	// Matches /{owner}/{repo}/pull/{number} optionally followed by /files or /commits.
	// This ensures it's a PR URL with explicit number, not just /pull/ as part of repo name.
	githubPRPathRe = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/(\d+)(?:/(files|commits))?$`)
)

// isGitHubPullRequestURL checks if the URL is likely a GitHub PR URL.
// Returns true only if the URL matches the expected PR pattern.
func isGitHubPullRequestURL(rawURL string) bool {
	if !strings.Contains(rawURL, "github.com/") {
		return false
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	path := strings.TrimPrefix(parsed.Path, "/")
	parts := strings.Split(path, "/")

	// Expected structure: [owner, repo, pull, number] or [owner, repo, pull, number, files/commits]
	// So parts[2] must be "pull" and parts[3] must be a number
	if len(parts) < 4 {
		return false
	}

	// Check that parts[2] is exactly "pull" (not just a word containing pull)
	if parts[2] != "pull" {
		return false
	}

	// Check that parts[3] is a number (PR number)
	if len(parts[3]) == 0 {
		return false
	}
	for i := 0; i < len(parts[3]); i++ {
		if parts[3][i] < '0' || parts[3][i] > '9' {
			return false
		}
	}

	// If there are more parts, they must be "files" or "commits"
	if len(parts) > 4 {
		if parts[4] != "files" && parts[4] != "commits" {
			return false
		}
	}

	// Ensure there's no second "pull" in the path
	pullCount := 0
	for _, part := range parts {
		if part == "pull" {
			pullCount++
		}
	}
	// Must have exactly one "pull" in the path
	if pullCount != 1 {
		return false
	}

	return true
}

// githubURLPath returns the escaped path of a github.com URL without the query string,
// fragment and trailing slash. It returns an error for any other host.
func githubURLPath(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com":
	default:
		return "", errNotGitHubURL
	}

	return strings.TrimRight(u.EscapedPath(), "/"), nil
}

// extractGitHubTreeURL extracts owner, repo, branch, and path from GitHub tree URL.
// Pattern: https://github.com/owner/repo/tree/branch/path or https://github.com/owner/repo/commits/branch
func extractGitHubTreeURL(rawURL string) (string, string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", "", err
	}

	matches := githubTreePathRe.FindStringSubmatch(path)
	if matches != nil {
		// matches[5] (the path) is empty when the URL has no path after the branch.
		return matches[1], matches[2], matches[4], matches[5], nil
	}

	return "", "", "", "", errNotGitHubTreeURL
}

// extractGitHubFileURL extracts owner, repo, branch, and file path from GitHub blob URL.
// Pattern: https://github.com/owner/repo/blob/branch/path/to/file
func extractGitHubFileURL(rawURL string) (string, string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", "", err
	}

	matches := githubBlobPathRe.FindStringSubmatch(path)
	if matches != nil {
		owner := matches[1]
		repo := matches[2]
		branch := matches[3]
		file := matches[4]
		return owner, repo, branch, file, nil
	}

	return "", "", "", "", errNotGitHubBlobURL
}

func extractGitHubIssueURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	matches := githubIssuePathRe.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", errNotGitHubIssueURL
}

func extractGitHubMilestoneURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	matches := githubMilestonePathRe.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", errNotGitHubMilestoneURL
}

func extractGitHubPullRequestURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	matches := githubPRPathRe.FindStringSubmatch(path)
	if matches != nil {
		// matches[4] can be "files" or "commits" or empty
		return matches[1], matches[2], matches[3], nil
	}
	return "", "", "", errNotGitHubPRURL
}

func extractGitHubAdvisoryURL(url string) (string, error) {
	if strings.Contains(url, "/advisories/") {
		parts := strings.Split(url, "/advisories/")
		if len(parts) > 1 {
			idPart := parts[1]
			if idx := strings.Index(idPart, "/"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idx := strings.Index(idPart, "?"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idx := strings.Index(idPart, "#"); idx != -1 {
				idPart = idPart[:idx]
			}
			if idPart == "" {
				return "", fmt.Errorf("advisory ID is empty")
			}
			return idPart, nil
		}
	}

	return "", fmt.Errorf("not a GitHub advisory URL")
}

// extractGitHubReleaseURL extracts owner, repo, and tag from GitHub release URL.
// Pattern: https://github.com/owner/repo/releases/tag/{tag}
func extractGitHubReleaseURL(rawURL string) (string, string, string, error) {
	path, err := githubURLPath(rawURL)
	if err != nil {
		return "", "", "", err
	}

	// Check if the path contains "/releases/tag/"
	if !strings.Contains(path, "/releases/tag/") {
		return "", "", "", errNotGitHubReleaseURL
	}

	// Split at "/releases/tag/"
	parts := strings.SplitN(path, "/releases/tag/", 2)
	if len(parts) < 2 {
		return "", "", "", errNotGitHubReleaseURL
	}

	// The first part contains "owner/repo"
	repoPath := parts[0]
	repoPath = strings.TrimPrefix(repoPath, "/") // Remove leading slash

	// Split repoPath into owner and repo
	repoParts := strings.Split(repoPath, "/")
	if len(repoParts) < 2 {
		return "", "", "", fmt.Errorf("invalid repository path")
	}

	owner := repoParts[0]
	repo := repoParts[1]

	// The second part contains the tag name
	tag := parts[1]
	// Remove any trailing query string or fragment
	if idx := strings.Index(tag, "?"); idx != -1 {
		tag = tag[:idx]
	}
	if idx := strings.Index(tag, "#"); idx != -1 {
		tag = tag[:idx]
	}
	if idx := strings.Index(tag, "/"); idx != -1 {
		tag = tag[:idx]
	}

	if tag == "" {
		return "", "", "", fmt.Errorf("tag name is empty")
	}

	return owner, repo, tag, nil
}
