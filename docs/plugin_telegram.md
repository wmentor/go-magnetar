# Telegram Plugin

The Telegram plugin adds support for sending messages to Telegram chats from the agent. It registers the `telegram` tool when enabled via configuration.

## Overview

The plugin provides seamless integration with Telegram by:

- Registering the `telegram` tool with the agent's tool store
- Automatically converting Markdown to HTML for Telegram compatibility
- Enforcing security restrictions (allowed chat IDs)
- Supporting optional features like disabling URL previews and notifications

## Requirements

- Go 1.27.4 or higher
- Telegram bot token from @BotFather
- Valid Telegram chat ID (user, group, or channel)

## Configuration

Enable the Telegram plugin by setting `telegram.enable` to `true` in your configuration file:

```yaml
telegram:
  enable: true              # Enable Telegram integration (default: false)
  bot_token: "YOUR_TOKEN"   # Telegram bot token from @BotFather
  default_chat_id: "123456" # Default chat ID for messages
  allowed_chat_ids:         # Optional: list of allowed chat IDs for security
    - "123456"
    - "789012"
```

### Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `enable` | boolean | Optional | Enable Telegram integration (default: `false`) |
| `bot_token` | string | Required when enabled | Telegram bot token obtained from @BotFather |
| `default_chat_id` | string | Optional | Default chat ID for messages (can be overridden per call) |
| `allowed_chat_ids` | array of strings | Optional | List of allowed chat IDs for security (empty = allow all) |

### Environment Variables

You can use environment variables for sensitive configuration:

```yaml
telegram:
  enable: true
  bot_token: $env:TELEGRAM_BOT_TOKEN
  default_chat_id: $env:TELEGRAM_DEFAULT_CHAT_ID
  allowed_chat_ids:
    - $env:TELEGRAM_CHAT_ID_1
    - $env:TELEGRAM_CHAT_ID_2
```

## Tool Registration

The plugin registers the `telegram` tool only when `telegram.enable` is set to `true`. If disabled, the tool is not available.

## Usage

Once enabled, the `telegram` tool can be invoked by the agent:

```yaml
> Send a notification to Telegram
tool: telegram
params:
  chat_id: "123456"
  text: "# Alert\n\nServer CPU usage exceeded 90%."
```

Or using the default chat ID:

```yaml
> Send completion message
tool: telegram
params:
  text: "Task completed successfully!"
```

## Security

- The plugin only registers the tool when `telegram.enable=true`
- The bot token must be configured before the tool can be used
- Optional `allowed_chat_ids` restricts messages to specific chat IDs
- Bot tokens should be stored securely using environment variables

## Implementation Details

- Tool package: `internal/tools/telegram/`
- Plugin package: `internal/plugins/telegram/`
- Markdown-to-HTML conversion: Uses [Goldmark](https://github.com/yuin/goldmark)
- HTTP client: Uses `RequesterFactory` for proper client pooling
- Timeout: 30 seconds for Telegram API calls

## Testing

Run tests with:

```bash
make test
```

Test file: `internal/tools/telegram/send_test.go`

## See Also

- [tool_telegram.md](./tool_telegram.md) — Detailed tool documentation
- [agent_tools.md](./agent_tools.md) — Complete list of available tools
- [configuration.md](./configuration.md) — Configuration options
