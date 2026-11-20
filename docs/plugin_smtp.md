# SMTP Plugin

The SMTP plugin enables email sending capabilities for the agent, allowing it to send HTML emails with Markdown content and file attachments via SMTP servers.

## Enabling the Plugin

Set `smtp.enable: true` in your configuration file (see [configuration.md](./configuration.md) for complete configuration options).

## Configuration

```yaml
smtp:
  enable: false  # enable/disable SMTP sending (default: false)
  host: ""       # SMTP server host (e.g., smtp.gmail.com)
  port: 587      # SMTP server port (25, 465, 587)
  username: ""   # SMTP authentication username
  password: ""   # SMTP authentication password
  encryption: "starttls"  # encryption mode: "tls" (implicit TLS, port 465), "starttls" (port 587), "none" (port 25)
  from_name: ""  # sender display name
  from_email: "" # sender email address
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `smtp.enable` | boolean | Enable/disable SMTP plugin |
| `smtp.host` | string | SMTP server hostname (e.g., `smtp.gmail.com`, `smtp.office365.com`) |
| `smtp.port` | integer | SMTP server port (25, 465, or 587) |
| `smtp.username` | string | SMTP authentication username (usually email address) |
| `smtp.password` | string | SMTP authentication password or app-specific password |
| `smtp.encryption` | string | Encryption mode: `tls` (implicit TLS, port 465), `starttls` (STARTTLS, port 587), `none` (no encryption, port 25) |
| `smtp.from_name` | string | Display name of the sender (e.g., "John Doe") |
| `smtp.from_email` | string | Email address of the sender (e.g., "john@example.com") |

## Tools

When the SMTP plugin is enabled, the following tool becomes available:

| Tool | Signature | Description |
|------|-----------|-------------|
| `smtp` | `(to: string, subject: string, body: string, attachments: []string) -> string` | Sends an email via SMTP with Markdown body (auto-converted to HTML) and optional file attachments |

### Tool Details

#### `smtp`

Sends an email through the configured SMTP server with the following features:

- **Markdown to HTML conversion**: The email body is written in Markdown and automatically converted to HTML
- **Multiple recipients**: Supports comma-separated list of recipients
- **Automatic CC handling**: First recipient goes to "To" field, remaining recipients and sender are added to "Cc" field
- **File attachments**: Supports attaching files from the sandboxed filesystem
- **UTF-8 encoding**: All emails are sent with UTF-8 charset for proper international character support

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `to` | string | Recipient email address(es). Use comma to separate multiple recipients (e.g., "user1@example.com, user2@example.com") |
| `subject` | string | Email subject line |
| `body` | string | Email body in Markdown format. Will be automatically converted to HTML |
| `attachments` | []string | Optional list of file paths to attach to the email. Files are read from the sandboxed filesystem |

**Returns:** Success message with recipient list or error message.

## Encryption Modes

The plugin supports three encryption modes:

### TLS (Implicit TLS) - Port 465

Uses implicit TLS encryption from the start of the connection. This is the most secure option and is recommended for modern email providers.

```yaml
smtp:
  enable: true
  host: smtp.gmail.com
  port: 465
  encryption: tls
```

### STARTTLS - Port 587

Starts with an unencrypted connection and upgrades to TLS using the STARTTLS command. This is the most common configuration for modern email providers.

```yaml
smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  encryption: starttls
```

### None - Port 25

No encryption. Only use this for local development or trusted internal networks.

```yaml
smtp:
  enable: true
  host: localhost
  port: 25
  encryption: none
```

## Usage Examples

### Send Simple Email

```bash
> Send an email to john@example.com with subject "Meeting Reminder" and body "Don't forget about our meeting tomorrow at 3 PM."
[Agent calls smtp with to="john@example.com", subject="Meeting Reminder", body="Don't forget about our meeting tomorrow at 3 PM."]
```

### Send Email with Markdown Content

```bash
> Send an email to team@example.com with subject "Weekly Report" and the following content:
# Weekly Report

## Accomplishments
- Completed feature X
- Fixed bug Y

## Next Week
- Start feature Z
- Code review for PR #123

