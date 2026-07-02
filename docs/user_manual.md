# go-magnetar User Manual

## Description

go-magnetar — a knowledge base tool built on RAG (Retrieval-Augmented Generation). It combines a chat agent with an integrated `/index` command for document ingestion.

- **Chat agent** — answers questions strictly based on indexed data (no hallucinations or guessing)
- **Index command** — `/index <path|url> [-m <message>]` indexes documents directly from the chat REPL

> If the `webfetch` block is configured, it is used to clean HTML content from ads, navigation, and other noise when processing web pages. Confluence URLs (both standard and short links) are also supported via the `confluence` block. JIRA issues are supported via the `jira` block. GitLab merge requests (including file changes) are supported via the `gitlab` block. GitHub repositories, files, and directory trees are supported via the `github` block.

## Requirements

- Go 1.26.4+
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

```bash
cp configs/config.yaml my-config.yaml
```

Edit `my-config.yaml` with your values:

```yaml
llm:
  base_url: https://api.openai.com/v1
  api_key: sk-...
  model: gpt-4o
  context: 128000

rag:
  llm:
    base_url: https://api.openai.com/v1
    api_key: sk-...
    model: text-embedding-3-small
    vector_size: 1536
  disable: false
  chunk:
    size: 512      # max chunk size in runes (default: 512)
    overlap: 64    # overlap between adjacent chunks in runes (default: 64)
  search:
    limit: 10          # max results per query (default: 10)
    threshold: 0.40    # min cosine-similarity score 0–1 (default: 0.40)
    multi_query: 2     # extra LLM-generated query variants (default: 2, 0 = off)
    dedup_threshold: 0.95  # near-duplicate suppression threshold (default: 0.95, 0 = off)
  qdrant:
    connstr: http://localhost:6333
    collection: documents

log:
  level: info

verbose: true

compact:
  threshold: 0   # 0 = auto (80 % of llm.context)
  save_tail: 6   # keep the last 6 messages verbatim

webfetch:
  base_url: https://api.openai.com/v1
  api_key: sk-...
  model: gpt-4o
  context: 128000
  disable: false

confluence:
  base_url: https://your-domain.atlassian.net
  api_key: YOUR_API_KEY
  disable: false

jira:
  base_url: https://jira.example.com
  api_key: YOUR_API_KEY
  disable: false

gitlab:
  base_url: https://gitlab.example.com
  api_key: YOUR_API_KEY
  disable: false

github:
  base_url: https://api.github.com
  api_key: YOUR_GITHUB_TOKEN
  disable: false
```

> If the `webfetch` block is specified, the model parameters listed above are used to clean HTML content obtained from web pages.

> The `confluence` block enables fetching Confluence pages directly by URL (both standard and short links).

> The `jira` block enables fetching JIRA issues directly by URL.

> The `gitlab` block enables fetching GitLab merge requests directly by URL, including file changes via the `/changes` endpoint.
> The `github` block enables fetching GitHub repositories, files, and directory trees directly by repository path.

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

### 5. Ask Questions

```bash
./bin/go-magnetar -c my-config.yaml agent
```

```
> What is go-magnetar?
go-magnetar is a RAG-based knowledge base tool...

> What commands does it support?
It supports the /index command and chat commands like /help, /exit...

> ^D
```

## CLI

The `-c`/`--config` flag is **global** and must be placed before the command:

```
go-magnetar [-c <config>] <command> [flags]
```

If `-c` is omitted, `~/.go-magnetar.yaml` is used. The flag can also be set via the `GO_MAGNETAR_CONFIG` environment variable.

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

Commands are case-insensitive, processed locally, and never sent to the LLM.

#### Command history

Use **↑/↓** arrows to navigate through previously entered commands. History is persisted in `~/.go-magnetar-history.json` and limited to 200 entries.

### Chat tools

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem |
| `file_list` | `(options: object) -> []string` | Recursively lists files in the current directory |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment and current working directory |
| `system_date` | `() -> string` | Executes the date command to get the current system time |
| `system_grep` | `(filename: string, pattern: string, case_insensitive: bool, recursive: bool) -> string` | Executes system grep command with safe parameters |
| `rag_search` | `(query: string) -> string` | Returns relevant fragments from indexed data |
| `web_fetch` | `(url: string) -> string` | Fetches web pages (fallback if RAG returns no results); also fetches Confluence pages, JIRA issues, GitLab merge requests, and GitHub repositories when URL matches |
| `ask` | `(question: string) -> string` | Asks the user a clarifying question and returns the answer |
| | `(filename: string, operations: array) -> string` | Applies regex-based search-and-replace operations to a file; supports optional context constraints and file length limits |

