# Slack Tool

The `slack` tool posts a message to a Slack channel from the agent. It sends [Slack's own markdown](https://api.slack.com/reference/surfaces/formatting) (`*bold*`, `_italic_`, `` `code` ``, `<https://example.com|link>`), not HTML.

## Configuration

```yaml
slack:
  enable: true                      # Enable the Slack tool (default: false)
  token: "xoxb-..."                 # Bot token with the chat:write scope
  default_channel_id: "C01234567"   # Used when a call names no channel
  allowed_channel_ids:              # Optional: refuse every other channel
    - "C01234567"
```

Create the app at <https://api.slack.com/apps>, add the `chat:write` scope, install it to the workspace and copy the bot token. Invite the bot to a private channel before posting there; a public channel needs no invite.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channel` | string | Optional | Channel ID (`C01234567`) or name (`#general`). Falls back to `slack.default_channel_id`. |
| `text` | string | Required | Message text in Slack markdown, up to 39,000 characters. |
| `thread_ts` | string | Optional | Reply in the thread whose parent has this `ts`. |
| `unfurl_links` | boolean | Optional | Let Slack expand link previews. |
| `unfurl_media` | boolean | Optional | Let Slack expand media previews. |

## Usage examples

### Post to the default channel

```yaml
> Tell the team the deploy finished
tool: slack
params:
  text: ":white_check_mark: Deploy to production finished in 4m12s."
```

### Reply in a thread

```yaml
> Reply to the incident thread
tool: slack
params:
  channel: "C01234567"
  thread_ts: "1700000000.000100"
  text: "Root cause: the cache key missed the tenant id."
```

## Behaviour and errors

- Slack answers HTTP 200 even when it rejects a message, so the tool reads the `ok` field: `{"ok":false,"error":"channel_not_found"}` becomes `slack: API error: channel_not_found`.
- The token is sent as `Authorization: Bearer …` and never logged or echoed.
- `slack.allowed_channel_ids`, when non-empty, is enforced before any request; a channel outside the list fails with `slack: channel is not in the allowed list`.
- The request has a 30-second timeout and is cancelled when the agent stops.
