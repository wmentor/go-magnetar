# Confluence Plugin

The Confluence plugin enables integration with Atlassian Confluence for reading and editing pages.

## Enabling the Plugin

Set `confluence.enable: true` in your configuration file (see [configuration.md](./configuration.md) for complete configuration options).

## Configuration

```yaml
confluence:
  enable: false  # enable/disable Confluence fetching (default: false)
  readonly: false # read-only mode (default: false)
  base_url: https://your-domain.atlassian.net  # Confluence base URL
  api_key: ""  # Confluence API key
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `confluence.enable` | boolean | Enable/disable Confluence plugin |
| `confluence.readonly` | boolean | If `true`, disables page editing (default: `false`) |
| `confluence.base_url` | string | Confluence base URL (e.g., `https://your-domain.atlassian.net`) |
| `confluence.api_key` | string | Confluence API key (required for edit operations) |

## Tools

When the Confluence plugin is enabled, the following tool becomes available:

| Tool | Signature | Description |
|------|-----------|-------------|
| `confluence_page_edit` | `(page_id: string, markdown: string) -> string` | Edits a Confluence page by converting Markdown to HTML and updating via REST API |

### Read-Only Mode

When `confluence.readonly: true` is set, the `confluence_page_edit` tool is **not registered**, preventing any modifications to Confluence pages.

## Integration with web_fetch

When the Confluence plugin is enabled, `web_fetch` **transparently** retrieves content from Confluence pages via the Confluence REST API.

This means:
- You can fetch Confluence pages using the standard `web_fetch` tool
- The tool automatically detects Confluence URLs (including short links like `/x/` and share links like `/p/`)
- No additional configuration or special commands are required
- The fetched content is returned in Markdown format, consistent with other web sources

### Supported URL Formats

- Standard pages: `https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456`
- Short links: `https://your-domain.atlassian.net/x/abc123`
- Share links: `https://your-domain.atlassian.net/p/abc123`

## Chat Command

The plugin also provides a dedicated chat command:

| Command | Aliases | Description |
|---------|---------|-------------|
| `/confluence.edit <url> <markdown_file>` | | Edit a Confluence page from a Markdown file |

This command extracts the page ID from the URL (including short links) and uploads the content from the specified Markdown file.

## Security

The plugin respects the global `readonly` mode. If `state.ReadOnly` is `true`, all edit operations are blocked even if the API key is provided.
