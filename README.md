# go-magnetar

A knowledge base tool built on RAG (Retrieval-Augmented Generation). Combines three agents: an **indexer** that splits documents into semantic chunks and stores them in a vector database, a **chat agent** that answers questions strictly based on the indexed data — no hallucinations, no guessing, and an **HTML preprocessor** that cleans web pages and converts them to clean Markdown.

## How it works

**Indexing** — the agent reads `.md`, `.txt` files, or web pages (via URLs), delegates chunking to the LLM (up to 500 tokens per block, preserving semantic boundaries), computes embedding vectors and stores the chunks in Qdrant. Each chunk is identified by the SHA-256 hash of its text. For URLs, HTML pages are first preprocessed to remove ads, navigation, and other noise elements before being converted to clean Markdown and split into chunks.

**Chat** — an interactive REPL with multi-turn conversation support. Before answering, the agent searches the knowledge base via vector similarity when needed. If no relevant information is found, it says so explicitly rather than making something up. The conversation history is automatically managed to stay within the configured context window: when the history approaches the token threshold it is compacted by the built-in summarizer, which preserves recent turns verbatim and replaces older ones with a concise summary.

**HTML Preprocessing** — the preprocessor cleans HTML pages by removing advertising blocks, navigation headers/footers, related articles, social media widgets, cookie notices, and other noise elements, then converts the result to clean Markdown. The cleaned content is suitable for indexing or direct use.

## Requirements

