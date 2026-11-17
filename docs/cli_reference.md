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
| `--mcp` | Launch in MCP (Model Context Protocol) server mode. Starts a console-based MCP server that exposes the following tools to compatible clients: `web_fetch`, `file_read`, `rag_search`, `cve_info`. |
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

# Combined flags
go-magnetar -p production --read-only

# Use custom config directory
go-magnetar --config-dir /custom/path

# Set config directory via environment variable
export MAGNETAR_CONFIG_DIR=/custom/path
go-magnetar
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

When launching with the `--mcp` flag, go-magnetar starts a console-based Model Context Protocol (MCP) server that exposes the following tools to compatible MCP clients:

- `web_fetch` — Fetch content from URLs (Confluence, JIRA, GitHub, GitLab, etc.)
- `file_read` — Read files from the filesystem with support for gzip and various formats
- `rag_search` — Search the indexed RAG knowledge base
- `cve_info` — Lookup vulnerability information from OSV database (CVE, GO, GHSA, OSV, and more)

### Starting MCP Server

```bash
# Launch MCP server
go-magnetar --mcp
```

The server will:
- Start on stdin/stdout transport
- Advertise the following tools: `web_fetch`, `file_read`, `rag_search`, `cve_info`
- Wait for MCP client connections

### MCP Client Integration

Any MCP-compatible client can connect to the server and use the available tools. The server implements the [MCP specification](https://modelcontextprotocol.io/) and exposes the following tools registered in go-magnetar:

- `web_fetch` — Fetch content from URLs (Confluence, JIRA, GitHub, GitLab, etc.)
- `file_read` — Read files from the filesystem with support for gzip and various formats
- `rag_search` — Search the indexed RAG knowledge base
- `cve_info` — Lookup vulnerability information from OSV database (CVE, GO, GHSA, OSV, and more)

For example, an MCP client can request the server to:
- Read files with `file_read`
- Search the web with `web_fetch`
- Query the knowledge base with `rag_search`
- Lookup vulnerabilities with `cve_info`

See [docs/agent_tools.md](./agent_tools.md) for a complete list of available tools.

**Note:** Not all tools registered in go-magnetar are exposed via MCP. The MCP server only exports tools specifically registered for MCP via `RegisterOurToolMCP`. Other tools (like `web_search`, `file_write`, `file_list`, `edit`) are available only through the chat REPL interface.

### Example: Using with Claude Desktop

To use go-magnetar as an MCP server with Claude Desktop:

1. Start the MCP server:
   ```bash
   go-magnetar --mcp
   ```

2. Configure Claude Desktop to connect to the server (follow Claude Desktop's MCP configuration instructions)

The server will handle MCP protocol messages on stdin/stdout and respond to tool invocation requests from the client.

### Notes

- When running in MCP mode, all normal chat REPL functionality is disabled
- Output is directed to stderr to avoid interfering with MCP protocol communication on stdout
- The server runs until manually terminated (Ctrl+C)
