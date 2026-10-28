package web

import (
	"testing"
)

func TestExtractProjectAndMergeRequestFromGitLabURL(t *testing.T) {
	t.Parallel()

	const baseURL = "https://gitlab.example.com"

	tests := []struct {
		name        string
		input       string
		baseURL     string
		wantProject string
		wantMR      string
		wantErr     bool
	}{
		{
			name:        "standard merge request URL",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "base URL with trailing slash",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123",
			baseURL:     baseURL + "/",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "merge request tab after the ID",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123/diffs",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "query parameters",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123?foo=bar",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "fragment",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123#note_456",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "query parameters and fragment",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123?foo=bar#comment-456",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "subgroup with query parameters and fragment",
			input:       "https://gitlab.example.com/group/subgroup/project/-/merge_requests/123?foo=bar#comment-456",
			baseURL:     baseURL,
			wantProject: "group/subgroup/project",
			wantMR:      "123",
		},
		{
			name:        "short merge request URL",
			input:       "https://gitlab.example.com/group/project/-/mr/123",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "short merge request URL with subgroup, query parameters and fragment",
			input:       "https://gitlab.example.com/group/subgroup/project/-/mr/123?foo=bar#comment-456",
			baseURL:     baseURL,
			wantProject: "group/subgroup/project",
			wantMR:      "123",
		},
		{
			name:    "empty merge request ID",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "merge request ID is only a query",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/?foo=bar",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "merge request ID is only a fragment",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/#note_456",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "empty short merge request ID",
			input:   "https://gitlab.example.com/group/project/-/mr/",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "issue URL",
			input:   "https://gitlab.example.com/group/project/-/issues/5",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "project URL",
			input:   "https://gitlab.example.com/group/project",
			baseURL: baseURL,
			wantErr: true,
		},
		{
			name:    "empty URL",
			input:   "",
			baseURL: baseURL,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotProject, gotMR, err := ExtractProjectAndMergeRequestFromGitLabURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ExtractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q, nil), want an error", tt.input, gotProject, gotMR)
				}
				if gotProject != "" || gotMR != "" {
					t.Errorf("ExtractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q), want empty values with the error", tt.input, gotProject, gotMR)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractProjectAndMergeRequestFromGitLabURL(%q) unexpected error: %v", tt.input, err)
			}
			if gotProject != tt.wantProject {
				t.Errorf("project = %q, want %q", gotProject, tt.wantProject)
			}
			if gotMR != tt.wantMR {
				t.Errorf("merge request ID = %q, want %q", gotMR, tt.wantMR)
			}
		})
	}
}