- Go 1.26.4+
- [Qdrant](https://qdrant.tech/) — vector database
- An API key for any OpenAI-compatible provider (for both the LLM and the embedding model)

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
  qdrant:
    connstr: http://localhost:6333
    collection: documents

log:
  level: info

compact:
  threshold: 0   # 0 = auto (80 % of llm.context)
  save_tail: 6   # keep the last 6 messages verbatim
```

### 4. Index your documents

```bash
# Single file
./bin/go-magnetar indexer -c my-config.yaml -f docs/guide.md

# Entire directory (recursive)
./bin/go-magnetar indexer -c my-config.yaml -d docs/

# From URL
./bin/go-magnetar indexer -c my-config.yaml -u https://example.com/article
```

### 5. Ask questions

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

## Installation

```bash
git clone https://github.com/wmentor/go-magnetar
cd go-magnetar
make build
```

## Configuration reference

| Parameter | Description |
|---|---|
| `llm.base_url` | Endpoint of the OpenAI-compatible API for the chat model |
| `llm.api_key` | API key |
| `llm.model` | Chat model name (e.g. `gpt-4o`) |
| `llm.context` | Token limit for the context passed to the LLM per request |
| `rag.llm.base_url` | Endpoint for the embedding model (may be the same as `llm.base_url`) |
| `rag.llm.api_key` | API key for the embedding model |
| `rag.llm.model` | Embedding model name (e.g. `text-embedding-3-small`) |
| `rag.llm.vector_size` | Vector dimensionality — must match the embedding model |
| `rag.qdrant.connstr` | Qdrant address, e.g. `http://localhost:6333` |
| `rag.qdrant.collection` | Collection name (created automatically if absent) |
| `log.level` | Log level: `debug`, `info`, `warn`, `error` |
| `compact.threshold` | Token count that triggers history summarization. `0` means auto: 80 % of `llm.context` |
| `compact.save_tail` | Number of most-recent messages kept verbatim during summarization. `< 1` summarizes everything |

**Vector sizes for common embedding models:**

| Model | `vector_size` |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-ada-002` | 1536 |
| `text-embedding-3-large` | 3072 |

## Commands

### `indexer` — index documents

```
go-magnetar indexer -c <config> [-f <file>] [-d <directory>] [-u <url>]
```

| Flag | Description |
|---|---|
| `-c` | Path to the config file (default: `~/.go-magnetar.yaml`) |
| `-f` | Path to a single `.md` or `.txt` file to index |
| `-d` | Path to a directory — all `.md` and `.txt` files are processed recursively |
| `-u` | URL to fetch and index |

At least one of `-f`, `-d`, or `-u` is required. If a single file fails during directory indexing, the error is logged and processing continues.

### `agent` — interactive chat

```
go-magnetar agent -c <config>
```

| Flag | Description |
|---|---|
| `-c` | Path to the config file (default: `~/.go-magnetar.yaml`) |

The REPL reads questions from stdin line by line. Press `Ctrl+D` to exit.

#### Chat commands

The following built-in commands are available at the `>` prompt:

| Command | Aliases | Description |
|---|---|---|
| `/help` | `help` | Show the list of available chat commands |
| `/exit` | `exit` | End the session and exit the program |
| `/compact` | — | Immediately compress the conversation history via the summarizer, freeing up context space |
| `/stat` | — | Print context statistics: messages, estimated tokens, bytes, LLM model name, RAG embedding model name and vector size |

Commands are case-insensitive and are processed locally — they are never sent to the LLM.

## Makefile

```bash
make build      # compile -> bin/go-magnetar
make clean      # remove bin/
make run-agent  # build and run the agent with configs/config.yaml
make lint       # go vet ./...
make tidy       # go mod tidy
```

## Architecture

```
cmd/go-magnetar/main.go          — entry point
internal/
  config/config.go               — YAML config loading, slog initialisation
  cmd/
    cmd.go                       — root CLI (kong)
    indexer/cmd.go               — indexer subcommand
    agent/cmd.go                 — agent subcommand
  tools/
    generic/generic.go           — file_read tool
    rag/rag.go                   — rag_save and rag_search tools
    web/fetch.go                 — web_fetch tool (HTML preprocessing, URL fetching)
  agent/
    indexer/indexer.go           — indexer agent, tool-use loop
    chat/agent.go                — chat agent, REPL, tool-use loop
    summarizer/summarizer.go     — history compaction agent
    html/preprocessor.go         — HTML preprocessor agent
```

### Indexing flow

```
indexer -f file.md
  └── LLM: "read and split into chunks"
        ├── tool: file_read(file.md)   -> file contents
        ├── tool: rag_save(chunk_1)    -> uuid -> embed -> qdrant.Upsert
        ├── tool: rag_save(chunk_2)    -> ...
        └── "Saved N chunks"

indexer -u url
  └── LLM: "fetch and split into chunks"
        ├── tool: web_fetch(url)       -> cleaned Markdown
        ├── tool: rag_save(chunk_1)    -> uuid -> embed -> qdrant.Upsert
        ├── tool: rag_save(chunk_2)    -> ...
        └── "Saved N chunks"
```

### Chat flow

```
agent
  └── REPL: reads question from stdin
        └── ask(user_input)
              ├── [if token threshold reached]
              │     └── summarizer.Compact(history)
              │           ├── keep system message
              │           ├── keep last save_tail messages verbatim
              │           └── LLM: summarize older messages -> one summary message
              ├── trimMessages(history) -> fits within context window
              └── LLM: decides whether to search
                    ├── tool: rag_search(query)  -> embed -> qdrant.Query(top-5)
                    └── composes answer from retrieved chunks
```

## Logging

All logs are written to `stderr`. Log levels:

| Level | When |
|---|---|
| `DEBUG` | Tool name and arguments for every tool call; number of messages trimmed/compacted |
| `INFO` | Start/end of file indexing, collection creation, final LLM response; history compaction start |
| `WARN` | Overwriting an already existing chunk in Qdrant |
| `ERROR` | File read errors, embedding failures, Qdrant errors, LLM errors; compaction failure (non-fatal) |

Set `log.level: debug` in the config for verbose output.

## Dependencies

| Package | Purpose |
|---|---|
| [`github.com/alecthomas/kong`](https://github.com/alecthomas/kong) | CLI parser |
| [`github.com/sashabaranov/go-openai`](https://github.com/sashabaranov/go-openai) | OpenAI API client (LLM + embeddings) |
| [`github.com/qdrant/go-client`](https://github.com/qdrant/go-client) | Qdrant client (gRPC) |
| [`github.com/knadh/koanf/v2`](https://github.com/knadh/koanf) | YAML config loading |
| [`github.com/lmittmann/tint`](https://github.com/lmittmann/tint) | Colourised slog handler |
| [`github.com/charmbracelet/glamour`](https://github.com/charmbracelet/glamour) | Markdown rendering in terminal |
| `log/slog` | Structured logging (stdlib) |

## License

See [LICENSE](LICENSE).
