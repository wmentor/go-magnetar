package mask

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
)

func loadTestConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestNewServiceWithMaskDisabled(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	// Service should be created but with no records
	if len(srv.records) != 0 {
		t.Errorf("expected 0 records when mask is disabled, got %d", len(srv.records))
	}
}

func TestNewServiceWithMaskEnabled(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	// Service should have 6 records: email, github_token, gitlab_token, passport, snils, ssn
	if len(srv.records) != 6 {
		t.Errorf("expected 6 records when mask is enabled, got %d", len(srv.records))
	}
}

func TestNewServiceWithSkipEmail(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: true
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	// Service should have 5 records: github_token, gitlab_token, passport, snils, ssn (no email)
	if len(srv.records) != 5 {
		t.Errorf("expected 5 records when email is skipped, got %d", len(srv.records))
	}
}

func TestProcessEmailMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "single email",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Contact us at test@example.com for help",
			want:  "Contact us at [MASKED_EMAIL] for help",
		},
		{
			name: "multiple emails",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Emails: user1@domain.com, user2@domain.org",
			want:  "Emails: [MASKED_EMAIL], [MASKED_EMAIL]",
		},
		{
			name: "complex email",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Address: user+tag@sub.domain.example.com",
			want:  "Address: [MASKED_EMAIL]",
		},
		{
			name: "no emails",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No email here",
			want:  "No email here",
		},
		{
			name: "email masking disabled",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: true
`,
			input: "Contact: admin@example.com",
			want:  "Contact: admin@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessGitHubTokenMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "github token ghp_",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Token: ghp_" + repeatChar('x', 36),
			want:  "Token: [MASKED_TOKEN]",
		},
		{
			name: "github token github_pat_",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Token: github_pat_" + repeatChar('x', 22) + "_" + repeatChar('x', 59),
			want:  "Token: [MASKED_TOKEN]",
		},
		{
			name: "multiple github tokens",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "ghp_" + repeatChar('x', 36) + " ghp_" + repeatChar('y', 36) + " github_pat_" + repeatChar('a', 22) + "_" + repeatChar('a', 59),
			want:  "[MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN]",
		},
		{
			name: "no github tokens",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No tokens here",
			want:  "No tokens here",
		},
		{
			name: "invalid github token (too short)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "ghp_tooshort",
			want:  "ghp_tooshort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessGitLabTokenMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "gitlab token glpat_",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Token: glpat-" + repeatChar('x', 20),
			want:  "Token: [MASKED_TOKEN]",
		},
		{
			name: "gitlab token gldt_",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Token: gldt-" + repeatChar('x', 20),
			want:  "Token: [MASKED_TOKEN]",
		},
		{
			name: "multiple gitlab tokens",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "glpat-" + repeatChar('a', 20) + " gldt-" + repeatChar('b', 20),
			want:  "[MASKED_TOKEN] [MASKED_TOKEN]",
		},
		{
			name: "no gitlab tokens",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No tokens here",
			want:  "No tokens here",
		},
		{
			name: "invalid gitlab token (too short)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "glpat-tooshort",
			want:  "glpat-tooshort",
		},
		{
			name: "all gitlab token types",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "glpat-" + repeatChar('x', 20) + " gldt-" + repeatChar('x', 20) + " glft-" + repeatChar('x', 20) + " gloas-" + repeatChar('x', 20) + " glrt-" + repeatChar('x', 20) + " glptt-" + repeatChar('x', 20) + " glimt-" + repeatChar('x', 20) + " glcbt-" + repeatChar('x', 20) + " glagent-" + repeatChar('x', 20),
			want:  "[MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN] [MASKED_TOKEN]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessMixedContent(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	input := `User email: john@example.com
GitHub token: ghp_` + repeatChar('x', 36) + `
GitLab token: glpat-` + repeatChar('x', 20) + `
Normal text here`

	output := srv.Process(context.Background(), input)

	// Verify all sensitive data is masked
	if contains(output, "john@example.com") {
		t.Error("email was not masked")
	}
	if contains(output, "ghp_"+repeatChar('x', 36)) {
		t.Error("github token was not masked")
	}
	if contains(output, "glpat-"+repeatChar('x', 20)) {
		t.Error("gitlab token was not masked")
	}
	// Verify non-sensitive text is preserved
	if !contains(output, "Normal text here") {
		t.Error("non-sensitive text was removed")
	}
}

func TestProcessEmptyInput(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	input := ""
	want := ""

	got := srv.Process(context.Background(), input)

	if got != want {
		t.Errorf("Process(\"\") = %q, want %q", got, want)
	}
}

func TestProcessRussianPassportMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "passport with space",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Passport: 1234 567890",
			want:  "Passport: [MASKED_PASSPORT]",
		},
		{
			name: "passport with dash",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Passport: 1234-567890",
			want:  "Passport: [MASKED_PASSPORT]",
		},
		{
			name: "passport with double spaces (11 22 666666 format)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Passport: 11 22 666666",
			want:  "Passport: [MASKED_PASSPORT]",
		},
		{
			name: "multiple passports",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Passports: 1234 567890, 9876 123456",
			want:  "Passports: [MASKED_PASSPORT], [MASKED_PASSPORT]",
		},
		{
			name: "no passport",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No passport here",
			want:  "No passport here",
		},
		{
			name: "invalid passport (wrong format)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Passport: 123 567890",
			want:  "Passport: 123 567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessSNILSMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "snils with space",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SNILS: 123-456-789 01",
			want:  "SNILS: [MASKED_SNILS]",
		},
		{
			name: "snils without space",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SNILS: 123-456-789-01",
			want:  "SNILS: [MASKED_SNILS]",
		},
		{
			name: "multiple snils",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SNILS: 123-456-789 01, 987-654-321 99",
			want:  "SNILS: [MASKED_SNILS], [MASKED_SNILS]",
		},
		{
			name: "no snils",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No SNILS here",
			want:  "No SNILS here",
		},
		{
			name: "invalid snils (wrong format)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SNILS: 12-456-789 01",
			want:  "SNILS: 12-456-789 01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessRussianDocumentsMixed(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	input := `User data:
Passport: 1234 567890
SNILS: 123-456-789 01
Normal text here`

	output := srv.Process(context.Background(), input)

	// Verify all sensitive data is masked
	if contains(output, "1234 567890") {
		t.Error("passport was not masked")
	}
	if contains(output, "123-456-789 01") {
		t.Error("snils was not masked")
	}
	// Verify non-sensitive text is preserved
	if !contains(output, "User data:") {
		t.Error("text 'User data:' was removed")
	}
	if !contains(output, "Normal text here") {
		t.Error("non-sensitive text was removed")
	}
}

func TestProcessSSNMasking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		yaml  string
		input string
		want  string
	}{
		{
			name: "ssn with dashes",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SSN: 123-45-6789",
			want:  "SSN: [MASK_SSN]",
		},
		{
			name: "ssn with spaces",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SSN: 123 45 6789",
			want:  "SSN: [MASK_SSN]",
		},
		{
			name: "multiple ssns in both formats",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "First: 123-45-6789, second: 987 65 4321",
			want:  "First: [MASK_SSN], second: [MASK_SSN]",
		},
		{
			name: "ssn with adjacent punctuation and quotes",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: `SSN "123-45-6789", or '987 65 4321'.`,
			want:  `SSN "[MASK_SSN]", or '[MASK_SSN]'.`,
		},
		{
			name: "ssn with leading and trailing whitespace",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "  123-45-6789  ",
			want:  "  [MASK_SSN]  ",
		},
		{
			name: "no ssn",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "No SSN here",
			want:  "No SSN here",
		},
		{
			name: "invalid ssn (incomplete)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SSN: 123-45-678",
			want:  "SSN: 123-45-678",
		},
		{
			name: "invalid ssn (wrong grouping)",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "SSN: 12-345-6789",
			want:  "SSN: 12-345-6789",
		},
		{
			name: "nine digit number without separators is not an ssn",
			yaml: `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`,
			input: "Account: 123456789",
			want:  "Account: 123456789",
		},
		{
			name: "ssn masking disabled",
			yaml: `version: "1.0"
profile: test

mask:
  enable: false
`,
			input: "SSN: 123-45-6789",
			want:  "SSN: 123-45-6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			cfg := loadTestConfig(t, tt.yaml)
			srv := NewService(cfg)

			got := srv.Process(context.Background(), tt.input)

			if got != tt.want {
				t.Errorf("Process() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSSNFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "dash format", input: "SSN: 123-45-6789", want: "SSN: [MASK_SSN]"},
		{name: "space format", input: "SSN: 123 45 6789", want: "SSN: [MASK_SSN]"},
		{name: "multiple formats", input: "123-45-6789 and 987 65 4321", want: "[MASK_SSN] and [MASK_SSN]"},
		{name: "no ssn", input: "nothing to mask", want: "nothing to mask"},
		{name: "empty input", input: "", want: ""},
		{name: "partial ssn", input: "123-45-678", want: "123-45-678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			if got := SSN(tt.input); got != tt.want {
				t.Errorf("SSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessSSNMixedContent(t *testing.T) {
	t.Parallel()

	yaml := `version: "1.0"
profile: test

mask:
  enable: true
  skip:
    email: false
`

	cfg := loadTestConfig(t, yaml)
	srv := NewService(cfg)

	input := "User email: john@example.com\nSSN: 123-45-6789\nNormal text here"

	output := srv.Process(context.Background(), input)

	if contains(output, "123-45-6789") {
		t.Error("ssn was not masked")
	}
	if contains(output, "john@example.com") {
		t.Error("email was not masked")
	}
	if !contains(output, "Normal text here") {
		t.Error("non-sensitive text was removed")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to repeat a character
func repeatChar(c byte, n int) string {
	result := make([]byte, n)
	for i := range n {
		result[i] = c
	}
	return string(result)
}
