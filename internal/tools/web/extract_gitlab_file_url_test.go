package web

import (
	"testing"
)

func TestExtractGitLabFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  struct {
			project string
			branch  string
			file    string
		}
		wantErr bool
	}{
		{
			name:  "basic file URL",
			input: "https://gitlab.example.com/group/project/-/blob/main/src/main.go",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "main",
				file:    "src/main.go",
			},
		},
		{
			name:  "file URL with nested path",
			input: "https://gitlab.example.com/group/project/-/blob/develop/path/to/file.go",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "develop",
				file:    "path/to/file.go",
			},
		},
		{
			name:  "file URL with query parameters",
			input: "https://gitlab.example.com/group/project/-/blob/main/file.go?line=10",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "main",
				file:    "file.go",
			},
		},
		{
			name:  "file URL with fragment",
			input: "https://gitlab.example.com/group/project/-/blob/main/file.go#L20",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "main",
				file:    "file.go",
			},
		},
		{
			name:  "file URL with query parameters and fragment",
			input: "https://gitlab.example.com/group/project/-/blob/main/file.go?line=10#L20",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "main",
				file:    "file.go",
			},
		},
		{
			name:  "file URL on www.gitlab.com",
			input: "https://www.gitlab.example.com/group/project/-/blob/main/file.go",
			want: struct {
				project string
				branch  string
				file    string
			}{
				project: "group/project",
				branch:  "main",
				file:    "file.go",
			},
		},
		{
			name:    "blob URL is not a file URL",
			input:   "https://gitlab.example.com/group/project/-/tree/main",
			wantErr: true,
		},
		{
			name:    "file URL without file path",
			input:   "https://gitlab.example.com/group/project/-/blob/main/",
			wantErr: true,
		},
		{
			name:    "not a GitLab file URL",
			input:   "https://github.com/group/project/blob/main/file.go",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
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

			gotProject, gotBranch, gotFile, err := extractGitLabFileURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractGitLabFileURL(%q) = (%q, %q, %q, nil), want an error", tt.input, gotProject, gotBranch, gotFile)
				}
				if gotProject != "" || gotBranch != "" || gotFile != "" {
					t.Errorf("extractGitLabFileURL(%q) = (%q, %q, %q), want empty values with the error", tt.input, gotProject, gotBranch, gotFile)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractGitLabFileURL(%q) unexpected error: %v", tt.input, err)
			}
			if gotProject != tt.want.project {
				t.Errorf("project = %q, want %q", gotProject, tt.want.project)
			}
			if gotBranch != tt.want.branch {
				t.Errorf("branch = %q, want %q", gotBranch, tt.want.branch)
			}
			if gotFile != tt.want.file {
				t.Errorf("file = %q, want %q", gotFile, tt.want.file)
			}
		})
	}
}
