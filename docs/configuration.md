# Configuration

go-magnetar uses YAML configuration files to control all aspects of the application, from LLM settings to plugin configurations.

## Configuration File Location

By default, go-magnetar looks for the configuration file at `~/.go-magnetar/config.yml`. You can specify a custom configuration directory using the `--config-dir` flag or the `MAGNETAR_CONFIG_DIR` environment variable.

### Using --config-dir flag

```
go-magnetar --config-dir <path>
```

Set the configuration directory where config files and user skills/agents are stored. Default is `~/.go-magnetar`.

### Using MAGNETAR_CONFIG_DIR environment variable

```
export MAGNETAR_CONFIG_DIR=<path>
go-magnetar
```

Environment variable takes precedence over the default directory.

Example:
```bash
# Set custom config directory
export MAGNETAR_CONFIG_DIR=~/.custom-config
go-magnetar
```

## Profile-based Configuration

Configuration supports multiple profiles, allowing you to switch between different setups (e.g., development, production) without modifying the main config:

```bash
go-magnetar -p production
```

## Configuration Structure

### Top-Level Keys

| Key | Type | Description |
|---|---|---|
| `version` | string | Configuration file format version |
| `language` | string | Language for agent responses (default: `english`) |
| `verbose` | boolean | Enable verbose tool call output (default: `true`) |
| `profile` | string | Default profile name to use |
| `include` | array | List of plugin configuration files to include |
| `profiles` | object | Profile definitions for multi-environment support |
| `mcp` | object | MCP server configuration (stdio or HTTP mode) |

### LLM Configuration

The `llm` block configures the primary language model for chat responses:

| Key | Type | Description |
|---|---|---|
| `base_url` | string | OpenAI-compatible endpoint URL (e.g., `https://api.openai.com/v1`) |
| `api_key` | string | API key for the LLM service |
| `model` | string | Model name (e.g., `gpt-4o`) |
| `context` | integer | Context window size in tokens (default: `128000`) |
| `temperature` | float | Temperature for response generation (default: `0.9`) |
| `top_p` | float | Top-p sampling value (default: `0.95`) |
| `reasoning_effort` | string | Reasoning effort level: `low`, `medium`, or `high` (default: `high`) |

### RAG Configuration

The `rag` block controls retrieval-augmented generation settings:

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable RAG functionality (default: `false`) |
| `store.type` | string | Vector store type: `qdrant` or `chromem` (default: `chromem`, built-in) |
| `llm` | object | Embedding model configuration (same structure as `llm` block) |
| `chunk` | object | Document chunking parameters |
| `search` | object | Search parameters |
| `qdrant` | object | Qdrant vector database connection |
| `chromem` | object | Chromem vector store settings (used when `store.type: chromem`) |

#### RAG Chunk Parameters

| Key | Type | Description |
|---|---|---|
| `size` | integer | Maximum chunk size in runes (default: `2048`) |
| `overlap` | integer | Overlapping runes between chunks (default: `256`) |

#### RAG Search Parameters

| Key | Type | Description |
|---|---|---|
| `limit` | integer | Maximum results per query (default: `10`) |
| `threshold` | float | Minimum cosine similarity score (default: `0.40`) |
| `multi_query` | integer | Number of additional query reformulations (default: `2`) |
| `dedup_threshold` | float | Near-duplicate suppression threshold (default: `0.95`) |

#### Qdrant Configuration

| Key | Type | Description |
|---|---|---|
| `connstr` | string | Qdrant connection string (REST port 6333; gRPC 6334 used automatically) |
| `collection` | string | Collection name (created automatically if missing) |

#### Chromem Configuration

The Chromem vector store is a built-in, dependency-free alternative to Qdrant. It stores vectors in a local directory and supports persistent storage.

| Key | Type | Description |
|---|---|---|
| `data_dir` | string | Persistent data directory (default: `~/.go-magnetar/store`) |
| `collection` | string | Collection name (default: `documents`) |

### Plugin Configuration

All plugins are disabled by default. Enable a plugin by setting `enable: true`:

#### Confluence (`confluence`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable Confluence fetching (default: `false`) |
| `readonly` | boolean | Read-only mode: disables page editing (default: `false`) |
| `base_url` | string | Confluence base URL (e.g., `https://your-domain.atlassian.net`) |
| `api_key` | string | Confluence API key |

#### JIRA (`jira`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable JIRA fetching (default: `false`) |
| `readonly` | boolean | Read-only mode: disables issue creation and other write operations (default: `false`) |
| `base_url` | string | JIRA base URL (e.g., `https://jira.example.com`) |
| `api_key` | string | JIRA API key for REST API v2 authentication |

#### GitLab (`gitlab`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable GitLab fetching (default: `false`) |
| `base_url` | string | GitLab base URL |
| `api_key` | string | GitLab API key |

#### GitHub (`github`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable GitHub fetching (default: `false`) |
| `base_url` | string | GitHub API base URL (default: `https://api.github.com`) |
| `access_key` | string | GitHub access token |

