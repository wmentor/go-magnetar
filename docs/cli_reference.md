# CLI Reference

go-magnetar provides a unified CLI interface with the following flags:

## Flags

| Flag | Description |
|---|---|
| `-p, --profile <name>` | Use a specific configuration profile. Overrides the profile specified in the config file. |
| `--skill <name>` | Apply a skill at startup. Loads and activates the specified skill before processing input. |
| `-f, --file <path>` | Non-interactive mode: read input from file (or stdin if `-`), send to agent, print answer, and exit. Chat commands are not available in this mode. |
| `--session <path>` | Load a previously saved conversation session from a JSON file. |
| `--read-only` | Run in read-only mode. Disables file write operations and shell modifications. |
| `--enable-timing` | Start with the execution time display on. `/timing on\|off` can still change it during the session. |
| `-h, --help` | Show help and exit. |
| `-v, --version` | Show version information and exit. |

## Usage Examples

```bash
# Interactive REPL
go-magnetar

# Use a specific configuration profile
go-magnetar -p production

# Apply a skill at startup
go-magnetar --skill dev-docs

# Non-interactive mode (scripting / batch)
go-magnetar -f questions.txt

# Apply skill and process file
go-magnetar --skill dev-docs -f questions.txt

# Read from stdin
echo "What is go-magnetar?" | go-magnetar -f -

# Load a saved session
go-magnetar --session session.json

# Run in read-only mode
go-magnetar --read-only

# Enable execution time display
go-magnetar --enable-timing

# Combined flags
go-magnetar -p production --read-only
```

## Non-Interactive Mode

When using `-f` or `--file`, the program reads the input file (or stdin if `-`), sends it to the agent, prints the answer, and exits. Text preprocessor placeholders like `{{date}}`, `{{uuid}}`, `{{file:path}}` are expanded, but chat commands (e.g., `/index`, `/fetch`) are not available.

## Reading from stdin

To read input from stdin, use `-` as the file path:

```bash
echo "What is go-magnetar?" | go-magnetar -f -
cat questions.txt | go-magnetar -f -
```

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

## Skills

Skills allow you to customize the agent's behavior by applying specialized instructions at startup. Skills are loaded from two locations:
- **User skills**: `~/.go-magnetar/skills/` — global skills
- **Project skills**: `{current_dir}/.go-magnetar/skills/` — project-specific skills (takes priority)

### Apply Skill at Startup

Use the `--skill` flag to apply a skill immediately when starting the agent:

```bash
# Apply skill in interactive mode
go-magnetar --skill dev-docs

# Apply skill in non-interactive mode
go-magnetar --skill dev-docs -f input.txt

# Combine with profile selection
go-magnetar -p production --skill dev-docs -f input.txt
```

The skill is applied before any other processing and becomes part of the agent's context for the session.
