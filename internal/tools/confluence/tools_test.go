package confluence_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func TestHTMLFixTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no tags to fix",
			input: "<p>Hello world</p>",
			want:  "<p>Hello world</p>",
		},
		{
			name:  "single br tag",
			input: "<br>",
			want:  "<br/>",
		},
		{
			name:  "single hr tag",
			input: "<hr>",
			want:  "<hr/>",
		},
		{
			name:  "multiple br tags",
			input: "<br><br><br>",
			want:  "<br/><br/><br/>",
		},
		{
			name:  "multiple hr tags",
			input: "<hr><hr>",
			want:  "<hr/><hr/>",
		},
		{
			name:  "mixed br and hr tags",
			input: "<br><hr><br>",
			want:  "<br/><hr/><br/>",
		},
		{
			name:  "br with attributes",
			input: "<br class=\"test\">",
			want:  "<br class=\"test\">",
		},
		{
			name:  "hr with attributes",
			input: "<hr id=\"divider\">",
			want:  "<hr id=\"divider\">",
		},
		{
			name:  "closing br tag removal",
			input: "text</br>more",
			want:  "textmore",
		},
		{
			name:  "closing hr tag removal",
			input: "text</hr>more",
			want:  "textmore",
		},
		{
			name:  "closing br and hr removal",
			input: "text</br></hr>more",
			want:  "textmore",
		},
		{
			name:  "mixed valid and invalid tags",
			input: "<p>text</p><br></br><hr></hr>",
			want:  "<p>text</p><br/><hr/>",
		},
		{
			name:  "real-world HTML with line breaks",
			input: "<p>First line<br>Second line<br>Third line</p>",
			want:  "<p>First line<br/>Second line<br/>Third line</p>",
		},
		{
			name:  "real-world HTML with horizontal rule",
			input: "<p>Section 1</p><hr><p>Section 2</p>",
			want:  "<p>Section 1</p><hr/><p>Section 2</p>",
		},
		{
			name:  "complex HTML with both fixes",
			input: "<div><p>Text<br/>More</p><hr/><p>Next</p></div>",
			want:  "<div><p>Text<br/>More</p><hr/><p>Next</p></div>",
		},
		{
			name:  "html with attributes and line breaks",
			input: "<div class=\"content\"><p>Hello<br>World</p><hr></div>",
			want:  "<div class=\"content\"><p>Hello<br/>World</p><hr/></div>",
		},
		{
			name:  "nested tags with br",
			input: "<ul><li>Item 1<br></li><li>Item 2<br></li></ul>",
			want:  "<ul><li>Item 1<br/></li><li>Item 2<br/></li></ul>",
		},
		{
			name:  "multiple consecutive closing tags",
			input: "text</br></br></hr></hr>more",
			want:  "textmore",
		},
		{
			name:  "br in list items",
			input: "<ul><li>First<br>Second</li></ul>",
			want:  "<ul><li>First<br/>Second</li></ul>",
		},
		{
			name:  "hr between sections",
			input: "<section>Before</section><hr><section>After</section>",
			want:  "<section>Before</section><hr/><section>After</section>",
		},
		{
			name:  "mixed case in input (should not match)",
			input: "<BR><HR>",
			want:  "<BR><HR>",
		},
		{
			name:  "partial matches (should not fix)",
			input: "<brata><hrdata>",
			want:  "<brata><hrdata>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.HTMLFixTags(tt.input)
			if got != tt.want {
				t.Errorf("HtmlFixTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveShortPageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "LZCFFw",
			input:   "LZCFFw",
			want:    "394629165",
			wantErr: false,
		},
		{
			name:    "PAENFQ",
			input:   "PAENFQ",
			want:    "353173820",
			wantErr: false,
		},
		{
			name:    "KYDhBg",
			input:   "KYDhBg",
			want:    "115441705",
			wantErr: false,
		},
		{
			name:    "A4Ag",
			input:   "A4Ag",
			want:    "2129923",
			wantErr: false,
		},
		{
			name:    "S5FCDg",
			input:   "S5FCDg",
			want:    "239243595",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "0",
			wantErr: false,
		},
		{
			name:    "single character A",
			input:   "A",
			want:    "0",
			wantErr: false,
		},
		{
			name:    "single character B",
			input:   "B",
			want:    "4",
			wantErr: false,
		},
		{
			name:    "invalid base64 character",
			input:   "!!!@@@",
			want:    "",
			wantErr: true,
		},
		{
			name:    "two characters AB",
			input:   "AB",
			want:    "4096",
			wantErr: false,
		},
		{
			name:    "longer code A4HhC",
			input:   "A4HhC",
			want:    "148996355",
			wantErr: false,
		},
		{
			name:    "11 characters (no padding needed)",
			input:   "ABCDEFGHIGK",
			want:    "277024768",
			wantErr: false,
		},
		{
			name:    "special characters",
			input:   "_baJCg",
			want:    "176797433",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := confluence.ResolveShortPageID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveShortPageID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveShortPageID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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