#### Mask (`mask`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable/disable automatic sensitive data masking (default: `true`) |
| `skip.email` | boolean | Skip email address masking (default: `false`) |
| `skip.phone` | boolean | Skip phone number masking (default: `false`) |

The mask plugin automatically redacts sensitive data (email addresses, phone numbers, GitHub tokens, GitLab tokens, Russian passports, Russian SNILS, US Social Security Numbers, credit card numbers) from all user inputs and tool responses to prevent accidental data leakage. Masking is applied to system prompts, user messages, and tool results.

These live in `plugins/mask.yml`. Add `plugins/mask.yml` to the `include` list of an existing `config.yml`; new configurations include it already.

See [Security > Sensitive Data Masking](./security.md#sensitive-data-masking) for more details about the masking mechanism.

#### Guard (`guard`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable guard agent for exec commands (default: `false`) |
| `ask` | boolean | Ask user for confirmation when guard blocks a command (default: `false`) |
| `allow` | array of strings | List of trusted utilities that are considered safe to execute (default: empty array). Example: `["ls", "pwd"]`. |

#### Web (`web`)

| Key | Type | Description |
|---|---|---|
| `disable.fetch` | boolean | Disable the `web_fetch` tool. The model is not offered it (default: `false`) |
| `disable.search` | boolean | Disable the `web_search` tool. The model is not offered it; `rag_search` is unaffected (default: `false`) |
| `proxy.http` | string | HTTP proxy URL (e.g., `http://proxy.example.com:8080`) |
| `proxy.https` | string | HTTPS proxy URL (e.g., `https://proxy.example.com:8080`) |
| `proxy.noproxy` | string | Comma-separated list of hosts that should bypass the proxy (e.g., `localhost,127.0.0.1,.internal.example.com`) |

The proxy settings apply to all HTTP clients created by go-magnetar, including LLM API calls, RAG embedding requests, and all web fetching operations (`web_fetch`, `web_search`, and tool-specific web calls for GitHub, GitLab, Confluence, and JIRA).

These live in `plugins/web.yml`. Add `plugins/web.yml` to the `include` list of an existing `config.yml`; new configurations include it already.

#### SSH (`ssh`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable SSH execution (default: `false`) |
| `user` | string | SSH username (optional, defaults to current user) |
| `key` | string | SSH private key path (default: `~/.ssh/id_rsa`) |
| `password` | string | SSH password (optional) |
| `use_ssh_agent` | boolean | Use ssh-agent for authentication (default: `false`) |
| `remote_dir` | string | Remote working directory (optional, default: `$HOME`) |
| `timeout` | integer | SSH command timeout in seconds (default: `120`) |

#### SMTP (`smtp`)

| Key | Type | Description |
|---|---|---|
| `enable` | boolean | Enable SMTP email sending (default: `false`) |
| `host` | string | SMTP server hostname (e.g., `smtp.gmail.com`) |
| `port` | integer | SMTP server port (25, 465, or 587) |
| `username` | string | SMTP authentication username |
| `password` | string | SMTP authentication password |
| `encryption` | string | Encryption mode: `tls` (port 465), `starttls` (port 587), or `none` (port 25) |
| `from_name` | string | Sender display name |
| `from_email` | string | Sender email address |

The SMTP plugin enables sending emails with Markdown content (automatically converted to HTML) and file attachments. When multiple recipients are specified, the first goes to the "To" field, while remaining recipients and the sender are added to the "Cc" field.

See [plugin_smtp.md](./plugin_smtp.md) for detailed information about SMTP configuration and usage examples.

#### MCP (`mcp`)

The MCP plugin configures the Model Context Protocol server. It supports two transport modes: `stdio` (default) and `http`.

| Key | Type | Description |
|---|---|---|
| `mode` | string | Transport mode: `stdio` (default) or `http` |
| `server.host` | string | Host for HTTP server (default: `localhost`) |
| `server.port` | integer | Port for HTTP server (default: `8080`) |
| `server.cert` | string | Path to TLS certificate file (empty for HTTP, required for HTTPS) |
| `server.key` | string | Path to TLS private key file (empty for HTTP, required for HTTPS) |
| `server.tls` | boolean | Enable TLS encryption for HTTP (default: `false`) |
| `server.endpoint` | string | HTTP endpoint path (default: `/mcp`) |
| `path.allowed` | array of strings | Allowed paths for file access in MCP mode (default: `[$env:HOME]`). Required for `file_read` tool to work in MCP mode. |

**Stdio Mode** (default, no additional configuration required):

**Stdio Mode** (default, no additional configuration required):
```yaml
mcp:
  mode: stdio
```

**HTTP Mode**:
```yaml
mcp:
  mode: http
  server:
    host: localhost
    port: 8080
    endpoint: "/mcp"
  path:
    allowed: [$env:HOME]  # allowed paths for file access
```

**HTTPS Mode** (requires TLS certificate and key):
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
  path:
    allowed: [$env:HOME]  # allowed paths for file access
```

See [mcp_mode.md](./mcp_mode.md) for detailed information about MCP mode and client connection examples.

### Compact Configuration

Controls conversation history compression:

| Key | Type | Description |
|---|---|---|
| `mode` | string | Compacting strategy: `soft` (preserve `save_tail` messages) or `compact` (summarize entire history into a single message; default: `soft`) |
| `threshold` | integer | Token threshold for compression; `0` = auto (80% of `llm.context`) |
| `save_tail` | integer | Number of trailing messages to preserve unchanged in `soft` mode (default: `6`) |

## Environment Variables and Configuration

go-magnetar supports setting configuration parameters via environment variables. This provides a secure way to manage sensitive data like API keys and configuration values without hardcoding them in config files.

### Environment Variable Mapping

Environment variables follow the `MAGNETAR_*` prefix convention and are mapped to configuration parameters using the transformation:

```
MAGNETAR_{SECTION}_{PARAM} -> {section}.{param}
```

For example:
- `MAGNETAR_CONFLUENCE_READONLY` → `confluence.readonly`
- `MAGNETAR_MCP_MODE` → `mcp.mode`
- `MAGNETAR_MCP_SERVER_PORT` → `mcp.server.port`

All environment variable names use uppercase with underscores separating hierarchical sections. The transformation converts underscores to dots and transforms the key to lowercase.

### Environment Variable Usage

Environment variables can be set directly or referenced in config files using `$env:VAR_NAME`:

```bash
export MAGNETAR_CONFLUENCE_READONLY=true
export MAGNETAR_JIRA_API_KEY="your-api-key"
export MAGNETAR_MCP_MODE="http"
export MAGNETAR_MCP_SERVER_PORT="9000"
```

```yaml
jira:
  enable: true
  base_url: $env:MAGNETAR_JIRA_BASE_URL
  api_key: $env:MAGNETAR_JIRA_API_KEY
  readonly: $env:MAGNETAR_JIRA_READONLY

mcp:
  mode: $env:MAGNETAR_MCP_MODE
  server:
    port: $env:MAGNETAR_MCP_SERVER_PORT
```

If an environment variable is not set, it will be replaced with an empty string.

Environment variables take precedence over values defined in configuration files, allowing easy overrides for different environments (development, production, etc.).

### Environment Variables and File Substitution

Environment variables also support file substitution syntax for dynamic file paths:

```yaml
llm:
  api_key: $file:$env:API_KEY_FILE
```

The environment variable is substituted first, then the file is read.

### File Content Substitution

Use `$file:filename` to substitute file contents:

```yaml
llm:
  api_key: $file:secrets/api_key.txt
```

File paths are resolved relative to the configuration file location. This allows organizing secrets in subdirectories relative to your config file.

Absolute paths and `~/` home directory shortcuts are also supported:

```yaml
llm:
  api_key: /absolute/path/to/api_key.txt
  secret: ~/.secrets/my-secret.txt
```

File contents are substituted at read time. If a file is not found, it will be replaced with an empty string.

Environment variables can be used in file paths to dynamically resolve filenames:

```yaml
llm:
  api_key: $file:$env:API_KEY_FILE
```

The environment variable is substituted first, then the file is read.

## Include Directive

Use the `include` directive to split configuration across multiple files:

```yaml
version: "1.0"
profile: default
include:
  - plugins/jira.yml
  - plugins/github.yml
  - plugins/rag.yml
  - plugins/guard.yml
  - plugins/mcp.yml
```

Include files are loaded **after** the main config, so they **override** values from the main configuration file.

### Config Loading Order

1. Default values
2. Main config file
3. Include files (override main config values)

This order ensures that plugin-specific settings in include files take precedence over general settings in the main config.

## Example Configuration

See [example-config.md](./example-config.md) for a complete example configuration file.

### MCP Configuration Example

```yaml
version: "1.0"
profile: default
include:
  - plugins/mcp.yml

mcp:
  mode: http  # or stdio (default)
  server:
    host: localhost
    port: 8080
    endpoint: "/mcp"
  path:
    allowed: [$env:HOME]  # allowed paths for file access
```

For more information about MCP mode, available tools, and client connection examples, see [mcp_mode.md](./mcp_mode.md).

### JIRA Configuration Example

```yaml
version: "1.0"
profile: default
include:
  - plugins/jira.yml

jira:
  enable: true
  base_url: https://jira.example.com
  api_key: $env:JIRA_API_KEY
  readonly: false  # Set to true to disable issue creation
```

For more information about JIRA tools and available operations, see [tool_jira.md](./tool_jira.md).

### SMTP Configuration Example

```yaml
version: "1.0"
profile: default
include:
  - plugins/smtp.yml

smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  username: $env:SMTP_USERNAME
  password: $env:SMTP_PASSWORD
  encryption: starttls
  from_name: "Your Name"
  from_email: your-email@gmail.com
```

For more information about SMTP tools and available operations, see [plugin_smtp.md](./plugin_smtp.md).
