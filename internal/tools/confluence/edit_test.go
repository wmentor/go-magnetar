package confluence_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func TestExtractPageIDURL_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "full URL with path",
			input:   "https://confluence.example.com/wiki/spaces/TEAM/pages/123456/Some+Page+Title+With+Details",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "short link with trailing slash",
			input:   "https://confluence.example.com/wiki/x/ABC123/",
			want:    "ABC123",
			wantErr: false,
		},
		{
			name:    "share link with query",
			input:   "https://confluence.example.com/wiki/p/XYZ789?workspace=abc",
			want:    "XYZ789",
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
