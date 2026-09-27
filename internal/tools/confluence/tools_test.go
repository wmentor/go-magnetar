package confluence_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func TestExtractPageIDURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "short link URL",
			input:   "https://example.atlassian.net/wiki/x/ABC123",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "short link URL with trailing slash",
			input:   "https://example.atlassian.net/wiki/x/ABC123/",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "short link URL with query parameters",
			input:   "https://example.atlassian.net/wiki/x/ABC123?foo=bar",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "short link URL with fragment",
			input:   "https://example.atlassian.net/wiki/x/ABC123#section",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "short link URL with query and fragment",
			input:   "https://example.atlassian.net/wiki/x/ABC123?foo=bar#section",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "share link URL",
			input:   "https://example.atlassian.net/wiki/p/XYZ789",
			want:    "XYZ789",
			wantErr: false,
		},
		{
			name:    "share link URL with trailing slash",
			input:   "https://example.atlassian.net/wiki/p/XYZ789/",
			want:    "XYZ789",
			wantErr: false,
		},
		{
			name:    "share link URL with query parameters",
			input:   "https://example.atlassian.net/wiki/p/XYZ789?foo=bar",
			want:    "XYZ789",
			wantErr: false,
		},
		{
			name:    "share link URL with fragment",
			input:   "https://example.atlassian.net/wiki/p/XYZ789#section",
			want:    "XYZ789",
			wantErr: false,
		},
		{
			name:    "standard URL with pages path",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/123456/Some+Page+Title",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "standard URL with pages path no title",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/123456",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "standard URL with pages path with query",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/123456?foo=bar",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "standard URL with pages path with fragment",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/123456#section",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "standard URL with pages path with query and fragment",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/123456?foo=bar#section",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "empty page ID in short link",
			input:   "https://example.atlassian.net/wiki/x/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty page ID in share link",
			input:   "https://example.atlassian.net/wiki/p/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty page ID in standard URL",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE/pages/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "not a Confluence URL",
			input:   "https://example.com/article",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Confluence URL without pages path",
			input:   "https://example.atlassian.net/wiki/spaces/SPACE",
			want:    "",
			wantErr: true,
		},
		{
			name:    "short link with complex page ID",
			input:   "https://example.atlassian.net/wiki/x/A4HhC",
			want:    "A4HhC",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := confluence.ExtractPageIDURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPageIDURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractPageIDURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
