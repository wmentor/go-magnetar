# go-magnetar

A knowledge base tool built on RAG (Retrieval-Augmented Generation). Combines a **chat agent** with an integrated **index command** for document ingestion — all in a unified interactive REPL.

## Requirements

- Go 1.27.0
- Qdrant (gRPC port `6334`)
- API key for OpenAI-compatible LLM and embedding model

## Documentation

- All documentation, comments, and technical writing must be written in English only

Start Qdrant locally:

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## Build

```bash
make build        # binary -> bin/go-magnetar
make tidy         # synchronize go.mod / go.sum
make lint         # go vet ./...
make clean        # remove bin/
```

## Configuration

See [docs/configuration.md](./docs/configuration.md) for complete configuration options.

### Profile parameters

When accessing profile-specific parameters in plugins, use the `ProfileParam*` helper methods. These automatically prepend the `profiles.{profile_name}.` prefix based on the currently selected profile.

| Method | Type | Description |
|---|---|---|
| `ProfileParamString(key string) string` | string | Returns string value for the given key |
| `ProfileParamInt(key string) int` | int | Returns integer value for the given key |
| `ProfileParamFloat64(key string) float64` | float64 | Returns float64 value for the given key |
| `ProfileParamBool(key string) bool` | bool | Returns boolean value for the given key |

**Important:** In the `key` parameter, specify only the path portion after `profiles.{profile_name}.`. For example:
- To access `profiles.default.llm.api_key`, call `cfg.ProfileParamString("llm.api_key")`
- To access `profiles.default.llm.temperature`, call `cfg.ProfileParamFloat64("llm.temperature")`

The `profiles.{profile_name}.` prefix is automatically added based on the `profile` field in the configuration.

Example usage in a plugin:

```go
func (p *MyPlugin) Init(s plugin.State, hub plugin.Hub) error {
    cfg := s.Config()
    
    // Automatically resolves to profiles.default.llm.api_key
    apiKey := cfg.ProfileParamString("llm.api_key")
    
    // Automatically resolves to profiles.default.llm.temperature
    temperature := cfg.ProfileParamFloat64("llm.temperature")
    
    // Automatically resolves to profiles.default.llm.context
    context := cfg.ProfileParamInt("llm.context")
    
    // Automatically resolves to profiles.default.llm.model
    model := cfg.ProfileParamString("llm.model")
    
    // ...
}
```

## CLI

The `-c`/`--config` flag is **global** — it must be placed before the command:

```
go-magnetar [-c <config>] <command> [flags]
```

If omitted, `~/.go-magnetar.yaml` is used. Can also be set via `GO_MAGNETAR_CONFIG` env var.

### `-f/--file` — non-interactive mode

```
go-magnetar [-c <config>] -f <input-file>
```

Reads input from the specified file, sends it to the agent, prints the answer, and exits. This mode is useful for scripting and batch processing.

go-magnetar has a **single unified command** — `agent` — which provides an interactive REPL. The indexer functionality is now a chat command (`/index`).

## Chat Agent (Unified REPL)

The agent provides an interactive REPL with multi-turn conversation support and integrated document indexing.

```bash
./bin/go-magnetar -c my-config.yaml agent
# or via Makefile (uses configs/config.yaml):
make run-agent
```

### Working with the agent

```
> What is go-magnetar?
go-magnetar is a RAG-based knowledge base tool...

> /index docs/guide.md or docs/report.odt
Indexed 15 chunks from docs/guide.md or docs/report.odt

> What commands does it support?
It supports the /index command and chat commands like /help, /exit...
```

Exit — `Ctrl+D` (EOF) or `/exit` command. Empty lines are ignored.

### Built-in chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `/h` | Show the list of available commands (assembled dynamically from all registered plugins) |
| `/exit` | `/quit` | End the session and exit the program |
| `/compact` | — | Immediately compress history via summarizer, without waiting for automatic threshold |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: number of messages, estimated tokens, size in bytes, LLM model name, RAG model name, and vector size |
| `/index` | `/i` | Index file or URL into RAG knowledge base (auto-detects URL vs file) |
| `/idxtab` | — | Index multiple files/URLs from a JSON lines file (one per line, format: `{"source":"path|url","message":"text"}`) |
| `/write` | `/w` | Write content to a file |
| `/readonly` | — | Toggle read-only mode (blocks all modification operations) |
| `/fetch` | `/f` | Fetch content from a URL, optionally save to file |

Commands are dispatched in `handleCommand` (`internal/agent/chat/agent.go`) by iterating over `plugin.ChatCommands()`. Input is split into `name` + `args` on the first space; matching is case-insensitive against `Name` and `Aliases`. Commands are never added to the message history.

### Command history

The chat agent maintains command history in `~/.go-magnetar-history.json`. Use **↑/↓** arrows to navigate through previous commands. History is persisted across sessions and limited to 200 entries.

### Indexing via chat

The `/index` command replaces the separate indexer CLI subcommand:

| Command | Aliases | Description |
|---|---|---|
| `/index <path|url> [-m <message>]` | `/i` | Index file or URL into RAG knowledge base (auto-detects URL vs file) |

The command auto-detects whether the argument is a file path or URL based on protocol prefix. Supports `-m <message>` to prepend context to each chunk.

### Fetching from URLs

The `/fetch` command retrieves and displays content from URLs:

| Command | Aliases | Description |
|---|---|---|
| `/fetch <url> [file]` | `/f` | Fetch content from URL, display or save to file |

The command uses the configured `webfetch` block to clean HTML content and convert it to Markdown. If a filename is provided, the content is saved to that file; otherwise, it's displayed in the terminal (using `less` if available).