[Agent calls smtp with to="team@example.com", subject="Weekly Report", body="# Weekly Report\n\n## Accomplishments\n- Completed feature X\n- Fixed bug Y\n\n## Next Week\n- Start feature Z\n- Code review for PR #123"]
```

### Send Email to Multiple Recipients

```bash
> Send an email to alice@example.com, bob@example.com, and charlie@example.com with subject "Project Update" and body "Here's the latest project update..."
[Agent calls smtp with to="alice@example.com, bob@example.com, charlie@example.com", subject="Project Update", body="Here's the latest project update..."]
```

The first recipient (alice@example.com) will be in the "To" field, while bob@example.com, charlie@example.com, and the sender will be in the "Cc" field.

### Send Email with Attachments

```bash
> Send an email to manager@example.com with subject "Report" and body "Please find the report attached.", and attach the file "reports/weekly_report.pdf"
[Agent calls smtp with to="manager@example.com", subject="Report", body="Please find the report attached.", attachments=["reports/weekly_report.pdf"]]
```

### Send Email with Multiple Attachments

```bash
> Send an email to client@example.com with subject "Project Files" and body "Here are the project files you requested.", and attach "docs/spec.pdf", "src/main.go", and "README.md"
[Agent calls smtp with to="client@example.com", subject="Project Files", body="Here are the project files you requested.", attachments=["docs/spec.pdf", "src/main.go", "README.md"]]
```

## Authentication

### Using Environment Variables

For security, it's recommended to use environment variables for credentials:

```bash
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"
```

```yaml
smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  username: $env:SMTP_USERNAME
  password: $env:SMTP_PASSWORD
  encryption: starttls
  from_name: "Your Name"
  from_email: $env:SMTP_USERNAME
```

### Using File-based Secrets

You can also load credentials from files:

```yaml
smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  username: $file:secrets/smtp_username.txt
  password: $file:secrets/smtp_password.txt
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@gmail.com
```

## Common SMTP Server Configurations

### Gmail

```yaml
smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  username: your-email@gmail.com
  password: your-app-password  # Use App Password, not your regular password
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@gmail.com
```

**Note:** For Gmail, you need to:
1. Enable 2-factor authentication
2. Generate an App Password at https://myaccount.google.com/apppasswords
3. Use the App Password in the configuration

### Outlook / Office 365

```yaml
smtp:
  enable: true
  host: smtp.office365.com
  port: 587
  username: your-email@outlook.com
  password: your-password
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@outlook.com
```

### Yahoo Mail

```yaml
smtp:
  enable: true
  host: smtp.mail.yahoo.com
  port: 587
  username: your-email@yahoo.com
  password: your-app-password
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@yahoo.com
```

### Custom SMTP Server

```yaml
smtp:
  enable: true
  host: smtp.yourcompany.com
  port: 587
  username: your-email@yourcompany.com
  password: your-password
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@yourcompany.com
```

## Security

- Credentials can be loaded from environment variables or files for better security
- The plugin uses TLS encryption for secure email transmission (recommended)
- All emails are sent with UTF-8 charset for proper international character support
- File attachments are read from the sandboxed filesystem for security
- The plugin respects the global `readonly` mode if implemented

## Troubleshooting

### Common Errors

**"smtp: host is not configured"**
- Ensure `smtp.host` is set in your configuration

**"smtp: from_email is not configured"**
- Ensure `smtp.from_email` is set in your configuration

**"smtp: failed to send email: dial tcp: connection refused"**
- Check that the SMTP server is running and accessible
- Verify the host and port are correct
- Check firewall settings

**"smtp: failed to send email: authentication failed"**
- Verify username and password are correct
- For Gmail, ensure you're using an App Password, not your regular password
- Check that the account has SMTP access enabled

**"smtp: failed to send email: TLS handshake failed"**
- Verify the encryption mode matches the port:
  - Port 465 → `encryption: tls`
  - Port 587 → `encryption: starttls`
  - Port 25 → `encryption: none`
- Check that the SMTP server supports the selected encryption mode

**"smtp: failed to read attachment: file not found"**
- Verify the file path is correct
- Ensure the file exists in the sandboxed filesystem
- Check file permissions

### Debugging

Enable verbose logging in your configuration:

```yaml
log:
  level: debug
```

This will show detailed SMTP communication logs for troubleshooting.

## Email Deliverability

To ensure your emails don't end up in spam folders:

1. **Use a reputable SMTP provider** (Gmail, Outlook, SendGrid, Mailgun, etc.)
2. **Configure SPF, DKIM, and DMARC** records for your domain (if using a custom domain)
3. **Avoid spam trigger words** in your email content
4. **Use proper HTML formatting** (the plugin handles this automatically)
5. **Include a text/plain alternative** (not currently supported, but planned)

For production use, consider using dedicated email services like SendGrid, Mailgun, or Amazon SES instead of personal email accounts.
