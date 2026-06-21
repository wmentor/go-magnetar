# go-magnetar User Manual

## Description

go-magnetar — a knowledge base tool built on RAG (Retrieval-Augmented Generation). It combines two agents:
- **Indexer** — splits documents into semantic chunks and stores them in a vector database
- **Chat agent** — answers questions strictly based on indexed data (no hallucinations or guessing)

> If the `webfetch` block is configured, it is used to clean HTML content from ads, navigation, and other noise when processing web pages.

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
  chunk:
    size: 512      # max chunk size in runes (default: 512)
    overlap: 64    # overlap between adjacent chunks in runes (default: 64)
  search:
    limit: 10      # max results per query (default: 10)
    threshold: 0.40  # min cosine-similarity score 0–1 (default: 0.40)
  qdrant:
    connstr: http://localhost:6333
    collection: documents

log:
  level: info

compact:
  threshold: 0   # 0 = auto (80 % of llm.context)
  save_tail: 6   # keep the last 6 messages verbatim

webfetch:
  base_url: https://api.openai.com/v1
  api_key: sk-...
  model: gpt-4o
  context: 128000
```

> If the `webfetch` block is specified, the model parameters listed above are used to clean HTML content obtained from web pages.

### 4. Index Documents

```bash
# Single file
./bin/go-magnetar indexer -c my-config.yaml -f docs/guide.md

# From URL
./bin/go-magnetar indexer -c my-config.yaml --url https://example.com/article
```

> If the `webfetch` block is configured in the config, web pages are cleaned of ads and navigation using an AI agent before being converted to Markdown and indexed.

### 5. Ask Questions

```bash
./bin/go-magnetar agent -c my-config.yaml
```

```
> What is go-magnetar?
go-magnetar is a RAG-based knowledge base tool...

> What commands does it support?
It supports two commands: indexer and agent...

> ^D
```

## Commands

### `indexer` — document indexing

```
go-magnetar indexer -c <config> [-f <file>] [--url <url>] [-m <message>]
```

| Flag | Description |
|---|---|
| `-c` | Path to config file |
| `-f` | Path to a single `.md` or `.txt` file to index |
| `--url` | URL to fetch and index |
| `-m` | Message to prepend to each chunk for improved search |

Specify either `-f` or `--url`. If a file fails to read, an error will be logged.

### `agent` — interactive chat

```
go-magnetar agent -c <config>
```

| Flag | Description |
|---|---|
| `-c` | Path to config file |

The REPL reads questions from stdin. Press `Ctrl+D` to exit.

#### Chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `help` | Show the list of available chat commands |
| `/exit` | `exit` | End the session and exit the program |
| `/compact` | — | Immediately compress conversation history |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics |

### Chat tools

| Tool | Description |
|---|---|
| `file_read` | Reads file contents from the filesystem |
| `file_list` | Recursively lists files in the current directory |
| `file_write` | Writes content to a file in the filesystem |
| `rag_search` | Returns relevant fragments from indexed data |
| `web_fetch` | Fetches web pages (used only if RAG returns no results) |

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
| `rag.qdrant.connstr` | — | Qdrant address, e.g. `http://localhost:6333` |
| `rag.qdrant.collection` | — | Collection name (created automatically if absent) |
| `log.level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `compact.threshold` | `0` | Token threshold for history compression. `0` = auto (80 % of llm.context) |
| `compact.save_tail` | `6` | Number of most-recent messages kept verbatim during compression |
| `webfetch.base_url` | — | Endpoint of OpenAI-compatible API for web content cleaning model |
| `webfetch.api_key` | — | API key for web content cleaning model |
| `webfetch.model` | — | Model name for web content cleaning (e.g. `gpt-4o`) |
| `webfetch.context` | — | Token limit of the model's context window for web cleaning |

**Vector sizes for common embedding models:**

| Model | `vector_size` |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-ada-002` | 1536 |
| `text-embedding-3-large` | 3072 |

## Chunking

Text is split into overlapping chunks:

- **Paragraph boundaries** — splits on blank lines (`\n\n`).
- **Markdown heading boundaries** — each heading starts a new chunk.
- **UTF-8 safe** — works correctly with Cyrillic, CJK, emoji.

Default values: `size = 512` runes, `overlap = 64` runes (~12.5 %).

## Search Tuning

Two parameters control retrieval quality:

| Parameter | Default | Effect |
|---|---|---|
| `rag.search.limit` | `10` | How many candidate chunks Qdrant returns |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; lower = more results but more noise |

For a dense, single-topic knowledge base (technical documentation) use `threshold: 0.40–0.45`. For mixed-topic bases lower it to `0.25–0.35`.

## Logging

All logs are written to `stderr`. Log levels:

| Level | When |
|---|---|
| `DEBUG` | Detailed tool call information |
| `INFO` | Operation start/end information |
| `WARN` | Warnings (e.g., chunk overwrites) |
| `ERROR` | File read errors, connection failures, etc. |

Set `log.level: debug` in the config for verbose output.

## License

See [LICENSE](LICENSE).
