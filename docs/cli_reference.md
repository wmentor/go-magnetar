# CLI Reference

go-magnetar provides a unified CLI interface with the following flags:

## Flags

| Flag | Description |
|---|---|
| `-p, --profile <name>` | Use a specific configuration profile. Overrides the profile specified in the config file. |
| `-f, --file <path>` | Non-interactive mode: read input from file, send to agent, print answer, and exit. Chat commands are not available in this mode. |
| `--session <path>` | Load a previously saved conversation session from a JSON file. |
| `--read-only` | Run in read-only mode. Disables file write operations and shell modifications. |
| `-h, --help` | Show help and exit. |
| `-v, --version` | Show version information and exit. |

## Usage Examples

```bash
# Interactive REPL
go-magnetar

# Use a specific configuration profile
go-magnetar -p production

# Non-interactive mode (scripting / batch)
go-magnetar -f questions.txt

# Load a saved session
go-magnetar --session session.json

# Run in read-only mode
go-magnetar --read-only

# Combined flags
go-magnetar -p production --read-only
```

## Non-Interactive Mode

When using `-f` or `--file`, the program reads the input file, sends it to the agent, prints the answer, and exits. Text preprocessor placeholders like `{{date}}`, `{{uuid}}`, `{{file:path}}` are expanded, but chat commands (e.g., `/index`, `/fetch`) are not available.

## Configuration Profiles

Profiles allow you to maintain multiple configurations (e.g., `default`, `production`, `development`) in a single config file. Use `-p` to select a profile:

```yaml
profiles:
  default:
    llm:
      model: gpt-4o
  production:
    llm:
      model: gpt-4o-2024-08-06
```

```bash
go-magnetar -p production
```

## Sessions

Save conversations to JSON files and restore them later:

```bash
# In REPL
> /session.save /tmp/my-session.json

# Restore later
go-magnetar --session /tmp/my-session.json
```

## Read-Only Mode

When the system is in read-only mode, **NO modification operations are allowed** — all such commands and tool calls are automatically rejected.
