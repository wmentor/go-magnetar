package cve

import (
	"testing"
)

func TestParseCVEID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "CVE identifier",
			input:   "CVE-2021-3114",
			want:    "CVE-2021-3114",
			wantErr: false,
		},
		{
			name:    "GO identifier",
			input:   "GO-2021-0123",
			want:    "GO-2021-0123",
			wantErr: false,
		},
		{
			name:    "GHSA identifier",
			input:   "GHSA-vp9c-fpxx-744v",
			want:    "GHSA-vp9c-fpxx-744v",
			wantErr: false,
		},
		{
			name:    "OSV identifier",
			input:   "OSV-2020-111",
			want:    "OSV-2020-111",
			wantErr: false,
		},
		{
			name:    "GSD identifier",
			input:   "GSD-2021-12345",
			want:    "GSD-2021-12345",
			wantErr: false,
		},
		{
			name:    "ALPINE identifier",
			input:   "ALPINE-2021-12345",
			want:    "ALPINE-2021-12345",
			wantErr: false,
		},
		{
			name:    "ALSA identifier",
			input:   "ALSA-2021-1234",
			want:    "ALSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "ALBA identifier",
			input:   "ALBA-2021-1234",
			want:    "ALBA-2021-1234",
			wantErr: false,
		},
		{
			name:    "ALEA identifier",
			input:   "ALEA-2021-1234",
			want:    "ALEA-2021-1234",
			wantErr: false,
		},
		{
			name:    "ASB-A identifier",
			input:   "ASB-A-2021-1234",
			want:    "ASB-A-2021-1234",
			wantErr: false,
		},
		{
			name:    "PUB-A identifier",
			input:   "PUB-A-2021-1234",
			want:    "PUB-A-2021-1234",
			wantErr: false,
		},
		{
			name:    "AZL identifier",
			input:   "AZL-2021-1234",
			want:    "AZL-2021-1234",
			wantErr: false,
		},
		{
			name:    "BELL identifier",
			input:   "BELL-2021-1234",
			want:    "BELL-2021-1234",
			wantErr: false,
		},
		{
			name:    "BIT identifier",
			input:   "BIT-2021-1234",
			want:    "BIT-2021-1234",
			wantErr: false,
		},
		{
			name:    "BREW identifier",
			input:   "BREW-2021-1234",
			want:    "BREW-2021-1234",
			wantErr: false,
		},
		{
			name:    "CGA identifier",
			input:   "CGA-2021-1234",
			want:    "CGA-2021-1234",
			wantErr: false,
		},
		{
			name:    "CLEANSTART identifier",
			input:   "CLEANSTART-2021-1234",
			want:    "CLEANSTART-2021-1234",
			wantErr: false,
		},
		{
			name:    "CURL identifier",
			input:   "CURL-2021-1234",
			want:    "CURL-2021-1234",
			wantErr: false,
		},
		{
			name:    "DEBIAN identifier",
			input:   "DEBIAN-2021-1234",
			want:    "DEBIAN-2021-1234",
			wantErr: false,
		},
		{
			name:    "DSA identifier",
			input:   "DSA-2021-1234",
			want:    "DSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "DLA identifier",
			input:   "DLA-2021-1234",
			want:    "DLA-2021-1234",
			wantErr: false,
		},
		{
			name:    "DTSA identifier",
			input:   "DTSA-2021-1234",
			want:    "DTSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "ECHO identifier",
			input:   "ECHO-2021-1234",
			want:    "ECHO-2021-1234",
			wantErr: false,
		},
		{
			name:    "EEF identifier",
			input:   "EEF-2021-1234",
			want:    "EEF-2021-1234",
			wantErr: false,
		},
		{
			name:    "ELA identifier",
			input:   "ELA-2021-1234",
			want:    "ELA-2021-1234",
			wantErr: false,
		},
		{
			name:    "HSEC identifier",
			input:   "HSEC-2021-1234",
			want:    "HSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "JLSEC identifier",
			input:   "JLSEC-2021-1234",
			want:    "JLSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "KUBE identifier",
			input:   "KUBE-2021-1234",
			want:    "KUBE-2021-1234",
			wantErr: false,
		},
		{
			name:    "LBSEC identifier",
			input:   "LBSEC-2021-1234",
			want:    "LBSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "LSN identifier",
			input:   "LSN-2021-1234",
			want:    "LSN-2021-1234",
			wantErr: false,
		},
		{
			name:    "MGASA identifier",
			input:   "MGASA-2021-1234",
			want:    "MGASA-2021-1234",
			wantErr: false,
		},
		{
			name:    "MAL identifier",
			input:   "MAL-2021-1234",
			want:    "MAL-2021-1234",
			wantErr: false,
		},
		{
			name:    "MINI identifier",
			input:   "MINI-2021-1234",
			want:    "MINI-2021-1234",
			wantErr: false,
		},
		{
			name:    "OESA identifier",
			input:   "OESA-2021-1234",
			want:    "OESA-2021-1234",
			wantErr: false,
		},
		{
			name:    "OSEC identifier",
			input:   "OSEC-2021-1234",
			want:    "OSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "PHSA identifier",
			input:   "PHSA-2021-1234",
			want:    "PHSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "PSF identifier",
			input:   "PSF-2021-1234",
			want:    "PSF-2021-1234",
			wantErr: false,
		},
		{
			name:    "PYSEC identifier",
			input:   "PYSEC-2021-1234",
			want:    "PYSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "RHSA identifier",
			input:   "RHSA-2021-1234",
			want:    "RHSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "RHBA identifier",
			input:   "RHBA-2021-1234",
			want:    "RHBA-2021-1234",
			wantErr: false,
		},
		{
			name:    "RHEA identifier",
			input:   "RHEA-2021-1234",
			want:    "RHEA-2021-1234",
			wantErr: false,
		},
		{
			name:    "RLSA identifier",
			input:   "RLSA-2021-1234",
			want:    "RLSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "RXSA identifier",
			input:   "RXSA-2021-1234",
			want:    "RXSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "RSEC identifier",
			input:   "RSEC-2021-1234",
			want:    "RSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "ROOT identifier",
			input:   "ROOT-2021-1234",
			want:    "ROOT-2021-1234",
			wantErr: false,
		},
		{
			name:    "RUSTSEC identifier",
			input:   "RUSTSEC-2021-1234",
			want:    "RUSTSEC-2021-1234",
			wantErr: false,
		},
		{
			name:    "SUSE-SU identifier",
			input:   "SUSE-SU-2021-1234",
			want:    "SUSE-SU-2021-1234",
			wantErr: false,
		},
		{
			name:    "SUSE-RU identifier",
			input:   "SUSE-RU-2021-1234",
			want:    "SUSE-RU-2021-1234",
			wantErr: false,
		},
		{
			name:    "SUSE-FU identifier",
			input:   "SUSE-FU-2021-1234",
			want:    "SUSE-FU-2021-1234",
			wantErr: false,
		},
		{
			name:    "SUSE-OU identifier",
			input:   "SUSE-OU-2021-1234",
			want:    "SUSE-OU-2021-1234",
			wantErr: false,
		},
		{
			name:    "openSUSE-SU identifier",
			input:   "openSUSE-SU-2021-1234",
			want:    "openSUSE-SU-2021-1234",
			wantErr: false,
		},
		{
			name:    "UBUNTU identifier",
			input:   "UBUNTU-2021-1234",
			want:    "UBUNTU-2021-1234",
			wantErr: false,
		},
		{
			name:    "USN identifier",
			input:   "USN-2021-1234",
			want:    "USN-2021-1234",
			wantErr: false,
		},
		{
			name:    "V8 identifier",
			input:   "V8-2021-1234",
			want:    "V8-2021-1234",
			wantErr: false,
		},
		{
			name:    "VCPKG identifier",
			input:   "VCPKG-2021-1234",
			want:    "VCPKG-2021-1234",
			wantErr: false,
		},
		{
			name:    "CLSA identifier",
			input:   "CLSA-2021-1234",
			want:    "CLSA-2021-1234",
			wantErr: false,
		},
		{
			name:    "invalid identifier",
			input:   "INVALID-2021-1234",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "identifier with whitespace",
			input:   "  CVE-2021-1234  ",
			want:    "CVE-2021-1234",
			wantErr: false,
		},
		{
			name:    "partial prefix",
			input:   "CVE",
			want:    "",
			wantErr: true,
		},
		{
			name:    "CVE without hyphen",
			input:   "CVE2021-1234",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCVEID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCVEID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseCVEID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
