package web

import (
	"strings"
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
		wantList    bool
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
			name:        "merge request list URL (no trailing slash)",
			input:       "https://gitlab.example.com/group/project/-/merge_requests",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL with trailing slash (empty ID)",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL (no ID)",
			input:       "https://gitlab.example.com/group/project/-/merge_requests",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL with query",
			input:       "https://gitlab.example.com/group/project/-/merge_requests?state=opened&author=alice",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "short merge request list URL (mr)",
			input:       "https://gitlab.example.com/group/project/-/mr",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "short merge request list URL with fragment",
			input:       "https://gitlab.example.com/group/project/-/mr#filter",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL with subgroup",
			input:       "https://gitlab.example.com/group/subgroup/project/-/merge_requests?state=opened",
			baseURL:     baseURL,
			wantProject: "group/subgroup/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL with trailing slash and query",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/?foo=bar",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "merge request list URL with trailing slash and fragment",
			input:       "https://gitlab.example.com/group/project/-/merge_requests/#note_456",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
		},
		{
			name:        "short merge request list URL with trailing slash (empty ID)",
			input:       "https://gitlab.example.com/group/project/-/mr/",
			baseURL:     baseURL,
			wantProject: "group/project",
			wantMR:      "",
			wantList:    true,
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

			gotProject, gotMR, isList, err := ExtractProjectAndMergeRequestFromGitLabURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ExtractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q, %v, nil), want an error", tt.input, gotProject, gotMR, isList)
				}
				if gotProject != "" || gotMR != "" || isList != tt.wantList {
					t.Errorf("ExtractProjectAndMergeRequestFromGitLabURL(%q) = (%q, %q, %v), want empty values with the error", tt.input, gotProject, gotMR, isList)
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
			if isList != tt.wantList {
				t.Errorf("isList = %v, want %v", isList, tt.wantList)
			}
		})
	}
}

// TestExtractProjectAndMergeRequestFromGitLabURLErrorIncludesURL verifies that
// when a URL cannot be recognized as a GitLab merge request URL, the returned
// error names the offending URL. Without it, callers and log readers get a
// generic message with no way to tell which input failed validation.
func TestExtractProjectAndMergeRequestFromGitLabURLErrorIncludesURL(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"https://gitlab.example.com/group/project",
		"https://gitlab.example.com/group/project/-/issues/5",
		"not-even-a-url",
	}

	for _, input := range inputs {
		_, _, _, err := ExtractProjectAndMergeRequestFromGitLabURL(input)
		if err == nil {
			t.Fatalf("ExtractProjectAndMergeRequestFromGitLabURL(%q) returned no error, want one", input)
		}
		if !strings.Contains(err.Error(), input) {
			t.Errorf("error %q does not include the offending URL %q", err.Error(), input)
		}
	}
}
