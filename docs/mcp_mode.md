# MCP Mode

go-magnetar can be launched in MCP (Model Context Protocol) server mode, which allows it to act as an integrated knowledge base for other AI agents and applications that support the MCP protocol.

## MCP Server Modes

go-magnetar supports two transport modes for MCP servers:

- **stdio** — Communicates via standard input/output (default mode, suitable for local clients)
- **http** — Communicates via HTTP endpoint (suitable for remote clients and web applications)

## Launching go-magnetar in MCP Mode

To start go-magnetar in MCP mode, use the `--mcp` flag:

```bash
./bin/go-magnetar --mcp
```

The MCP server will start with the default configuration from `configs/plugins/mcp.yml` and listen for connections from MCP-compatible clients.

## Configuration

MCP server behavior is controlled by the `mcp` section in `configs/plugins/mcp.yml`:

```yaml
mcp:
  mode: stdio  # mcp mode: stdio | http (default: stdio)
  server:
    host: localhost  # host for HTTP server (default: localhost)
    port: 8080  # port for HTTP server (default: 8080)
    cert: ""  # path to TLS certificate (empty for HTTP, required for HTTPS)
    key: ""  # path to TLS private key (empty for HTTP, required for HTTPS)
    tls: false  # enable TLS encryption (default: false)
    endpoint: "/mcp"  # HTTP endpoint path (default: /mcp)
```

### Stdio Mode (Default)

Stdio mode is the default and requires no additional configuration:

```yaml
mcp:
  mode: stdio  # default
```

This mode is suitable for local clients that can connect via stdin/stdout, such as Claude Desktop and other CLI tools.

### HTTP Mode

HTTP mode allows remote clients to connect via HTTP:

```yaml
mcp:
  mode: http
  server:
    host: localhost
    port: 8080
    endpoint: "/mcp"
```

For HTTPS, provide certificate and key paths:

```yaml
mcp:
  mode: http
  server:
    host: localhost
    port: 8080
    cert: "/path/to/certificate.pem"
    key: "/path/to/private.key"
    tls: true
    endpoint: "/mcp"
```

## Important: Single Instance Constraint with Local RAG Storage

When using RAG with the **local Chromem storage** (default configuration in `configs/config.yml`), **only one instance** of go-magnetar can be launched at a time.

### Why?

The Chromem vector store uses an in-memory database that is **not shared across multiple processes**. When multiple instances try to access the same RAG database simultaneously:

- Each instance maintains its own isolated in-memory index
- Updates from one instance are not visible to others
- This can lead to inconsistent search results and data synchronization issues

### Configuration

By default, Chromem is configured as the RAG storage backend:

```yaml
rag:
  store:
    type: chromem  # Default
```

### Solutions

If you need to run multiple instances or concurrent access to the RAG knowledge base:

1. **Use Qdrant as external storage** (recommended for multi-instance setups):

```yaml
rag:
  store:
    type: qdrant
    qdrant:
      host: localhost
      port: 6333
      api_key: ${env:QDRANT_API_KEY}  # Optional, if authentication is enabled
```

   Qdrant is an external vector database that supports concurrent access from multiple processes and machines.

2. **Run only one instance** when using Chromem local storage

### Checking Active Instances

Before starting go-magnetar in MCP mode with Chromem storage, ensure no other instances are running:

```bash
# Check for existing go-magnetar processes
ps aux | grep go-magnetar

# Kill existing instances (if needed)
pkill -f go-magnetar
```

### Best Practices

1. **For development/testing**: Use Chromem local storage with a single instance
2. **For production/multi-user**: Use Qdrant external storage for concurrent access
3. **Keep knowledge base updated**: Use the `/index` command or `/idxtab` command to add new documents
4. **Monitor memory usage**: Chromem stores all vectors in memory; monitor for large knowledge bases

## Example: Starting MCP Server

### Stdio Mode (Default)

```bash
# Single instance with local Chromem storage (default config)
./bin/go-magnetar --mcp
```

### HTTP Mode

```bash
# HTTP server on localhost:8080
./bin/go-magnetar --mcp
```

With custom port:

```bash
# Edit configs/plugins/mcp.yml to change port
mcp:
  mode: http
  server:
    port: 9000
```

### HTTPS Mode

```bash
# HTTPS server with TLS
# First generate certificates (e.g., using mkcert)
# Then configure:
mcp:
  mode: http
  server:
    host: localhost
    port: 8080
    cert: "/path/to/cert.pem"
    key: "/path/to/key.pem"
    tls: true
```

Once running, MCP clients can connect to the server and use the available tools for RAG-based searches, web searches, vulnerability lookups, and more.

## Connecting MCP Clients

### OpenCode

To connect go-magnetar as an MCP server in OpenCode, add the following to your OpenCode configuration:

**Stdio Mode:**
```json
{
  "mcp": {
    "magnetar": {
      "type": "local",
      "command": [
        "/Users/mvk/.local/bin/go-magnetar", "--mcp"
      ],
      "cwd": ".",
      "enabled": true
    }
  }
}
```

**HTTP Mode:**
```json
{
  "mcp": {
    "magnetar": {
      "type": "remote",
      "url": "http://localhost:8080/mcp",
      "enabled": true
    }
  }
}
```

Replace the `command` path with the actual path to the go-magnetar binary if it differs.

### Claude Desktop

For Claude Desktop, add the following to your `claude_desktop_config.json`:

**Stdio Mode (Default):**
```json
{
  "mcpServers": [
    {
      "name": "magnetar",
      "command": "/Users/mvk/.local/bin/go-magnetar",
      "args": ["--mcp"]
    }
  ]
}
```

**HTTP Mode:**
```json
{
  "mcpServers": [
    {
      "name": "magnetar",
      "type": "http",
      "url": "http://localhost:8080/mcp"
    }
  ]
}
```

On macOS, the config file is typically located at:
- `~/Library/Application Support/Claude/claude_desktop_config.json`

Replace the `command` path with the actual path to the go-magnetar binary if it differs.

### CLI (Direct Use)

You can also connect to the MCP server directly using an MCP client:

```bash
# Start go-magnetar in MCP mode (in one terminal)
./bin/go-magnetar --mcp

# Connect from another terminal or application using MCP client
# The server will be available at the configured transport (stdio or HTTP endpoint)
```

## Available Tools

Once connected, the MCP client can use the following tools from go-magnetar:

- `web_fetch` — Fetch content from URLs (Confluence, JIRA, GitHub, GitLab, etc.)
- `file_read` — Read files from the filesystem with support for gzip and various formats
- `rag_search` — Search the indexed RAG knowledge base
- `cve_info` — Lookup vulnerability information from OSV database (CVE, GO, GHSA, OSV, and more)
- `index` — Index files or URLs into the RAG knowledge base

For detailed documentation on each tool, see [agent_tools.md](./agent_tools.md).

### Tools Not Available via MCP

The MCP server only exports tools specifically registered for MCP. Other tools are available only through the chat REPL interface:

- `web_search` — Search the web (available via `/web.search` chat command)
- `file_write` — Write files to the filesystem (available via `/file.write` chat command)
- `file_list` — List directory contents (available via `/file.list` chat command)
- `edit` — Edit files (available via `/file.edit` chat command)

See [docs/chat_command.md](./chat_command.md) for a complete list of chat commands.
