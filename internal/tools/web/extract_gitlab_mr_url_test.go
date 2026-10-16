package web

import (
	"testing"
)

func TestExtractProjectAndMergeRequestFromGitLabURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantProject string
		wantMR      string
		wantErr     bool
	}{
		{
			name:        "standard merge request URL",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "merge request tab after the ID",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123/diffs",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "query parameters",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123?foo=bar",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "fragment",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123#note_456",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "query parameters and fragment",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/123?foo=bar#comment-456",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "subgroup with query parameters and fragment",
			input:       "https://gitlab.example.com/group/subgroup/project/-/merge_requests/123?foo=bar#comment-456",
			wantProject: "group/subgroup/project",
			wantMR:      "123",
		},
		{
			name:        "short merge request URL",
			input:       "https://gitlab.example.com/group/project/-/mr/123",
			wantProject: "group/project",
			wantMR:      "123",
		},
		{
			name:        "short merge request URL with subgroup, query parameters and fragment",
			input:       "https://gitlab.example.com/group/subgroup/project/-/mr/123?foo=bar#comment-456",
			wantProject: "group/subgroup/project",
			wantMR:      "123",
		},
		{
			name:    "empty merge request ID",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/",
			wantErr: true,
		},
		{
			name:    "merge request ID is only a query",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/?foo=bar",
			wantErr: true,
		},
		{
			name:    "merge request ID is only a fragment",
			input:   "https://gitlab.example.com/group/project/-/merge_requests/#note_456",
			wantErr: true,
		},
		{
			name:    "empty short merge request ID",
			input:   "https://gitlab.example.com/group/project/-/mr/",
			wantErr: true,
		},
		{
			name:    "issue URL",
			input:   "https://gitlab.example.com/group/project/-/issues/5",
			wantErr: true,
		},
		{
			name:    "project URL",
			input:   "https://gitlab.example.com/group/project",
			wantErr: true,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotProject, gotMR, err := extractProjectAndMergeRequestFromGitLabURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q, nil), want an error", tt.input, gotProject, gotMR)
				}
				if gotProject != "" || gotMR != "" {
					t.Errorf("extractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q), want empty values with the error", tt.input, gotProject, gotMR)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractProjectAndMergeRequestFromGitLabURL(%q) unexpected error: %v", tt.input, err)
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
