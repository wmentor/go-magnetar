# Slack Plugin

The Slack plugin enables go-magnetar to send messages to Slack channels using the `slack` tool. This plugin is disabled by default and must be explicitly enabled in the configuration.

## Overview

The Slack plugin registers the `slack` tool with the agent, allowing the LLM to send messages to Slack channels using Slack's own markdown formatting. Messages support bold text, italics, code blocks, links, and other Slack markdown features.

## Configuration

The plugin is configured in the main configuration file (`config.yml`) or via environment variables.

### Required Configuration

| Key | Type | Description |
|---|---|---|
| `slack.enable` | boolean | Enable the Slack plugin (default: `false`) |
| `slack.token` | string | Slack bot token with `chat:write` scope |

### Optional Configuration

| Key | Type | Description |
|---|---|---|
| `slack.default_channel_id` | string | Default channel ID or name to use when no channel is specified |
| `slack.allowed_channel_ids` | array of strings | List of allowed channel IDs/names; if set, only these channels can receive messages |

### Example Configuration

```yaml
slack:
  enable: true
  token: "xoxb-..."
  default_channel_id: "C01234567"
  allowed_channel_ids:
    - "C01234567"  # #general
    - "C02345678"  # #announcements
```

### Environment Variables

You can also configure Slack using environment variables:

```bash
export MAGNETAR_SLACK_ENABLE=true
export MAGNETAR_SLACK_TOKEN="xoxb-..."
export MAGNETAR_SLACK_DEFAULT_CHANNEL_ID="C01234567"
export MAGNETAR_SLACK_ALLOWED_CHANNEL_IDS='["C01234567","C02345678"]'
```

Or reference environment variables in the config file:

```yaml
slack:
  enable: $env:MAGNETAR_SLACK_ENABLE
  token: $env:MAGNETAR_SLACK_TOKEN
  default_channel_id: $env:MAGNETAR_SLACK_DEFAULT_CHANNEL_ID
```

## Setup Instructions

### 1. Create a Slack App

1. Go to [https://api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App"
3. Choose "From scratch"
4. Enter a name for your app and select the workspace

### 2. Add the `chat:write` Scope

1. In your app configuration, navigate to "OAuth & Permissions"
2. Under "Scopes", find "Bot Token Scopes"
3. Add the `chat:write` scope (required for sending messages)

### 3. Install the App

1. Click "Install to Workspace" (or "Reinstall to Workspace" if updating)
2. Authorize the app to install to your workspace

### 4. Copy the Bot Token

1. After installation, copy the "Bot User OAuth Token" (starts with `xoxb-`)
2. This is your `slack.token` value

### 5. Invite the Bot to Channels

- **Public channels**: No invitation needed; just use the channel name (e.g., `#general`)
- **Private channels**: Invite the bot using `/invite @YourBotName` in the channel

## Usage

Once enabled, the `slack` tool becomes available to the agent. The tool accepts the following parameters:

| Parameter | Type | Required | Description |
|---|---|---|
| `channel` | string | Optional | Channel ID (e.g., `C01234567`) or name (e.g., `#general`). Falls back to `slack.default_channel_id` if not provided |
| `text` | string | Required | Message text in Slack markdown format |
| `thread_ts` | string | Optional | Thread timestamp to reply in an existing thread |
| `unfurl_links` | boolean | Optional | Let Slack expand link previews |
| `unfurl_media` | boolean | Optional | Let Slack expand media previews |

### Example: Send a Message to Default Channel

```yaml
> Tell the team the deploy finished
tool: slack
params:
  text: ":white_check_mark: Deploy to production finished in 4m12s."
```

### Example: Send to Specific Channel

```yaml
> Post an update to the announcements channel
tool: slack
params:
  channel: "C02345678"
  text: "New feature release coming soon!"
```

### Example: Reply in a Thread

```yaml
> Reply to the incident thread
tool: slack
params:
  channel: "C01234567"
  thread_ts: "1700000000.000100"
  text: "Root cause: the cache key missed the tenant id."
```

### Example: Send with Link Previews Disabled

```yaml
> Send a message without link previews
tool: slack
params:
  channel: "C01234567"
  text: "Check out https://example.com"
  unfurl_links: false
```

## Slack Markdown Formatting

The `slack` tool uses [Slack's markdown format](https://api.slack.com/reference/surfaces/formatting):

| Syntax | Result |
|---|---|
| `*bold text*` | **bold text** |
| `_italic text_` | *italic text* |
| `~strikethrough~` | ~~strikethrough~~ |
| `` `code` `` | `code` |
| `*bold* _italic_ \`code\`` | **bold** *italic* `code` |
| `<https://example.com|link text>` | [link text](https://example.com) |
| `:emoji_name:` | :emoji_name: (Slack emoji) |

## Security Considerations

- **Token Security**: The Slack bot token provides full access to send messages in authorized channels. Never commit it to version control.
- **Channel Restrictions**: Use `slack.allowed_channel_ids` to restrict which channels the bot can message, preventing accidental messages to sensitive channels.
- **Message Length**: Slack limits messages to 40,000 characters. The plugin enforces a limit of 39,000 characters to stay safely under the limit.

## Error Handling

The plugin handles various error conditions:

| Error | Description |
|---|---|
| `slack: integration is disabled` | `slack.enable` is not set to `true` |
| `slack: token is not configured` | `slack.token` is missing |
| `slack: channel is not provided and default_channel_id is not configured` | No channel specified and no default configured |
| `slack: channel is not in the allowed list` | Channel not in `slack.allowed_channel_ids` (when configured) |
| `slack: text is required` | Empty message text |
| `slack: text is longer than 39000 characters` | Message exceeds length limit |
| `slack: API error: channel_not_found` | Slack API returned an error (e.g., invalid channel) |
| `slack: failed to send message` | Network or HTTP error |

## Plugin Files

The Slack plugin consists of:

- `internal/tools/slack/send.go` — Tool implementation
- `internal/tools/slack/send_test.go` — Tool unit tests
- `internal/plugins/slack/plugin.go` — Plugin registration
- `internal/plugins/slack/plugin_test.go` — Plugin integration tests

## Related Documentation

- [tool_slack.md](./tool_slack.md) — Detailed tool documentation
- [configuration.md](./configuration.md) — Configuration reference
- [security.md](./security.md) — Security best practices
