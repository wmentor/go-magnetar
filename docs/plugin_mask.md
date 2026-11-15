# Mask Plugin

The mask plugin enables automatic sensitive data masking for all chat inputs and tool responses to prevent accidental leakage of secrets and personal information.

## Enabling the Plugin

The mask plugin is **enabled by default** and does not require any configuration. You can customize its behavior by setting `mask.enable` in your configuration file (see [configuration.md](./configuration.md) for complete configuration options).

## Configuration

```yaml
mask:
  enable: true          # enable/disable automatic sensitive data masking (default: true)
  skip:
    email: false        # skip email address masking (default: false)
    phone: false        # skip phone number masking (default: false)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `mask.enable` | boolean | Enable/disable automatic sensitive data masking (default: `true`) |
| `mask.skip.email` | boolean | Skip email address masking (default: `false`) |
| `mask.skip.phone` | boolean | Skip phone number masking (default: `false`) |

## How It Works

The mask plugin automatically processes:

1. **System prompts** — The initial system prompt is masked before being sent to the LLM
2. **User inputs** — All user messages are masked before being added to conversation history
3. **Tool responses** — Results from all tools are masked before being returned to the LLM

### What Gets Masked

The mask service automatically detects and replaces the following patterns:

| Pattern | Replacement |
|---------|-------------|
| Email addresses | `[MASKED_EMAIL]` |
| Phone numbers | `[MASKED_PHONE]` |
| GitHub tokens | `[MASKED_TOKEN]` |
| GitLab tokens | `[MASKED_TOKEN]` |
| Russian passport | `[MASKED_PASSPORT]` |
| Russian SNILS | `[MASKED_SNILS]` |
| US Social Security Numbers (SSN) | `[MASK_SSN]` |

## Important Notes

- **Always enabled** — Masking runs automatically for all inputs and outputs; there's no way to disable it per-request
- **Non-intrusive** — Masked data is only replaced in the conversation history; original values are never stored
- **Persists across sessions** — Masked data remains masked even when loading saved conversation sessions
- **Tool-specific** — Each tool receives its own masked context, ensuring proper isolation

## Security

For complete information about the security implications and additional security features, see [docs/security.md](./security.md).