Example:
```bash
> /fetch https://example.com/article
> /fetch https://example.com/article output.md
```

### Testing

go-magnetar uses the **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** framework for its interactive REPL, which requires a real terminal interface with keyboard input. Therefore, **you cannot test go-magnetar by piping data to stdin**.

The Bubble Tea program uses `tea.WithOutput(os.Stderr)` and expects interactive key presses to collect user input via `readInput()` function. Attempting to pipe or redirect input (e.g., `echo "test" | go-magnetar agent`) will not work because Bubble Tea takes control of stdin for keyboard events.

To test go-magnetar, run it interactively:

```bash
./bin/go-magnetar -c my-config.yaml agent
> /help
> What is go-magnetar?
```

### Non-interactive mode

For scripting and batch processing, use the `-f`/`--file` flag:

```bash
go-magnetar [-c <config>] -f <input-file>
```

This mode reads input from the specified file, sends it to the agent, prints the answer, and exits.

> **Note:** In `-f/--file` mode, the text preprocessor is not applied and chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available. Only the raw input from the file is processed.

### Search strategy

The agent **always** first calls `rag_search`, even if it believes it already knows the answer. If `rag_search` returns relevant results, the answer is formed exclusively based on those results, `web_fetch` is not called. `web_fetch` is used only as a fallback: when `rag_search` returns no relevant results and the user needs external or up-to-date information. If neither tool provides a result, the agent explicitly states this.

`rag_search` internally runs multiple queries in parallel when `rag.search.multi_query > 0`: the original query plus N LLM-generated reformulations. Results are merged by keeping the best score per unique chunk, trimmed to `rag.search.limit`, and then near-duplicates are suppressed based on `rag.search.dedup_threshold`.

### Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.

### Chat agent tools

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem; supports `.txt`, `.md`, `.docx`, `.pdf`, `.odt`; `limit` and `offset` specify line range (0 = read all) |
| `file_list` | `(filter: string) -> []string` | Recursively lists files in the current directory using glob pattern (e.g. `*.go`) |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem (blocked in read-only mode) |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment, current working directory, and built-in safety guard |
| `system_date` | `() -> string` | Executes the date command to get the current system time |
| `system_grep` | `(filename: string, pattern: string) -> string` | Executes system grep command with safe parameters: -n (always), -i (case-insensitive), -r (recursive), -E (extended regex) |
| `rag_search` | `(query: string) -> string` | Returns top-N relevant fragments from Qdrant (N is set by `rag.search.limit`) |
| `web_fetch` | `(url: string) -> string` | Fetches and cleans a web page (fallback if RAG returns no results); also fetches Confluence pages and JIRA issues when URL matches |
| `github_repo` | `(repo: string) -> string` | Fetches GitHub repository information and returns its details in Markdown format |
| `github_file` | `(repo: string, branch: string, file: string) -> string` | Fetches a file from GitHub repository and returns its content |
| `github_tree` | `(repo: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |

## Security restrictions

See [docs/security.md](./docs/security.md) for complete security information.

## Indexer (via `/index` command)

The indexer reads `.md`, `.txt`, `.docx`, and `.pdf` files or web pages (by URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores them in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content — re-indexing the same file does not create duplicates.

**Note:** If you need to read a `.docx` or `.pdf` file directly in your code, use the `internal/docx.ReadFile` or `internal/pdf.ReadFile` package functions instead of `os.ReadFile`.

### Index a single file

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index path/to/document.md
> /index path/to/document.docx
> /index path/to/document.pdf
```

### Index a URL

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index https://example.com/article

# From Confluence URL (standard or short link)
> /index https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456

# From JIRA issue URL
> /index https://jira.example.com/browse/PROJECT-123
```

### Prepend message to chunks

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index path/to/document.md -m "Custom context or metadata"
```

The `-m` option prepends the specified message to each chunk for improved search quality.

### Indexing multiple files via `/idxtab`

The `/idxtab` command indexes multiple documents from a JSON lines file:

```bash
./bin/go-magnetar -c my-config.yaml agent
> /idxtab configs/cmd_idxtab.txt
```

Each line should have format: `{"source":"path|url","message":"text"}`

**Example file** (`configs/cmd_idxtab.txt`):

```json
{"source":"docs/guide.md","message":"API docs"}
{"source":"https://example.com/article","message":""}
{"source":"docs/api.md","message":"API reference"}
{"source":"https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456","message":""}
{"source":"https://jira.example.com/browse/PROJECT-123","message":""}
```

The command processes entries sequentially, auto-detects URLs vs file paths, and supports prepending context via the `message` field. Errors for individual entries don't stop processing — they are logged and skipped.

### Duplicate handling

Each chunk's ID is UUID v5 derived from its text (`uuid.NewSHA1`). Repeated calls to `rag_save` with the same content perform `Upsert` to the same ID — the existing point is overwritten, not duplicated.

## Architecture

See [docs/architecture.md](./architecture.md) for complete architecture documentation.

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/alecthomas/kong` | CLI parser |
| `github.com/sashabaranov/go-openai` | OpenAI API client (LLM + embeddings) |
| `github.com/qdrant/go-client` | Qdrant client (gRPC) |
| `github.com/google/uuid` | UUID v5 for deterministic chunk IDs |
| `github.com/knadh/koanf/v2` | YAML config loading |
| `github.com/charmbracelet/glamour` | Markdown rendering in terminal |
| `log/slog` | Structured logging (stdlib, replaced by internal/printer) |

## Security restrictions

See [docs/security.md](./docs/security.md) for complete security information.
