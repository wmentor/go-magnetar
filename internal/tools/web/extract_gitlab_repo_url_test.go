package web

import (
	"testing"
)

func TestExtractGitLabRepoURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "basic project URL",
			input: "https://gitlab.example.com/group/project",
			want:  "group/project",
		},
		{
			name:  "project URL with namespace",
			input: "https://gitlab.example.com/namespace/subgroup/project",
			want:  "namespace/subgroup/project",
		},
		{
			name:  "project URL with trailing slash",
			input: "https://gitlab.example.com/group/project/",
			want:  "group/project",
		},
		{
			name:  "project URL on www.gitlab.com",
			input: "https://www.gitlab.example.com/group/project",
			want:  "group/project",
		},
		{
			name:    "project URL without path",
			input:   "https://gitlab.example.com/",
			wantErr: true,
		},
		{
			name:    "project URL without project",
			input:   "https://gitlab.example.com/group/",
			wantErr: true,
		},
		{
			name:  "project URL with /-/edit (path returned)",
			input: "https://gitlab.example.com/group/project/-/edit",
			want:  "group/project/-/edit",
		},
		{
			name:  "project URL with /-/tree (path returned)",
			input: "https://gitlab.example.com/group/project/-/tree/main",
			want:  "group/project/-/tree/main",
		},
		{
			name:  "project URL with /-/blob (path returned)",
			input: "https://gitlab.example.com/group/project/-/blob/main/file.go",
			want:  "group/project/-/blob/main/file.go",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			input:   "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractGitLabRepoURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractGitLabRepoURL(%q) = %q, want an error", tt.input, got)
				}
				if got != "" {
					t.Errorf("extractGitLabRepoURL(%q) = %q, want empty value with the error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractGitLabRepoURL(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("project = %q, want %q", got, tt.want)
			}
		})
	}
}
