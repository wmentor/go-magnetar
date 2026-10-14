package web

import "testing"

func TestExtractIssueKeyFromJIRAURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		suffix  string
		wantKey string
		wantErr string
	}{
		{
			name:    "basic issue key",
			suffix:  "GOARCH-60",
			wantKey: "GOARCH-60",
		},
		{
			name:    "different project and issue number",
			suffix:  "PROJECT_2-12345",
			wantKey: "PROJECT_2-12345",
		},
		{
			name:    "trailing slash",
			suffix:  "GOARCH-60/",
			wantKey: "GOARCH-60",
		},
		{
			name:    "additional path segments",
			suffix:  "GOARCH-60/comments/123",
			wantKey: "GOARCH-60",
		},
		{
			name:    "query parameters",
			suffix:  "GOARCH-60?focusedCommentId=123&foo=bar",
			wantKey: "GOARCH-60",
		},
		{
			name:    "fragment",
			suffix:  "GOARCH-60#comment-123",
			wantKey: "GOARCH-60",
		},
		{
			name:    "query parameters and fragment",
			suffix:  "GOARCH-60?foo=bar#comment-123",
			wantKey: "GOARCH-60",
		},
		{
			name:    "path query and fragment",
			suffix:  "GOARCH-60/comments?foo=bar#comment-123",
			wantKey: "GOARCH-60",
		},
		{
			name:    "slash in query parameter",
			suffix:  "GOARCH-60?next=/dashboard",
			wantKey: "GOARCH-60",
		},
		{
			name:    "slash and question mark in fragment",
			suffix:  "GOARCH-60#comment/123?foo=bar",
			wantKey: "GOARCH-60",
		},
		{
			name:    "empty issue key",
			wantErr: "issue key is empty",
		},
		{
			name:    "empty issue key before slash",
			suffix:  "/GOARCH-60",
			wantErr: "issue key is empty",
		},
		{
			name:    "empty issue key before query",
			suffix:  "?issue=GOARCH-60",
			wantErr: "issue key is empty",
		},
		{
			name:    "empty issue key before fragment",
			suffix:  "#GOARCH-60",
			wantErr: "issue key is empty",
		},
		{
			name:    "empty issue key followed by another issues path",
			suffix:  "/issues/OTHER-1",
			wantErr: "issue key is empty",
		},
	}

	for _, path := range []string{"/browse/", "/issues/"} {
		for _, prefix := range []string{"https://jira.example.com", "jira.example.com", ""} {
			for _, tt := range tests {
				t.Run(prefix+path+tt.name, func(t *testing.T) {
					t.Parallel()

					input := prefix + path + tt.suffix
					got, err := extractIssueKeyFromJIRAURL(input)
					if tt.wantErr != "" {
						if err == nil || err.Error() != tt.wantErr {
							t.Fatalf("extractIssueKeyFromJIRAURL(%q) error = %v, want %q", input, err, tt.wantErr)
						}
					} else if err != nil {
						t.Fatalf("extractIssueKeyFromJIRAURL(%q) unexpected error: %v", input, err)
					}
					if got != tt.wantKey {
						t.Errorf("extractIssueKeyFromJIRAURL(%q) = %q, want %q", input, got, tt.wantKey)
					}
				})
			}
		}
	}
}

func TestExtractIssueKeyFromJIRAURLInvalidInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"",
		"not a URL",
		"://",
		"https://jira.example.com",
		"https://jira.example.com/projects/GOARCH",
		"https://jira.example.com/browse",
		"https://jira.example.com/issues",
		"https://jira.example.com/browsing/GOARCH-60",
		"https://jira.example.com/issue/GOARCH-60",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := extractIssueKeyFromJIRAURL(input)
			if err == nil || err.Error() != "not a JIRA issue URL" {
				t.Fatalf("extractIssueKeyFromJIRAURL(%q) error = %v, want not a JIRA issue URL", input, err)
			}
			if got != "" {
				t.Errorf("extractIssueKeyFromJIRAURL(%q) = %q, want an empty key", input, got)
			}
		})
	}
}

func TestExtractIssueKeyFromJIRAURLPatternPrecedence(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"https://jira.example.com/browse/GOARCH-60/issues/OTHER-1",
		"https://jira.example.com/issues/OTHER-1/browse/GOARCH-60",
		"https://jira.example.com/browse/GOARCH-60/browse/OTHER-1",
		"https://jira.example.com/issues/GOARCH-60/issues/OTHER-1",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := extractIssueKeyFromJIRAURL(input)
			if err != nil {
				t.Fatalf("extractIssueKeyFromJIRAURL(%q) unexpected error: %v", input, err)
			}
			if got != "GOARCH-60" {
				t.Errorf("extractIssueKeyFromJIRAURL(%q) = %q, want GOARCH-60", input, got)
			}
		})
	}

	t.Run("empty browse key takes precedence over issues key", func(t *testing.T) {
		t.Parallel()

		input := "https://jira.example.com/issues/OTHER-1/browse/"
		got, err := extractIssueKeyFromJIRAURL(input)
		if err == nil || err.Error() != "issue key is empty" {
			t.Fatalf("extractIssueKeyFromJIRAURL(%q) error = %v, want issue key is empty", input, err)
		}
		if got != "" {
			t.Errorf("extractIssueKeyFromJIRAURL(%q) = %q, want an empty key", input, got)
		}
	})
}
