# go-magnetar User Manual

## Description

go-magnetar — a knowledge base tool built on RAG (Retrieval-Augmented Generation). It combines a chat agent with an integrated `/index` command for document ingestion and a `/fetch` command for URL content retrieval.

- **Chat agent** — answers questions strictly based on indexed data (no hallucinations or guessing)
- **Index command** — `/index <path|url> [-m <message>]` indexes documents directly from the chat REPL
- **Fetch command** — `/fetch <url> [file]` retrieves and displays content from URLs

> If the `webfetch` block is configured, it is used to clean HTML content from ads, navigation, and other noise when processing web pages. Confluence URLs (both standard and short links) are also supported via the `confluence` block. JIRA issues are supported via the `jira` block. GitLab merge requests (including file changes) are supported via the `gitlab` block. GitHub repositories, files, and directory trees are supported via the `github` block.

Supported file formats for indexing: `.md`, `.txt`, `.docx`, `.pdf`, `.odt`

## Requirements

- Go 1.27.0
- [Qdrant](https://qdrant.tech/) — vector database
- API key for any OpenAI-compatible provider (for LLM, embedding model, and optionally for web content cleaning)

## Quick Start

### 1. Start Qdrant

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 2. Build the Binary

```bash
make build
```

The binary will be placed at `bin/go-magnetar`.

### 3. Configuration

See [configuration.md](./configuration.md) for complete configuration options.

### 4. Index Documents

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index docs/guide.md

# From URL (web page)
> /index https://example.com/article

# From Confluence URL
> /index https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456

# From JIRA issue URL
> /index https://jira.example.com/browse/PROJECT-123

# From GitLab merge request URL
> /index https://gitlab.example.com/namespace/project/-/merge_requests/123

# From GitHub URL (repository, file, or tree)
> /index https://github.com/owner/repo
```

> If the `webfetch` block is configured in the config, web pages are cleaned of ads and navigation using an AI agent before being converted to Markdown and indexed. Confluence pages are also indexed via the `confluence` block. JIRA issues are indexed via the `jira` block.

### 5. Fetch Content from URLs

The `/fetch` command retrieves content from URLs, cleans HTML, and displays it in the terminal:

```bash
./bin/go-magnetar -c my-config.yaml agent
> /fetch https://example.com/article

# Save to file
> /fetch https://example.com/article output.md
```

The command uses the configured `webfetch` block to clean HTML content and convert it to Markdown. If a filename is provided, the content is saved to that file; otherwise, it's displayed in the terminal (using `less` if available).

### 6. Ask Questions

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

## CLI

The `-c`/`--config` flag is **global** and must be placed before the command:

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

## Commands

### `agent` — interactive chat

```
go-magnetar [-c <config>] agent
```

The REPL reads questions from stdin. Press `Ctrl+D` to exit.

#### Chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `/h` | Show the list of available chat commands |
| `/exit` | `/quit` | End the session and exit the program |
| `/compact` | — | Immediately compress conversation history via the summarizer |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: messages, tokens, bytes, LLM model, RAG model, vector size |
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

Use **↑/↓** arrows to navigate through previously entered commands. History is persisted in `~/.go-magnetar-history.json` and limited to 200 entries.

### Chat tools

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem (supports `.md`, `.txt`, `.docx`, `.pdf`, `.odt`) |
| `file_list` | `(filter: string) -> []string` | Recursively lists files in the current directory using glob pattern (e.g. `*.go`) |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment and current working directory; security guard analyzes all commands before execution |
| `system_date` | `() -> string` | Executes the date command to get the current system time |
| `system_grep` | `(filename: string, pattern: string) -> string` | Executes system grep command with safe parameters: -n (always), -i (case-insensitive), -r (recursive), -E (extended regex) |
| `rag_search` | `(query: string) -> string` | Returns relevant fragments from indexed data |
| `web_fetch` | `(url: string) -> string` | Fetches web pages (fallback if RAG returns no results); also fetches Confluence pages, JIRA issues, GitLab merge requests, and GitHub repositories when URL matches |
| `ask` | `(question: string) -> string` | Asks the user a clarifying question and returns the answer |

## Security restrictions

See [docs/security.md](./security.md) for complete security information.

## Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.



## Chunking

Text is split into overlapping chunks:

- **Paragraph boundaries** — splits on blank lines (`\n\n`).
- **Markdown heading boundaries** — each heading starts a new chunk so section titles stay with their content.
- **Word-boundary snapping** — boundaries are aligned to word edges; no word is ever cut mid-character.
- **UTF-8 safe** — works correctly with Cyrillic, CJK, emoji.

Default values: `size = 512` runes, `overlap = 64` runes (~12.5 %).

## Search Tuning

Four parameters control retrieval quality:

| Parameter | Default | Effect |
|---|---|---|
| `rag.search.limit` | `10` | How many candidate chunks Qdrant returns per query |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; lower = more results but more noise |
| `rag.search.multi_query` | `2` | Extra query reformulations generated by the LLM and searched in parallel. Improves recall when the index uses different phrasing than the user's question |
| `rag.search.dedup_threshold` | `0.95` | Near-duplicate suppression: drops the lower-scoring chunk when two results are too similar. Keeps the context window efficient |

### Recommendations

For a dense, single-topic knowledge base (technical documentation):
- `threshold: 0.40–0.45`, `multi_query: 1–2`, `dedup_threshold: 0.95`

For mixed-topic or large bases:
- `threshold: 0.25–0.35`, `multi_query: 2–3`, `dedup_threshold: 0.92`

To disable the new features (single-query, no dedup):
- `multi_query: 0`, `dedup_threshold: 0`

## Logging

Set `verbose: true` in the config for verbose output.

## Dependencies

See [LICENSE](../LICENSE).