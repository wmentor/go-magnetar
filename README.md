# go-magnetar

A knowledge base tool built on RAG (Retrieval-Augmented Generation). Combines two agents: an **indexer** that splits documents into semantic chunks and stores them in a vector database, and a **chat agent** that answers questions strictly based on the indexed data — no hallucinations, no guessing.

## How it works

**Indexing** — reads `.md`, `.txt` files or web pages (via URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores each chunk in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content, making re-indexing idempotent: the same chunk is never stored twice.

**Chat** — an interactive REPL with multi-turn conversation support. The agent always tries `rag_search` first; if the knowledge base returns relevant results, the answer is based exclusively on those results. `web_fetch` is used only as a fallback when the knowledge base has no relevant information and the user needs external or up-to-date data. If neither source has an answer, the agent says so explicitly. Conversation history is automatically managed to stay within the configured context window: when history approaches the token threshold it is compacted by the built-in summarizer, which replaces older turns with a concise summary while keeping the most recent ones verbatim.

**Search tool call limit** — to prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is reached, an error message is sent to the LLM to reflect the situation.

**HTML Preprocessing** — when `webfetch.*` parameters are configured, web pages fetched via `web_fetch` are cleaned of ads, navigation, cookie banners, and other noise using an AI agent before being converted to Markdown and indexed or returned to the agent.

## Requirements

- Go 1.26.4+
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

### 4. Index your documents

```bash
# Single file
./bin/go-magnetar indexer -c my-config.yaml -f docs/guide.md

# From URL
./bin/go-magnetar indexer -c my-config.yaml --url https://example.com/article
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

## Command history

The chat agent maintains command history in `~/.go-magnetar-history.json`. Use **↑/↓** arrows to navigate through previous commands. History is persisted across sessions and limited to 200 entries.

## Configuration reference

| Parameter | Default | Description |
|---|---|---|
| `llm.base_url` | — | Endpoint of the OpenAI-compatible API for the chat model |
| `llm.api_key` | — | API key for the chat model |
| `llm.model` | — | Chat model name (e.g. `gpt-4o`) |
| `llm.context` | — | Token limit of the model's context window |
| `rag.llm.base_url` | — | Endpoint for the embedding model |
| `rag.llm.api_key` | — | API key for the embedding model |
| `rag.llm.model` | — | Embedding model name (e.g. `text-embedding-3-small`) |
| `rag.llm.vector_size` | — | Vector dimensionality — must match the embedding model |
| `rag.chunk.size` | `512` | Maximum chunk size in Unicode runes |
| `rag.chunk.overlap` | `64` | Overlap between adjacent chunks in runes (~12.5 %) |
| `rag.search.limit` | `10` | Maximum number of results returned per query |
| `rag.search.threshold` | `0.40` | Minimum cosine-similarity score; results below this are discarded |
| `rag.qdrant.connstr` | — | Qdrant address, e.g. `http://localhost:6333` |
| `rag.qdrant.collection` | — | Collection name (created automatically if absent) |
| `log.level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `compact.threshold` | `0` | Token count that triggers history summarization. `0` = auto: 80 % of `llm.context` |
| `compact.save_tail` | `6` | Number of most-recent messages kept verbatim during summarization |
| `webfetch.base_url` | — | Endpoint of the OpenAI-compatible API for web page preprocessing model |
| `webfetch.api_key` | — | API key for web page preprocessing model |
| `webfetch.model` | — | Model name for web page preprocessing (e.g. `gpt-4o`) |
| `webfetch.context` | — | Token limit of the model's context window for web preprocessing |

**Vector sizes for common embedding models:**

| Model | `vector_size` |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-ada-002` | 1536 |
| `text-embedding-3-large` | 3072 |

## Commands

### `indexer` — index documents

```
go-magnetar indexer -c <config> [-f <file>] [--url <url>] [-m <message>]
```

| Flag | Description |
|---|---|
| `-c` | Path to the config file |
| `-f` | Path to a single `.md` or `.txt` file to index |
| `--url` | URL to fetch and index |
| `-m` | Message to prepend to each chunk for improved search |

Specify either `-f` or `--url`. If a file fails to read, an error is logged.

### `agent` — interactive chat

```
go-magnetar agent -c <config>
```

| Flag | Description |
|---|---|
| `-c` | Path to the config file |

The REPL reads questions from stdin. Press `Ctrl+D` to exit.

#### Chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `help` | Show the list of available chat commands |
| `/exit` | `exit` | End the session and exit the program |
| `/compact` | — | Immediately compress the conversation history via the summarizer |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: messages, estimated tokens, bytes, LLM model, RAG model, vector size |

Commands are case-insensitive and processed locally — never sent to the LLM.

### Command history

The REPL supports command history navigation with arrow keys:
- **↑** — previous command
- **↓** — next command

History is persistently stored in `~/.go-magnetar-history.json` and limited to 200 entries. On startup, history is loaded and available for navigation; no command is pre-filled in the input field.

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
  chunk/chunk.go                 — text chunking (paragraph/heading boundaries, UTF-8 safe)
  cmd/
    cmd.go                       — root CLI (kong)
    indexer/cmd.go               — indexer subcommand
    agent/cmd.go                 — agent subcommand
  tools/
    rag/rag.go                   — rag_save and rag_search tools; Qdrant connection
    web/fetch.go                 — web_fetch tool (HTML preprocessing, URL fetching)
    generic/generic.go           — file_read, file_list, file_write tools
  agent/
    indexer/indexer.go           — indexer agent
    chat/agent.go                — chat agent, REPL, tool-use loop
    summarizer/summarizer.go     — history compaction agent
```

### Indexing flow

```
indexer -f file.md [-m <message>]
  └── os.ReadFile(file.md)
        └── chunk.Split(content)       — paragraph/heading-aware, word-boundary snapping
              └── for each chunk:
                    └── rag.RagSave(chunk, prepend)
                          ├── prepend + "\n" + chunk (if prepend set)
                          ├── contentUUID(chunk) -> UUID v5 (deterministic, idempotent)
                          ├── embed(chunk)        -> []float32
                          └── qdrant.Upsert(id, vector, payload{text: chunk})

indexer --url https://example.com [-m <message>]
  └── web.WebFetch(url)               — fetch + HTML cleanup -> Markdown
        └── chunk.Split(content)
              └── (same as above)
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
              └── LLM (system prompt + history + user_input)
                    ├── tool: rag_search(query)
                    │         ├── embed(query) -> []float32
                    │         ├── qdrant.Query(vector, limit=N, score_threshold=T)
                    │         └── return top-N texts joined by "---"
                    ├── [if rag_search returned results] -> compose answer, skip web_fetch
                    └── [if no results] -> tool: web_fetch(url) -> compose answer
```

## Chunking

The `internal/chunk` package splits text into overlapping segments optimised for RAG:

- **Paragraph boundaries** — splits on blank lines (`\n\n`).
- **Markdown heading boundaries** — each ATX heading (`# … ######`) starts a new chunk so section titles stay with their content.
- **Word-boundary snapping** — chunk and overlap boundaries are aligned to word edges; no word is ever cut mid-character.
- **UTF-8 safe** — all size accounting is in Unicode runes, not bytes. Cyrillic, CJK, emoji all work correctly.
- **Idempotent storage** — chunk IDs are UUID v5 derived from content; re-indexing the same file does not create duplicates.

Default values: `size = 512` runes, `overlap = 64` runes (~12.5 %).

## Search tuning

Two parameters control retrieval quality:

| Parameter | Default | Effect |
|---|---|---|
| `rag.search.limit` | `10` | How many candidate chunks Qdrant returns |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; lower = more results but more noise |

For a dense, single-topic knowledge base (technical docs) `threshold: 0.40–0.45` works well. For mixed-topic bases lower it to `0.25–0.35`.

## Logging

All logs are written to `stderr`. Log levels:

| Level | When |
|---|---|
| `DEBUG` | Tool name and arguments for every call; search scores and previews; trimmed message counts |
| `INFO` | Start/end of file indexing, collection creation, history compaction |
| `WARN` | Overwriting an already existing chunk in Qdrant |
| `ERROR` | File read errors, embedding failures, Qdrant errors, LLM errors; compaction failure (non-fatal) |

Set `log.level: debug` in the config for verbose output.

## Dependencies

| Package | Purpose |
|---|---|
| [`github.com/alecthomas/kong`](https://github.com/alecthomas/kong) | CLI parser |
| [`github.com/sashabaranov/go-openai`](https://github.com/sashabaranov/go-openai) | OpenAI API client (LLM + embeddings) |
| [`github.com/qdrant/go-client`](https://github.com/qdrant/go-client) | Qdrant client (gRPC) |
| [`github.com/google/uuid`](https://github.com/google/uuid) | UUID v5 for deterministic chunk IDs |
| [`github.com/knadh/koanf/v2`](https://github.com/knadh/koanf) | YAML config loading |
| [`github.com/lmittmann/tint`](https://github.com/lmittmann/tint) | Colourised slog handler |
| [`github.com/charmbracelet/glamour`](https://github.com/charmbracelet/glamour) | Markdown rendering in terminal |
| `log/slog` | Structured logging (stdlib) |

## License

See [LICENSE](LICENSE).
