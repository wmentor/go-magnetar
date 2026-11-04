// Package github unit tests for the GitHub tools.
//
// This file contains comprehensive unit tests for the formatTreeMarkdown method
// which formats GitHub repository tree data as Markdown. The tests cover:
//   - Empty file lists
//   - Single files
//   - Multiple files
//   - Files with paths
//   - Directories (tree type exclusion)
//   - Mixed files and directories
//   - Edge cases (zero size, large files)
//
// Tests follow table-driven patterns with t.Parallel() for concurrent execution.

package github

import (
	"testing"
)

func TestFormatTreeMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			Sha         string `json:"sha"`
			Size        int    `json:"size"`
			URL         string `json:"url"`
			HTMLURL     string `json:"html_url"`
			GitURL      string `json:"git_url"`
			DownloadURL string `json:"download_url"`
			Type        string `json:"type"`
			Encoding    string `json:"encoding,omitempty"`
			Links       struct {
				Self string `json:"self"`
				Git  string `json:"git"`
				HTML string `json:"html"`
			} `json:"_links"`
		}
		path string
		want string
	}{
		{
			name: "empty file list",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n",
		},
		{
			name: "single file",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "README.md",
					Path:        "README.md",
					Sha:         "abc123",
					Size:        1024,
					URL:         "https://api.github.com/repos/test/test/contents/README.md",
					HTMLURL:     "https://github.com/test/test/blob/main/README.md",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/abc123",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/README.md",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/README.md",
						Git:  "https://api.github.com/repos/test/test/git/blobs/abc123",
						HTML: "https://github.com/test/test/blob/main/README.md",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| README.md | 1024 bytes |\n",
		},
		{
			name: "multiple files",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "README.md",
					Path:        "README.md",
					Sha:         "abc123",
					Size:        1024,
					URL:         "https://api.github.com/repos/test/test/contents/README.md",
					HTMLURL:     "https://github.com/test/test/blob/main/README.md",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/abc123",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/README.md",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/README.md",
						Git:  "https://api.github.com/repos/test/test/git/blobs/abc123",
						HTML: "https://github.com/test/test/blob/main/README.md",
					},
				},
				{
					Name:        "main.go",
					Path:        "main.go",
					Sha:         "def456",
					Size:        2048,
					URL:         "https://api.github.com/repos/test/test/contents/main.go",
					HTMLURL:     "https://github.com/test/test/blob/main/main.go",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/def456",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/main.go",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/main.go",
						Git:  "https://api.github.com/repos/test/test/git/blobs/def456",
						HTML: "https://github.com/test/test/blob/main/main.go",
					},
				},
				{
					Name:        "utils.go",
					Path:        "utils.go",
					Sha:         "ghi789",
					Size:        512,
					URL:         "https://api.github.com/repos/test/test/contents/utils.go",
					HTMLURL:     "https://github.com/test/test/blob/main/utils.go",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/ghi789",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/utils.go",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/utils.go",
						Git:  "https://api.github.com/repos/test/test/git/blobs/ghi789",
						HTML: "https://github.com/test/test/blob/main/utils.go",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| README.md | 1024 bytes |\n| main.go | 2048 bytes |\n| utils.go | 512 bytes |\n",
		},
		{
			name: "file with path",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "test.go",
					Path:        "internal/test/test.go",
					Sha:         "jkl012",
					Size:        3072,
					URL:         "https://api.github.com/repos/test/test/contents/internal/test/test.go",
					HTMLURL:     "https://github.com/test/test/blob/main/internal/test/test.go",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/jkl012",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/internal/test/test.go",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/internal/test/test.go",
						Git:  "https://api.github.com/repos/test/test/git/blobs/jkl012",
						HTML: "https://github.com/test/test/blob/main/internal/test/test.go",
					},
				},
			},
			path: "internal/test",
			want: "**Contents of internal/test**\n\n| Name | Size |\n|------|------|\n| internal/test/test.go | 3072 bytes |\n",
		},
		{
			name: "directory (tree type) should be excluded",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "cmd",
					Path:        "cmd",
					Sha:         "mno345",
					Size:        0,
					URL:         "https://api.github.com/repos/test/test/contents/cmd",
					HTMLURL:     "https://github.com/test/test/tree/main/cmd",
					GitURL:      "https://api.github.com/repos/test/test/git/trees/mno345",
					DownloadURL: "",
					Type:        "tree",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/cmd",
						Git:  "https://api.github.com/repos/test/test/git/trees/mno345",
						HTML: "https://github.com/test/test/tree/main/cmd",
					},
				},
				{
					Name:        "main.go",
					Path:        "main.go",
					Sha:         "pqr678",
					Size:        1536,
					URL:         "https://api.github.com/repos/test/test/contents/main.go",
					HTMLURL:     "https://github.com/test/test/blob/main/main.go",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/pqr678",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/main.go",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/main.go",
						Git:  "https://api.github.com/repos/test/test/git/blobs/pqr678",
						HTML: "https://github.com/test/test/blob/main/main.go",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| main.go | 1536 bytes |\n",
		},
		{
			name: "mixed files and directories",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "docs",
					Path:        "docs",
					Sha:         "stu901",
					Size:        0,
					URL:         "https://api.github.com/repos/test/test/contents/docs",
					HTMLURL:     "https://github.com/test/test/tree/main/docs",
					GitURL:      "https://api.github.com/repos/test/test/git/trees/stu901",
					DownloadURL: "",
					Type:        "tree",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/docs",
						Git:  "https://api.github.com/repos/test/test/git/trees/stu901",
						HTML: "https://github.com/test/test/tree/main/docs",
					},
				},
				{
					Name:        "README.md",
					Path:        "README.md",
					Sha:         "vwx234",
					Size:        2560,
					URL:         "https://api.github.com/repos/test/test/contents/README.md",
					HTMLURL:     "https://github.com/test/test/blob/main/README.md",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/vwx234",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/README.md",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/README.md",
						Git:  "https://api.github.com/repos/test/test/git/blobs/vwx234",
						HTML: "https://github.com/test/test/blob/main/README.md",
					},
				},
				{
					Name:        "pkg",
					Path:        "pkg",
					Sha:         "yza567",
					Size:        0,
					URL:         "https://api.github.com/repos/test/test/contents/pkg",
					HTMLURL:     "https://github.com/test/test/tree/main/pkg",
					GitURL:      "https://api.github.com/repos/test/test/git/trees/yza567",
					DownloadURL: "",
					Type:        "tree",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/pkg",
						Git:  "https://api.github.com/repos/test/test/git/trees/yza567",
						HTML: "https://github.com/test/test/tree/main/pkg",
					},
				},
				{
					Name:        "main.go",
					Path:        "main.go",
					Sha:         "bcd890",
					Size:        1024,
					URL:         "https://api.github.com/repos/test/test/contents/main.go",
					HTMLURL:     "https://github.com/test/test/blob/main/main.go",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/bcd890",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/main.go",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/main.go",
						Git:  "https://api.github.com/repos/test/test/git/blobs/bcd890",
						HTML: "https://github.com/test/test/blob/main/main.go",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| README.md | 2560 bytes |\n| main.go | 1024 bytes |\n",
		},
		{
			name: "file with zero size",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "empty.txt",
					Path:        "empty.txt",
					Sha:         "efg123",
					Size:        0,
					URL:         "https://api.github.com/repos/test/test/contents/empty.txt",
					HTMLURL:     "https://github.com/test/test/blob/main/empty.txt",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/efg123",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/empty.txt",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/empty.txt",
						Git:  "https://api.github.com/repos/test/test/git/blobs/efg123",
						HTML: "https://github.com/test/test/blob/main/empty.txt",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| empty.txt | 0 bytes |\n",
		},
		{
			name: "file with large size",
			files: []struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Sha         string `json:"sha"`
				Size        int    `json:"size"`
				URL         string `json:"url"`
				HTMLURL     string `json:"html_url"`
				GitURL      string `json:"git_url"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
				Encoding    string `json:"encoding,omitempty"`
				Links       struct {
					Self string `json:"self"`
					Git  string `json:"git"`
					HTML string `json:"html"`
				} `json:"_links"`
			}{
				{
					Name:        "large.bin",
					Path:        "large.bin",
					Sha:         "hij456",
					Size:        10485760, // 10 MB
					URL:         "https://api.github.com/repos/test/test/contents/large.bin",
					HTMLURL:     "https://github.com/test/test/blob/main/large.bin",
					GitURL:      "https://api.github.com/repos/test/test/git/blobs/hij456",
					DownloadURL: "https://raw.githubusercontent.com/test/test/main/large.bin",
					Type:        "file",
					Encoding:    "utf-8",
					Links: struct {
						Self string `json:"self"`
						Git  string `json:"git"`
						HTML string `json:"html"`
					}{
						Self: "https://api.github.com/repos/test/test/contents/large.bin",
						Git:  "https://api.github.com/repos/test/test/git/blobs/hij456",
						HTML: "https://github.com/test/test/blob/main/large.bin",
					},
				},
			},
			path: "",
			want: "**Contents of **\n\n| Name | Size |\n|------|------|\n| large.bin | 10485760 bytes |\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a minimal GitHubTools instance (formatTreeMarkdown doesn't use it)
			ghTools := &GitHubTools{}

			got := ghTools.formatTreeMarkdown(tt.files, tt.path)
			if got != tt.want {
				t.Errorf("formatTreeMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
