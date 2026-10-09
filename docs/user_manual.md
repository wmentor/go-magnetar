# go-magnetar User Manual

## Description

go-magnetar — a knowledge base tool built on RAG (Retrieval-Augmented Generation). It combines a chat agent with an integrated `/index` command for document ingestion and a `/fetch` command for URL content retrieval.

- **Chat agent** — answers questions strictly based on indexed data (no hallucinations or guessing)
- **Index command** — `/index <path|url> [-m <message>]` indexes documents directly from the chat REPL
- **Fetch command** — `/fetch <url> [file]` retrieves and displays content from URLs

> Web pages are cleaned of ads, navigation, and other noise, then processed through readability extraction and converted to Markdown. Confluence URLs (both standard and short links) are also supported via the `confluence` block. JIRA issues are supported via the `jira` block. GitLab merge requests (including file changes) are supported via the `gitlab` block. GitHub repositories, files, and directory trees are supported via the `github` block.

Supported file formats for indexing: `.md`, `.txt`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, and `.html`. See [docs/file_codecs.md](./docs/file_codecs.md) for complete format support and codec details.

## Requirements

- Go 1.27.2
- API key for any OpenAI-compatible provider (for LLM, embedding model, and optionally for web content cleaning)

**Note:** Qdrant is **no longer required**. The default vector store is **Chromem** (built-in, in-memory with optional persistence). Qdrant is available as an optional external store for production deployments.

### Vector Stores

go-magnetar supports two vector storage backends:

#### Chromem (default)

- **Built-in**: No external dependencies required
- **In-memory by default**: Store exists only during session (when `rag.chromem.data_dir` is empty)
- **Persistent mode**: Set `rag.chromem.data_dir` to persist vectors to disk
- **Use case**: Local development, testing, edge deployments

#### Qdrant (external)

- **External server**: Requires running Qdrant container
- **Full persistence**: Automatic disk persistence with advanced filtering
- **Use case**: Production, large-scale deployments

Switch between stores via config:
```yaml
rag:
  store:
    type: chromem  # or qdrant
```

See [docs/architecture.md](./architecture.md) for vector store architecture details.

## Quick Start

### 1. Build the Binary

```bash
make build
```

The binary will be placed at `bin/go-magnetar`.

### 2. Configuration

See [configuration.md](./configuration.md) for complete configuration options.

### 3. Index Documents

```bash
./bin/go-magnetar
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

> Web pages are cleaned of ads and navigation, then processed through readability extraction and converted to Markdown before being indexed. Confluence pages are also indexed via the `confluence` block. JIRA issues are indexed via the `jira` block.

### 4. Fetch Content from URLs

The `/fetch` command retrieves content from URLs, cleans HTML, and displays it in the terminal:

```bash
./bin/go-magnetar
> /fetch https://example.com/article

# Save to file
> /fetch https://example.com/article output.md
```

The command cleans HTML content and converts it to Markdown. If a filename is provided, the content is saved to that file; otherwise, it's displayed in the terminal (using `less` if available).

### 5. Ask Questions

```bash
./bin/go-magnetar
```

```
> What is go-magnetar?
go-magnetar is a RAG-based knowledge base tool...

> What commands does it support?
It supports the /index and /fetch commands and chat commands like /help, /exit...

> ^D
```

## CLI

Configuration file is always loaded from `~/.go-magnetar/config.yml`.

### `-p/--profile` — select configuration profile

```
go-magnetar -p <profile_name>
```

The `-p` flag overrides the profile specified in the configuration file. This allows switching between different configurations (e.g., `default`, `production`, `development`) without modifying the config file.

### `-f/--file` — non-interactive mode

```
go-magnetar -f <input-file>
```

Reads input from the specified file, sends it to the agent, prints the answer, and exits. This mode is useful for scripting and batch processing.

> **Note:** In `-f/--file` mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available. The text preprocessor expands placeholders like `{{home}}`, `{{uuid}}`, `{{date}}`, `{{now}}`, and `{{file:filename}}`.

## Commands

### Interactive chat

```
go-magnetar
```

The REPL reads questions from stdin. Press `Ctrl+D` to exit.

#### Chat commands

See [docs/chat_command.md](./chat_command.md) for complete documentation on chat commands.

#### Text preprocessing

## Text preprocessing

See [preprocessor.md](./preprocessor.md) for a complete reference on text preprocessors and available placeholders.

## Command history

Use **↑/↓** arrows to navigate through previously entered commands. History is persisted in `~/.go-magnetar-history.json` and limited to 200 entries.

### Chat tools

See [docs/agent_tools.md](./agent_tools.md) for a complete list of available tools, search strategy, and search tool call limits.

## Data Residency

go-magnetar offers **100% local processing** options:

- **Chromem** (built-in vector store) — no external dependencies
- **Local LLMs** (Ollama, lm-studio, etc.) — run on your machine
- **Zero data exfiltration** — everything stays on your computer

When configured with Chromem + local LLM, **your documents, embeddings, and conversations never leave your machine**.

See [docs/data_residency.md](./data_residency.md) for complete details.

## Security restrictions

See [docs/security.md](./security.md) for complete security information.

## Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`search` + `web_fetch`). The unified `search` tool counts as 1 search call (not 2 separate calls). By default, the limit is 20 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.

See [chat_agent_ask.md](./chat_agent_ask.md) for complete documentation on the Ask method, including tool categorization, protection mechanisms, and full algorithm description.



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
| `rag.search.limit` | `10` | How many candidate chunks vectore store returns per query |
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

See [docs/dependencies.md](./dependencies.md) for a complete list of dependencies.

## License

See [LICENSE](../LICENSE).