## Security restrictions

The following commands are automatically blocked by the `exec` tool:

- **Destructive commands**: `rm -rf /`, `sudo`, `mkfs`, `dd` with `/dev/`
- **Untrusted script execution**: `curl ... | bash`, `wget ... | sh`, `zsh`
- **Git modifications**: `git commit`, `git push`, `git rebase`, `git pull`, `git cherry-pick`, `git reset`, `git stash`, `git clean`, `git reflog`

No output is ever displayed from blocked commands — they return an error message instead.

#### Search replace functionality

The tool allows for regex-based search-and-replace operations on files:

| Parameter | Description |
|---|---|
| `filename` | Path to the file to modify |
| `operations` | Array of search-and-replace operations |

Each operation can contain the following fields:

| Field | Type | Description |
|---|---|---|
| `search` | string | Regex pattern to search for (required) |
| `replace` | string | Replacement string (supports $1, $2, etc.) (required) |
| `before` | string | Optional context before the match (plain text, not regex) |
| `after` | string | Optional context after the match (plain text, not regex) |
| `max_len` | integer | Maximum file length allowed for this operation (0 = unlimited) |

The tool validates regex patterns, checks context constraints, verifies file size limits, and applies all successful operations sequentially. It returns success status, modified content, and a list of any errors encountered.

### Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.

## Configuration

| Parameter | Default | Description |
|---|---|---|
| `llm.base_url` | — | Endpoint of OpenAI-compatible API for chat model |
| `llm.api_key` | — | API key for chat model |
| `llm.model` | — | Chat model name (e.g. `gpt-4o`) |
| `llm.context` | — | Model context token limit |
| `rag.llm.base_url` | — | Endpoint for embedding model |
| `rag.llm.api_key` | — | API key for embedding model |
| `rag.llm.model` | — | Embedding model name (e.g. `text-embedding-3-small`) |
| `rag.llm.vector_size` | — | Vector dimensionality — must match the embedding model |
| `rag.chunk.size` | `512` | Maximum chunk size in Unicode runes |
| `rag.chunk.overlap` | `64` | Overlap between adjacent chunks in runes (~12.5 %) |
| `rag.search.limit` | `10` | Maximum number of results per query |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity score; results below this are discarded |
| `rag.search.multi_query` | `2` | Number of extra query reformulations generated by the LLM to improve recall. `0` disables multi-query |
| `rag.search.dedup_threshold` | `0.95` | Cosine similarity above which two result chunks are considered near-duplicates; the lower-scoring one is dropped. `0` disables deduplication |
| `rag.qdrant.connstr` | — | Qdrant address, e.g. `http://localhost:6333` |
| `rag.qdrant.collection` | — | Collection name (created automatically if absent) |
| `verbose` | `true` | Enable verbose output (tool calls, debug messages) |
| `compact.threshold` | `0` | Token threshold for history compression. `0` = auto (80 % of llm.context) |
| `compact.save_tail` | `6` | Number of most-recent messages kept verbatim during compression |
| `webfetch.base_url` | — | Endpoint of OpenAI-compatible API for web content cleaning model |
| `webfetch.api_key` | — | API key for web content cleaning model |
| `webfetch.model` | — | Model name for web content cleaning (e.g. `gpt-4o`) |
| `webfetch.context` | — | Token limit of the model's context window for web cleaning |
| `confluence.base_url` | — | Confluence instance base URL (e.g. `https://your-domain.atlassian.net`) |
| `confluence.api_key` | — | Confluence API key for fetching pages |
| `jira.base_url` | — | JIRA instance base URL |
| `jira.api_key` | — | JIRA API key for fetching issues |
| `gitlab.base_url` | — | GitLab instance base URL |
| `gitlab.api_key` | — | GitLab API key for fetching merge requests and file changes |
| `github.base_url` | — | GitHub API base URL (e.g. `https://api.github.com`) |
| `github.api_key` | — | GitHub access token for fetching repositories, files, and trees |

**Vector sizes for common embedding models:**

| Model | `vector_size` |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-ada-002` | 1536 |
| `text-embedding-3-large` | 3072 |

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

## License

See [LICENSE](../LICENSE).
