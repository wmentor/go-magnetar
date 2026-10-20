package web

import (
	"testing"
)

func TestExtractGitLabTreeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  struct {
			project string
			branch  string
			path    string
		}
		wantErr bool
	}{
		{
			name:  "basic tree URL",
			input: "https://gitlab.example.com/group/project/-/tree/main",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "",
			},
		},
		{
			name:  "tree URL with path",
			input: "https://gitlab.example.com/group/project/-/tree/main/src",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "src",
			},
		},
		{
			name:  "commits URL without path",
			input: "https://gitlab.example.com/group/project/-/commits/main",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "",
			},
		},
		{
			name:  "commits URL with path",
			input: "https://gitlab.example.com/group/project/-/commits/main/src",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "src",
			},
		},
		{
			name:  "tree URL with query on branch (no path)",
			input: "https://gitlab.example.com/group/project/-/tree/main?foo=bar",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "",
			},
		},
		{
			name:  "tree URL with query parameters on path",
			input: "https://gitlab.example.com/group/project/-/tree/main/src?foo=bar",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "src",
			},
		},
		{
			name:  "tree URL with fragment",
			input: "https://gitlab.example.com/group/project/-/tree/main#files",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "",
			},
		},
		{
			name:  "tree URL with path and fragment",
			input: "https://gitlab.example.com/group/project/-/tree/main/src#files",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "src",
			},
		},
		{
			name:  "tree URL on www.gitlab.com",
			input: "https://www.gitlab.example.com/group/project/-/tree/main",
			want: struct {
				project string
				branch  string
				path    string
			}{
				project: "group/project",
				branch:  "main",
				path:    "",
			},
		},
		{
			name:    "blob URL is not a tree URL",
			input:   "https://gitlab.example.com/group/project/-/blob/main/file.go",
			wantErr: true,
		},
		{
			name:    "not a GitLab tree URL",
			input:   "https://github.com/group/project/tree/main",
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

			gotProject, gotBranch, gotPath, err := extractGitLabTreeURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractGitLabTreeURL(%q) = (%q, %q, %q, nil), want an error", tt.input, gotProject, gotBranch, gotPath)
				}
				if gotProject != "" || gotBranch != "" || gotPath != "" {
					t.Errorf("extractGitLabTreeURL(%q) = (%q, %q, %q), want empty values with the error", tt.input, gotProject, gotBranch, gotPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractGitLabTreeURL(%q) unexpected error: %v", tt.input, err)
			}
			if gotProject != tt.want.project {
				t.Errorf("project = %q, want %q", gotProject, tt.want.project)
			}
			if gotBranch != tt.want.branch {
				t.Errorf("branch = %q, want %q", gotBranch, tt.want.branch)
			}
			if gotPath != tt.want.path {
				t.Errorf("path = %q, want %q", gotPath, tt.want.path)
			}
		})
	}
}
