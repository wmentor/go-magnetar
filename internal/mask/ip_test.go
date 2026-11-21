package mask

import (
	"context"
	"strings"
	"testing"
)

func newTestConfig(t *testing.T, skipIPs []string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("version: \"1.0\"\nmask:\n  enable: true\n")
	if len(skipIPs) > 0 {
		b.WriteString("  skip:\n    ip:\n")
		for _, ip := range skipIPs {
			b.WriteString("      - \"" + ip + "\"\n")
		}
	}
	return b.String()
}

func TestIPMasking_IPv4AndIPv6(t *testing.T) {
	srv := NewService(loadTestConfig(t, newTestConfig(t, nil)))
	ctx := context.Background()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ipv4 address",
			input: "connect to 192.168.1.42 now",
			want:  "connect to [MASKED_IP] now",
		},
		{
			name:  "ipv6 full address",
			input: "host 2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want:  "host [MASKED_IP]",
		},
		{
			name:  "ipv6 compressed address",
			input: "host 2001:db8::1",
			want:  "host [MASKED_IP]",
		},
		{
			name:  "invalid ipv4 octets left untouched",
			input: "version 999.999.999.999",
			want:  "version 999.999.999.999",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := srv.Process(ctx, tc.input)
			if got != tc.want {
				t.Fatalf("Process(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIPMasking_SkipRules(t *testing.T) {
	cfg := loadTestConfig(t, newTestConfig(t, []string{
		"127.0.0.1",
		"192.168.1.0/24",
		"2001:db8::/32",
		"10.0.*.*",
	}))
	srv := NewService(cfg)
	ctx := context.Background()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single ip skipped",
			input: "localhost 127.0.0.1",
			want:  "localhost 127.0.0.1",
		},
		{
			name:  "cidr ipv4 skipped",
			input: "lan 192.168.1.55",
			want:  "lan 192.168.1.55",
		},
		{
			name:  "cidr ipv4 outside range masked",
			input: "other 192.168.2.55",
			want:  "other [MASKED_IP]",
		},
		{
			name:  "cidr ipv6 skipped",
			input: "doc 2001:db8::42",
			want:  "doc 2001:db8::42",
		},
		{
			name:  "wildcard ipv4 skipped",
			input: "internal 10.0.12.34",
			want:  "internal 10.0.12.34",
		},
		{
			name:  "public ip masked",
			input: "public 8.8.8.8",
			want:  "public [MASKED_IP]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := srv.Process(ctx, tc.input)
			if got != tc.want {
				t.Fatalf("Process(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
