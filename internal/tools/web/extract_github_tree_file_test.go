package web

import (
	"testing"
)

func TestExtractGitHubTreeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantBranch string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "basic tree URL",
			input:      "https://github.com/golang/go/tree/master",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "tree URL with path",
			input:      "https://github.com/golang/go/tree/master/src/cmd",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src/cmd",
			wantErr:    false,
		},
		{
			name:       "tree URL with trailing slash",
			input:      "https://github.com/golang/go/tree/master/",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "tree URL with query parameters on path",
			input:      "https://github.com/golang/go/tree/master/src?foo=bar",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
			wantErr:    false,
		},
		{
			name:       "tree URL with query on branch (no path)",
			input:      "https://github.com/golang/go/tree/master?foo=bar",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "commits URL without path",
			input:      "https://github.com/golang/go/commits/master",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "commits URL with path",
			input:      "https://github.com/golang/go/commits/master/src",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
			wantErr:    false,
		},
		{
			name:    "blob URL is not a tree URL",
			input:   "https://github.com/golang/go/blob/master/README.md",
			wantErr: true,
		},
		{
			name:    "issue URL is not a tree URL",
			input:   "https://github.com/golang/go/issues/1234",
			wantErr: true,
		},
		{
			name:       "tree URL with fragment",
			input:      "https://github.com/golang/go/tree/master/src#readme",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
		},
		{
			name:       "tree URL on www.github.com",
			input:      "https://www.github.com/golang/go/tree/master/src",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantPath:   "src",
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/tree/master",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/tree/master",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, repo, branch, path, err := ExtractGitHubTreeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubTreeURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || branch != tt.wantBranch || path != tt.wantPath {
				t.Errorf("ExtractGitHubTreeURL(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tt.input, owner, repo, branch, path, tt.wantOwner, tt.wantRepo, tt.wantBranch, tt.wantPath)
			}
		})
	}
}

func TestExtractGitHubFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantBranch string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "basic blob URL",
			input:      "https://github.com/golang/go/blob/master/README.md",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with nested path",
			input:      "https://github.com/golang/go/blob/master/src/cmd/go/main.go",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "src/cmd/go/main.go",
			wantErr:    false,
		},
		{
			name:       "blob URL with trailing slash",
			input:      "https://github.com/golang/go/blob/master/README.md/",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with query parameters",
			input:      "https://github.com/golang/go/blob/master/README.md?raw=1",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:       "blob URL with fragment",
			input:      "https://github.com/golang/go/blob/master/README.md#L10",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "README.md",
			wantErr:    false,
		},
		{
			name:    "blob URL without file path",
			input:   "https://github.com/golang/go/blob/master",
			wantErr: true,
		},
		{
			name:    "tree URL is not a blob URL",
			input:   "https://github.com/golang/go/tree/master/src",
			wantErr: true,
		},
		{
			name:       "blob URL with line range fragment",
			input:      "https://github.com/golang/go/blob/master/src/cmd/go/main.go#L10-L20",
			wantOwner:  "golang",
			wantRepo:   "go",
			wantBranch: "master",
			wantFile:   "src/cmd/go/main.go",
		},
		{
			name:    "not a GitHub URL",
			input:   "https://gitlab.com/golang/go/blob/master/README.md",
			wantErr: true,
		},
		{
			name:    "non-GitHub host containing github.com",
			input:   "https://notgithub.com/golang/go/blob/master/README.md",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, repo, branch, file, err := ExtractGitHubFileURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractGitHubFileURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || branch != tt.wantBranch || file != tt.wantFile {
				t.Errorf("ExtractGitHubFileURL(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tt.input, owner, repo, branch, file, tt.wantOwner, tt.wantRepo, tt.wantBranch, tt.wantFile)
			}
		})
	}
}
