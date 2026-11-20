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
| `--config-dir <path>` | Set the configuration directory. Default: `~/.go-magnetar`. Can also be set via `MAGNETAR_CONFIG_DIR` environment variable. |
| `<prompt>` | Direct prompt argument(s). When provided, runs in non-interactive mode and executes the prompt(s) as input. Multiple arguments are joined with newlines. |
| `-d, --daemon` | Run as a background daemon. Requires `--mcp` with `mcp.mode: http` configuration. Starts go-magnetar in detached mode, suitable for long-running server processes. |
| `--mcp` | Launch in MCP (Model Context Protocol) server mode. For detailed configuration options (stdio/HTTP modes) and usage examples, see [docs/mcp_mode.md](./mcp_mode.md). |
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
cat questions.txt | go-magnetar -f -

# Load a saved session
go-magnetar --session session.json

# Run in read-only mode
go-magnetar --read-only

# Enable execution time display
go-magnetar --enable-timing

# Launch in MCP server mode
go-magnetar --mcp

# Run as a background daemon (requires mcp.mode: http in config)
go-magnetar --daemon --mcp

# Direct prompt in daemon mode (requires mcp.mode: http in config)
go-magnetar --daemon --mcp "What is go-magnetar?"

# Combined flags
go-magnetar -p production --read-only

# Use custom config directory
go-magnetar --config-dir /custom/path

# Set config directory via environment variable
export MAGNETAR_CONFIG_DIR=/custom/path
go-magnetar

# Direct prompt arguments
go-magnetar "What is go-magnetar?"

# Multiple prompt arguments
go-magnetar "Explain the following concept:" "How does RAG work?"

# Combine with other flags
go-magnetar -p production "What is the default model?"
```

## Direct Prompt Arguments

You can pass prompts directly as command-line arguments without using files or stdin. This is useful for quick one-liners or scripting:

```bash
# Single prompt
go-magnetar "What is go-magnetar?"

# Multiple prompts (joined with newlines)
go-magnetar "Explain the following concept:" "How does RAG work?"

# Combine with flags
go-magnetar -p production "What is the default model?"
```

Multiple arguments are joined with newlines and processed as a single input. This mode is non-interactive and behaves similarly to `-f` mode.

## Non-Interactive Mode

When using `-f` or `--file`, the program reads the input file (or stdin if `-`), sends it to the agent, prints the answer, and exits. Text preprocessor placeholders like `{{date}}`, `{{uuid}}`, `{{file:path}}` are expanded, but chat commands (e.g., `/index`, `/fetch`) are not available.

## Reading from stdin

Pipe commands to go-magnetar for processing:

```bash
echo "What is go-magnetar?" | go-magnetar
cat questions.txt | go-magnetar

# Combine stdin with a prompt (stdin comes before prompt)
ls -la | go-magnetar "Analyze this directory listing:"
```

**Note:** When using pipe mode, the content from stdin is combined with any direct prompt arguments. The stdin content appears first, followed by the prompt(s).

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

## Configuration Directory

The configuration directory is where go-magnetar stores its configuration files, skills, and agents. By default, it uses `~/.go-magnetar`, but you can customize it with the `--config-dir` flag or the `MAGNETAR_CONFIG_DIR` environment variable.

```bash
# Use custom config directory
go-magnetar --config-dir /custom/path

# Set via environment variable
export MAGNETAR_CONFIG_DIR=/custom/path
go-magnetar
```

All paths for config files, skills, and agents are relative to the configured directory.

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
- **User skills**: `<config-dir>/skills/` — global skills
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

# Combine with custom config directory
go-magnetar --config-dir /custom/path --skill dev-docs
```

## MCP Server Mode

When launching with the `--mcp` flag, go-magnetar starts a Model Context Protocol (MCP) server. For detailed information about MCP modes (stdio/HTTP), configuration, and client integration, see [docs/mcp_mode.md](./mcp_mode.md).
