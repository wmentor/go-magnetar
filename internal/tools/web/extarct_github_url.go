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
)

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
