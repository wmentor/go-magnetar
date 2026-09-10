package web

import (
	"testing"
)

func TestExtractGitHubAdvisoryURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "basic advisory URL",
			input:   "https://github.com/golang/go/advisories/GHSA-1234-5678-9abc",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "advisory URL with trailing slash",
			input:   "https://github.com/golang/go/advisories/GHSA-1234-5678-9abc/",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "advisory URL with query parameters",
			input:   "https://github.com/golang/go/advisories/GHSA-1234-5678-9abc?foo=bar",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "advisory URL with fragment",
			input:   "https://github.com/golang/go/advisories/GHSA-1234-5678-9abc#section",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "security advisories URL",
			input:   "https://github.com/golang/go/security/advisories/GHSA-1234-5678-9abc",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "security advisories URL with query",
			input:   "https://github.com/golang/go/security/advisories/GHSA-1234-5678-9abc?foo=bar",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
		{
			name:    "empty advisory ID",
			input:   "https://github.com/golang/go/advisories/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "not a advisory URL",
			input:   "https://github.com/golang/go/issues/1234",
			want:    "",
			wantErr: true,
		},
		{
			name:    "URL with multiple slashes in advisory part",
			input:   "https://github.com/golang/go/advisories/GHSA-1234-5678-9abc/extra",
			want:    "GHSA-1234-5678-9abc",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractGitHubAdvisoryURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubAdvisoryURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractGitHubAdvisoryURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
