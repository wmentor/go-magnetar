# Telegram Tool

The `telegram` tool enables sending messages to Telegram chats from the agent. It accepts Markdown-formatted text and automatically converts it to HTML before sending to Telegram.

## Configuration

The telegram tool requires the following configuration in your config file:

```yaml
telegram:
  enable: true          # Enable telegram integration (default: false)
  bot_token: "YOUR_BOT_TOKEN"  # Telegram bot token from @BotFather
  default_chat_id: "YOUR_CHAT_ID"  # Default chat ID for messages
  allowed_chat_ids:     # Optional: list of allowed chat IDs for security
    - "123456"
    - "789012"
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `chat_id` | string | Optional | Telegram chat ID where the message will be sent. If not provided, uses `default_chat_id` from config. |
| `text` | string | Required | Message text in Markdown format. Will be automatically converted to HTML. |
| `disable_preview` | boolean | Optional | Disable URL preview for links in the message. Default: `false`. |

## Usage Examples

### Send a message with explicit chat ID

```yaml
> Send a message to Telegram
tool: telegram
params:
  chat_id: "123456"
  text: "# Hello World\n\nThis is a **test** message from the agent."
```

### Send a message using default chat ID

```yaml
> Send a notification
tool: telegram
params:
  text: "Agent completed the task successfully!"
```

### Send a formatted message with code blocks

```yaml
> Send command output
tool: telegram
params:
  text: "```bash\n./bin/go-magnetar --version\n```\n\nVersion: 1.0.0"
  disable_preview: true
```

## Markdown to HTML Conversion

The tool uses [Goldmark](https://github.com/yuin/goldmark) to convert Markdown to HTML. Supported Markdown features include:

- Headings: `# Heading`, `## Subheading`, etc.
- Bold text: `**bold**` or `__bold__`
- Italic text: `*italic*` or `_italic_`
- Code blocks: triple backticks with optional language
- Inline code: single backticks
- Links: `[text](url)`
- Images: `![alt](url)`
- Lists: ordered and unordered
- Blockquotes: `> quote`

## Security

- The `telegram.enable` flag must be set to `true` for the tool to function
- The `telegram.bot_token` must be configured with a valid bot token from @BotFather
- Optionally, `telegram.allowed_chat_ids` can restrict messages to specific chat IDs
- Bot token should be stored securely, e.g., using environment variables: `$env:TELEGRAM_BOT_TOKEN`

## API

The tool sends messages to Telegram API via:
```
POST https://api.telegram.org/bot{token}/sendMessage
```

## Response

On success, the tool returns:
```json
{
  "status": "OK",
  "result": "Message sent successfully to chat {chat_id}"
}
```

On error, returns appropriate error message.
