package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-gomail/gomail"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// Params represents parameters for the smtp tool.
type Params struct {
	To          string   `json:"to" jsonschema_description:"Recipient email address. Use comma to separate multiple recipients."`
	Subject     string   `json:"subject" jsonschema_description:"Email subject line."`
	Body        string   `json:"body" jsonschema_description:"Email body in Markdown format. Will be automatically converted to HTML."`
	Attachments []string `json:"attachments,omitempty" jsonschema_description:"List of file paths to attach to the email."`
}

// Tool provides smtp send operation as LLM tool.
type Tool struct {
	state *plugin.State
}

// New creates a new Tool instance.
func New(state *plugin.State) *Tool {
	return &Tool{
		state: state,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "smtp"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Send an email via SMTP. Accepts the body in Markdown format, which is automatically converted to HTML. " +
		"Requires SMTP configuration (host, port, credentials, encryption) to be set in the config."
}

// Exec executes the smtp tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *Params) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.state.Printer.ToolCall(printer.IconMail, "smtp", "to", params.To, "subject", params.Subject)

	if params.To == "" {
		return nil, errors.New("smtp: recipient address is required")
	}

	if params.Subject == "" {
		return nil, errors.New("smtp: subject is required")
	}

	if params.Body == "" {
		return nil, errors.New("smtp: body is required")
	}

	// Read SMTP configuration.
	host := t.state.Config.String("smtp.host")
	port := t.state.Config.Int("smtp.port")
	username := t.state.Config.String("smtp.username")
	password := t.state.Config.String("smtp.password")
	encryption := t.state.Config.String("smtp.encryption")
	fromName := t.state.Config.String("smtp.from_name")
	fromEmail := t.state.Config.String("smtp.from_email")

	if host == "" {
		return nil, errors.New("smtp: host is not configured")
	}

	if fromEmail == "" {
		return nil, errors.New("smtp: from_email is not configured")
	}

	// Convert markdown body to HTML.
	htmlBody, err := convertMarkdownToHTML(params.Body)
	if err != nil {
		t.state.Printer.Error("smtp: failed to convert markdown to HTML", "err", err)
		return nil, fmt.Errorf("smtp: failed to convert markdown to HTML: %w", err)
	}

	// Parse recipients and split into To (first) and Cc (rest + sender).
	recipients := parseRecipients(params.To)
	toHeader := recipients[0]
	ccList := buildCCList(recipients[1:], fromEmail)

	// Create email message using gomail.
	msg := gomail.NewMessage()
	if fromName != "" {
		msg.SetAddressHeader("From", fromEmail, fromName)
	} else {
		msg.SetHeader("From", fromEmail)
	}
	msg.SetHeader("To", toHeader)
	if len(ccList) > 0 {
		msg.SetHeader("Cc", ccList...)
	}
	msg.SetHeader("Subject", params.Subject)
	msg.SetBody("text/html; charset=UTF-8", htmlBody)

	// Attach files if any.
	if len(params.Attachments) > 0 {
		if t.state.Root == nil {
			return nil, errors.New("smtp: root filesystem is not set")
		}
		for _, path := range params.Attachments {
			data, err := t.state.Root.ReadFile(path)
			if err != nil {
				t.state.Printer.Error("smtp: failed to read attachment", "file", path, "err", err)
				return nil, fmt.Errorf("smtp: failed to read attachment %q: %w", path, err)
			}
			filename := filepath.Base(path)
			msg.Attach(filename, gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(data)
				return err
			}))
		}
	}

	// Send the email using gomail dialer.
	if err := sendMail(host, port, encryption, username, password, msg); err != nil {
		t.state.Printer.Error("smtp: failed to send email", "to", params.To, "err", err)
		return nil, fmt.Errorf("smtp: failed to send email: %w", err)
	}

	t.state.Printer.ToolCall(printer.IconDone, "smtp: email sent successfully", "to", params.To)

	return &tools.Response{
		Status: tools.StatusOK,
		Result: fmt.Sprintf("Email sent successfully to %s.", params.To),
	}, nil
}

// convertMarkdownToHTML converts markdown content to HTML using goldmark with GFM support.
func convertMarkdownToHTML(markdownContent string) (string, error) {
	var buf strings.Builder
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	if err := markdown.Convert([]byte(markdownContent), &buf); err != nil {
		return "", fmt.Errorf("convert markdown to HTML: %w", err)
	}
	return buf.String(), nil
}

// parseRecipients splits a comma-separated list of email addresses.
func parseRecipients(to string) []string {
	parts := strings.Split(to, ",")
	recipients := make([]string, 0, len(parts))
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr != "" {
			recipients = append(recipients, addr)
		}
	}
	return recipients
}

// buildCCList builds the CC recipient list from the remaining recipients and the sender email.
// It deduplicates addresses (case-insensitive) and preserves order.
func buildCCList(rest []string, fromEmail string) []string {
	seen := make(map[string]bool, len(rest)+1)
	cc := make([]string, 0, len(rest)+1)

	for _, addr := range rest {
		key := strings.ToLower(addr)
		if !seen[key] {
			seen[key] = true
			cc = append(cc, addr)
		}
	}

	fromKey := strings.ToLower(fromEmail)
	if fromEmail != "" && !seen[fromKey] {
		cc = append(cc, fromEmail)
	}

	return cc
}

// sendMail sends an email using gomail with the specified encryption mode.
func sendMail(host string, port int, encryption, username, password string, msg *gomail.Message) error {
	d := gomail.NewDialer(host, port, username, password)

	// Configure TLS based on encryption mode.
	switch strings.ToLower(encryption) {
	case "tls":
		d.SSL = true
		d.TLSConfig = &tls.Config{
			ServerName: host,
		}
	case "starttls":
		d.SSL = false
		d.TLSConfig = &tls.Config{
			ServerName: host,
		}
	case "none", "":
		d.SSL = false
		d.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	default:
		return fmt.Errorf("unsupported encryption mode: %s", encryption)
	}

	// Create a sender with context support.
	var sendCloser gomail.SendCloser
	var err error

	// Use gomail's Send function with context.
	sendCloser, err = d.Dial()
	if err != nil {
		return fmt.Errorf("SMTP dial: %w", err)
	}
	defer sendCloser.Close()

	if err := gomail.Send(sendCloser, msg); err != nil {
		return fmt.Errorf("SMTP send: %w", err)
	}

	return nil
}
