# go-magnetar

A knowledge base tool built on RAG (Retrieval-Augmented Generation). Combines a **chat agent** with an integrated **index command** for document ingestion — all in a unified interactive REPL.

## How it works

**Indexing** — reads `.md`, `.txt`, `.docx`, `.pdf`, and `.odt` files or web pages (via URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores each chunk in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content, making re-indexing idempotent: the same chunk is never stored twice.

**Chat** — an interactive REPL with multi-turn conversation support. The agent always tries `rag_search` first; if the knowledge base returns relevant results, the answer is based exclusively on those results. `web_fetch` is used only as a fallback when the knowledge base has no relevant information and the user needs external or up-to-date data. If neither source has an answer, the agent says so explicitly. Conversation history is automatically managed to stay within the configured context window: when history approaches the token threshold it is compacted by the built-in summarizer, which replaces older turns with a concise summary while keeping the most recent ones verbatim.

**Indexing via chat** — the `/index` command (alias `/i`) replaces the separate indexer CLI subcommand. Simply type `/index <path|url>` in the REPL to index documents directly.

**Fetch content** — the `/fetch` command (alias `/f`) retrieves content from URLs, cleans HTML, and displays it in the terminal (using `less` if available) or saves it to a file.

**Search tool call limit** — to prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is reached, an error message is sent to the LLM.

**HTML Preprocessing** — when `webfetch.*` parameters are configured, web pages fetched via `web_fetch` are cleaned of ads, navigation, cookie banners, and other noise using an AI agent before being converted to Markdown and indexed or returned to the agent. Confluence URLs are also handled via the `confluence` block to fetch pages by ID, JIRA issues via the `jira` block, GitHub repositories via the `github` block to fetch repository information, files, and directory trees, and GitLab merge requests via the `gitlab` block to fetch MR details and file changes.

## Requirements

- Go 1.27.0
- [Qdrant](https://qdrant.tech/) — vector database
- An API key for any OpenAI-compatible provider (for the chat model, embedding model, and optionally for web page preprocessing)

## Quick start

### 1. Start Qdrant

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 2. Build the binary

```bash
make build
```

The binary will be placed at `bin/go-magnetar`.

### 3. Configure

See [docs/configuration.md](./docs/configuration.md) for complete configuration options.

### 4. Index your documents

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index docs/guide.md or docs/report.odt

# From URL
> /index https://example.com/article

# From Confluence URL (standard or short link)
> /index https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456

# From JIRA issue URL
> /index https://jira.example.com/browse/PROJECT-123

# From GitLab merge request URL
> /index https://gitlab.example.com/namespace/project/-/merge_requests/123

# From GitHub URL (repository, file, or tree)
> /index https://github.com/owner/repo
```

### 5. Fetch content from URLs

The `/fetch` command retrieves content from URLs, cleans HTML, and displays it in the terminal:

```bash
./bin/go-magnetar -c my-config.yaml agent
> /fetch https://example.com/article

# Save to file
> /fetch https://example.com/article output.md
```

### 6. Ask questions

```bash
./bin/go-magnetar -c my-config.yaml agent
```

```
> What is go-magnetar?
go-magnetar is a RAG-based knowledge base tool...

> What commands does it support?
It supports the /index and /fetch commands and chat commands like /help, /exit...

> ^D
```

## Installation

```bash
git clone https://github.com/wmentor/go-magnetar
cd go-magnetar
make build
```

## Command history

The chat agent maintains command history in `~/.go-magnetar-history.json`. Use **↑/↓** arrows to navigate through previous commands. History is persisted across sessions and limited to 200 entries.



## Commands

The `-c`/`--config` flag is global and must be placed **before** the command:

```
go-magnetar [-c <config>] <command> [flags]
```

If `-c` is omitted, `~/.go-magnetar.yaml` is used. The flag can also be set via the `GO_MAGNETAR_CONFIG` environment variable.

### `-p/--profile` — select configuration profile

```
go-magnetar [-c <config>] -p <profile_name>
```

The `-p` flag overrides the profile specified in the configuration file. This allows switching between different configurations (e.g., `default`, `production`, `development`) without modifying the config file.

### `-f/--file` — non-interactive mode

```
go-magnetar [-c <config>] -f <input-file>
```

Reads input from the specified file, sends it to the agent, prints the answer, and exits. This mode is useful for scripting and batch processing.

> **Note:** In `-f/--file` mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available. The text preprocessor expands placeholders like `{{home}}`, `{{uuid}}`, `{{date}}`, `{{now}}`, and `{{file:filename}}`.

```
go-magnetar [-c <config>]
```

Run the interactive agent REPL. Press `Ctrl+D` to exit.

#### Chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `/h` | Show the list of available chat commands |
| `/exit` | `/quit` | End the session and exit the program |
| `/compact` | — | Immediately compress the conversation history via the summarizer |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: messages, estimated tokens, bytes, LLM model, RAG model, vector size |
| `/index` | `/i` | Index file or URL into RAG knowledge base (auto-detects URL vs file) |
| `/idxtab` | — | Index multiple files/URLs from a JSON lines file (one per line, format: `{"source":"path\|url","message":"text"}`) |
| `/write` | `/w` | Write content to a file |
| `/readonly` | — | Toggle read-only mode (blocks all modification operations) |
| `/fetch` | `/f` | Fetch content from a URL, optionally save to file |

#### Text preprocessing

The agent applies text preprocessors to user input before processing. The built-in generic plugin provide the following placeholders that are automatically expanded:

| Placeholder | Description |
|---|---|
| `{{home}}` | Your home directory path |
| `{{uuid}}` | A random UUID v4 |
| `{{date}}` | Current date in `YYYY-MM-DD` format |
| `{{now}}` | Current date and time in `YYYY-MM-DD HH:MM:SS` format |

#### Reading file contents

The agent supports inline file content injection using the `{{file:filename}}` placeholder:

| Placeholder | Description |
|---|---|
| `{{file:filename}}` | Reads file content and replaces placeholder with the file contents. Supports `.md`, `.txt`, `.docx`, `.pdf`, and `.odt` files, absolute paths and `~/` home directory prefix |

Example: `{{file:~/documents/notes.md}}` or `{{file:~/documents/report.docx}}` or `{{file:~/documents/paper.pdf}}` or `{{file:~/documents/report.odt}}` will be replaced with the content of that file.

Commands are case-insensitive, processed locally, and never sent to the LLM.

#### Command history

The REPL supports command history navigation with arrow keys:
- **↑** — previous command
- **↓** — next command

History is persistently stored in `~/.go-magnetar-history.json` and limited to 200 entries.

## Makefile

```bash
make build      # compile -> bin/go-magnetar
make clean      # remove bin/
make run-agent  # build and run the agent with configs/config.yaml
make lint       # go vet ./...
make tidy       # go mod tidy
```

## Architecture

See [docs/architecture.md](./docs/architecture.md) for complete architecture documentation.

## Logging

Set `verbose: true` in the config for verbose output.

## Dependencies

| Package | Purpose |
|---|---|
| [`github.com/alecthomas/kong`](https://github.com/alecthomas/kong) | CLI parser |
| [`github.com/sashabaranov/go-openai`](https://github.com/sashabaranov/go-openai) | OpenAI API client (LLM + embeddings) |
| [`github.com/qdrant/go-client`](https://github.com/qdrant/go-client) | Qdrant client (gRPC) |
| [`github.com/google/uuid`](https://github.com/google/uuid) | UUID v5 for deterministic chunk IDs |
| [`github.com/knadh/koanf/v2`](https://github.com/knadh/koanf) | YAML config loading |
| [`github.com/charmbracelet/glamour`](https://github.com/charmbracelet/glamour) | Markdown rendering in terminal |
| `log/slog` | Structured logging (stdlib, replaced by internal/printer) |

## Security restrictions

See [docs/security.md](./docs/security.md) for complete security information.

## License

See [LICENSE](LICENSE).