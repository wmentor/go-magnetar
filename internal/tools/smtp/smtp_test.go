package smtp

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-gomail/gomail"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func newTestTool(t *testing.T, yaml string) *Tool {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	p := printer.GetDefault()

	// Create root FS for tests that need file operations.
	rootFS, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootFS.Close() })

	state := &plugin.State{
		Config:  cfg,
		Printer: p,
		Root:    rootFS,
	}

	return New(state)
}

func TestToolName(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	if got := tool.Name(); got != "smtp" {
		t.Errorf("Name() = %q, want %q", got, "smtp")
	}
}

func TestToolDescription(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestExecNilParams(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	_, err := tool.Exec(t.Context(), nil)
	if err == nil {
		t.Error("Exec(nil) should return error")
	}
}

func TestExecEmptyTo(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	params := &Params{To: "", Subject: "Test", Body: "Hello"}
	_, err := tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec with empty To should return error")
	}
}

func TestExecEmptySubject(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	params := &Params{To: "test@example.com", Subject: "", Body: "Hello"}
	_, err := tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec with empty Subject should return error")
	}
}

func TestExecEmptyBody(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, "")
	params := &Params{To: "test@example.com", Subject: "Test", Body: ""}
	_, err := tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec with empty Body should return error")
	}
}

func TestExecMissingHost(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, `
smtp:
  enable: true
  host: ""
  from_email: "sender@example.com"
`)
	params := &Params{To: "test@example.com", Subject: "Test", Body: "Hello"}
	_, err := tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec without host should return error")
	}
}

func TestExecMissingFromEmail(t *testing.T) {
	t.Parallel()

	tool := newTestTool(t, `
smtp:
  enable: true
  host: "smtp.example.com"
  from_email: ""
`)
	params := &Params{To: "test@example.com", Subject: "Test", Body: "Hello"}
	_, err := tool.Exec(t.Context(), params)
	if err == nil {
		t.Error("Exec without from_email should return error")
	}
}

func TestConvertMarkdownToHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple paragraph",
			input:    "Hello, World!",
			contains: "<p>Hello, World!</p>",
		},
		{
			name:     "heading",
			input:    "# Title",
			contains: "<h1>Title</h1>",
		},
		{
			name:     "bold text",
			input:    "**bold**",
			contains: "<strong>bold</strong>",
		},
		{
			name:     "italic text",
			input:    "*italic*",
			contains: "<em>italic</em>",
		},
		{
			name:     "link",
			input:    "[link](https://example.com)",
			contains: `<a href="https://example.com">link</a>`,
		},
		{
			name:     "unordered list",
			input:    "- item1\n- item2",
			contains: "<li>item1</li>",
		},
		{
			name:     "code block",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			contains: "<code",
		},
		{
			name:     "table (GFM)",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: "<table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			result, err := convertMarkdownToHTML(tt.input)
			if err != nil {
				t.Fatalf("convertMarkdownToHTML() error: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("convertMarkdownToHTML() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestParseRecipients(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single recipient",
			input: "test@example.com",
			want:  []string{"test@example.com"},
		},
		{
			name:  "multiple recipients",
			input: "a@example.com, b@example.com, c@example.com",
			want:  []string{"a@example.com", "b@example.com", "c@example.com"},
		},
		{
			name:  "with extra spaces",
			input: "  a@example.com ,  b@example.com  ",
			want:  []string{"a@example.com", "b@example.com"},
		},
		{
			name:  "empty parts ignored",
			input: "a@example.com,,b@example.com",
			want:  []string{"a@example.com", "b@example.com"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := parseRecipients(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseRecipients() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseRecipients()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildGomailMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fromName    string
		fromEmail   string
		to          string
		cc          []string
		subject     string
		htmlBody    string
		attachments map[string][]byte // filename -> content
		contains    []string
		notContains []string
	}{
		{
			name:      "with from name and cc",
			fromName:  "Test Sender",
			fromEmail: "sender@example.com",
			to:        "recipient@example.com",
			cc:        []string{"cc1@example.com", "cc2@example.com"},
			subject:   "Test Subject",
			htmlBody:  "<p>Hello</p>",
			contains: []string{
				`From: "Test Sender" <sender@example.com>`,
				"To: recipient@example.com",
				"Cc: cc1@example.com, cc2@example.com",
				"Subject: Test Subject",
				"Content-Type: text/html",
				"<p>Hello</p>",
			},
		},
		{
			name:        "without from name and without cc",
			fromName:    "",
			fromEmail:   "sender@example.com",
			to:          "recipient@example.com",
			cc:          nil,
			subject:     "Test",
			htmlBody:    "<p>Body</p>",
			attachments: nil,
			contains: []string{
				"From: sender@example.com",
				"To: recipient@example.com",
				"<p>Body</p>",
			},
			notContains: []string{
				"Cc:",
			},
		},
		{
			name:      "with cc including sender",
			fromName:  "",
			fromEmail: "sender@example.com",
			to:        "a@example.com",
			cc:        []string{"b@example.com", "sender@example.com"},
			subject:   "Test",
			htmlBody:  "<p>Body</p>",
			contains: []string{
				"To: a@example.com",
				"Cc: b@example.com, sender@example.com",
			},
		},
		{
			name:      "with attachment",
			fromName:  "Sender",
			fromEmail: "sender@example.com",
			to:        "recipient@example.com",
			subject:   "With Attachment",
			htmlBody:  "<p>See attached</p>",
			attachments: map[string][]byte{
				"test.txt": []byte("Hello, World!"),
			},
			contains: []string{
				"multipart/mixed",
				"Content-Type: text/html",
				"<p>See attached</p>",
				"test.txt",
				"SGVsbG8sIFdvcmxkIQ==", // base64 of "Hello, World!"
			},
		},
		{
			name:      "with multiple attachments",
			fromName:  "",
			fromEmail: "sender@example.com",
			to:        "recipient@example.com",
			subject:   "Multiple Attachments",
			htmlBody:  "<p>See attached</p>",
			attachments: map[string][]byte{
				"file1.txt": []byte("Content 1"),
				"file2.pdf": []byte("PDF content"),
			},
			contains: []string{
				"multipart/mixed",
				"file1.txt",
				"file2.pdf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Build gomail message.
			msg := gomail.NewMessage()
			if tt.fromName != "" {
				msg.SetAddressHeader("From", tt.fromEmail, tt.fromName)
			} else {
				msg.SetHeader("From", tt.fromEmail)
			}
			msg.SetHeader("To", tt.to)
			if len(tt.cc) > 0 {
				msg.SetHeader("Cc", tt.cc...)
			}
			msg.SetHeader("Subject", tt.subject)
			msg.SetBody("text/html", tt.htmlBody)

			// Add attachments if any.
			for filename, data := range tt.attachments {
				msg.Attach(filename, gomail.SetCopyFunc(func(w io.Writer) error {
					_, err := w.Write(data)
					return err
				}))
			}

			// Write message to buffer.
			var buf strings.Builder
			if _, err := msg.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo() error: %v", err)
			}
			msgStr := buf.String()

			for _, want := range tt.contains {
				if !strings.Contains(msgStr, want) {
					t.Errorf("message = %q, want to contain %q", msgStr, want)
				}
			}
			for _, notWant := range tt.notContains {
				if strings.Contains(msgStr, notWant) {
					t.Errorf("message = %q, want NOT to contain %q", msgStr, notWant)
				}
			}
		})
	}
}

func TestBuildCCList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rest      []string
		fromEmail string
		want      []string
	}{
		{
			name:      "single recipient - adds sender to cc",
			rest:      nil,
			fromEmail: "sender@example.com",
			want:      []string{"sender@example.com"},
		},
		{
			name:      "multiple recipients - adds sender to cc",
			rest:      []string{"b@example.com", "c@example.com"},
			fromEmail: "sender@example.com",
			want:      []string{"b@example.com", "c@example.com", "sender@example.com"},
		},
		{
			name:      "sender already in rest - no duplicate",
			rest:      []string{"b@example.com", "sender@example.com"},
			fromEmail: "sender@example.com",
			want:      []string{"b@example.com", "sender@example.com"},
		},
		{
			name:      "case insensitive dedup",
			rest:      []string{"B@Example.COM"},
			fromEmail: "b@example.com",
			want:      []string{"B@Example.COM"},
		},
		{
			name:      "empty sender email",
			rest:      []string{"b@example.com"},
			fromEmail: "",
			want:      []string{"b@example.com"},
		},
		{
			name:      "empty rest and empty sender",
			rest:      nil,
			fromEmail: "",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := buildCCList(tt.rest, tt.fromEmail)
			if len(got) != len(tt.want) {
				t.Fatalf("buildCCList() len = %d, want %d; got=%v, want=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildCCList()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
