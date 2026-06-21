# go-magnetar

A utility that combines two AI agents: a **document indexer** for RAG and a **chat agent** that answers questions strictly based on indexed data.

## Requirements

- Go 1.26.4+
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

Copy the example and fill in actual values:

```bash
cp configs/config.yaml my-config.yaml
```

```yaml
llm:
  base_url: https://api.openai.com/v1   # OpenAI-compatible endpoint
  api_key: YOUR_API_KEY
  model: gpt-4o
  context: 128000                        # token limit of the context window

rag:
  llm:
    base_url: https://api.openai.com/v1
    api_key: YOUR_API_KEY
    model: text-embedding-3-small        # embedding model
    vector_size: 1536                    # vector dimensionality of the model
  chunk:
    size: 512                            # maximum chunk size in runes (default: 512)
    overlap: 64                          # overlap between adjacent chunks in runes (default: 64)
  search:
    limit: 10                            # maximum number of results per query (default: 10)
    threshold: 0.40                      # minimum cosine similarity threshold, 0–1 (default: 0.40)
  qdrant:
    connstr: http://localhost:6333       # Qdrant address (REST port; gRPC 6334 is used automatically)
    collection: documents                # collection name (created automatically if missing)

log:
  level: info                            # debug | info | warn | error

compact:
  threshold: 0    # token threshold for history compression; 0 = auto (80% of llm.context)
  save_tail: 6    # number of trailing messages kept unchanged

webfetch:
  base_url: https://api.openai.com/v1
  api_key: YOUR_API_KEY
  model: gpt-4o
  context: 128000
```

> If the `webfetch` block is specified, the listed model parameters are used to clean HTML content obtained from web pages.

> `vector_size` must match the dimensionality of the chosen embedding model.
> For `text-embedding-3-small` — 1536, for `text-embedding-ada-002` — 1536, for `text-embedding-3-large` — 3072.

### Parameters `rag.chunk`

| Parameter | Default | Description |
|---|---|---|
| `rag.chunk.size` | `512` | Maximum chunk size in Unicode runes |
| `rag.chunk.overlap` | `64` | Overlap between adjacent chunks in runes (~12.5 %) |

### Parameters `rag.search`

| Parameter | Default | Description |
|---|---|---|
| `rag.search.limit` | `10` | Maximum number of results returned by Qdrant per query |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; results below the threshold are discarded |

### Parameters `compact`

| Parameter | Description |
|---|---|
| `compact.threshold` | Token count threshold that triggers history compression. `0` = auto: 80 % of `llm.context` |
| `compact.save_tail` | Number of trailing messages kept unchanged. `< 1` — compress all messages |

## Indexer Agent

Reads `.md` and `.txt` files or web pages (by URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores them in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content — re-indexing the same file does not create duplicates.

### Index a single file

```bash
./bin/go-magnetar indexer -c my-config.yaml -f path/to/document.md
```

### Index a URL

```bash
./bin/go-magnetar indexer -c my-config.yaml --url https://example.com/article
```

> If the `webfetch` block is configured in the config, the HTML page is cleaned of ads, navigation, and other noise by an AI agent before conversion to Markdown and indexing.

### Duplicate handling

Each chunk's ID is UUID v5 derived from its text (`uuid.NewSHA1`). Repeated calls to `rag_save` with the same content perform `Upsert` to the same ID — the existing point is overwritten, not duplicated.

### Indexer tools

| Tool | Signature | Description |
|---|---|---|
| `rag_save` | `(content: string) -> bool` | Saves a fragment to Qdrant |
| `web_fetch` | `(url: string) -> string` | Fetches and cleans a web page, returns Markdown |

## Chat Agent

Interactive REPL. Supports multi-turn conversation — conversation history is maintained throughout the session.

```bash
./bin/go-magnetar agent -c my-config.yaml
# or via Makefile (uses configs/config.yaml):
make run-agent
```

### Working with the agent

```
> What is go-magnetar?
go-magnetar is a utility that combines two AI agents...

> What commands does it support?
It supports two commands: indexer and agent...

> ^D
```

Exit — `Ctrl+D` (EOF) or `/exit` command. Empty lines are ignored.

### Built-in chat commands

| Command | Aliases | Description |
|---|---|---|
| `/help` | `help` | Show the list of available commands |
| `/exit` | `exit` | End the session and exit the program |
| `/compact` | — | Immediately compress history via summarizer, without waiting for automatic threshold |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: number of messages, estimated tokens, size in bytes, LLM model name, RAG model name, and vector size |

Commands are processed locally in the `handleCommand` method (`internal/agent/chat/agent.go`) before user input is sent to the LLM; they are not added to the message history. Comparison is case-insensitive.

The help text is stored in the `helpText` constant in the same file. Data for `/stat` is taken directly from `a.cfg`: `cfg.LLM.Model`, `cfg.RAG.LLM.Model`, `cfg.RAG.LLM.VectorSize`.

### Search strategy

The agent **always** first calls `rag_search`, even if it believes it already knows the answer. If `rag_search` returns relevant results, the answer is formed exclusively based on those results, `web_fetch` is not called. `web_fetch` is used only as a fallback: when `rag_search` returns no relevant results and the user needs external or up-to-date information. If neither tool provides a result, the agent explicitly states this.

### Chat agent tools

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string) -> string` | Reads file contents from the filesystem |
| `file_list` | `(options: object) -> []string` | Recursively lists files in the current directory |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem |
| `rag_search` | `(query: string) -> string` | Returns top-N relevant fragments from Qdrant (N is set by `rag.search.limit`) |
| `web_fetch` | `(url: string) -> string` | Fetches and cleans a web page (fallback if RAG returns no results) |

## Architecture

```
cmd/go-magnetar/main.go          — entry point
internal/
  config/config.go               — YAML config loading, slog initialization
  chunk/chunk.go                 — text chunking (UTF-8, paragraph/heading boundaries)
  cmd/
    cmd.go                       — root CLI (kong)
    indexer/cmd.go               — indexer subcommand
    agent/cmd.go                 — agent subcommand
  tools/
    rag/rag.go                   — rag_save and rag_search tools; Qdrant connection
    web/fetch.go                 — web_fetch tool; HTML fetching and cleaning
    generic/generic.go           — file_read, file_list, file_write tools
  agent/
    indexer/indexer.go           — indexer agent
    chat/agent.go                — chat agent, REPL, tool-use loop
    summarizer/summarizer.go     — history compression agent
```

### Data flow: file indexing

```
CLI --> IndexFile(filename)
         --> os.ReadFile(filename)
         --> chunk.Split(content, cfg)
               --> splitParagraphs   — paragraph and Markdown heading boundaries
               --> greedy pack       — greedy packing up to MaxSize runes
               --> forceSplit        — for paragraphs longer than MaxSize
         --> for each chunk:
               --> rag.RagSave(chunk)
                     --> contentUUID(chunk) -> UUID v5 (deterministic)
                     --> embed(chunk)        -> []float32
                     --> qdrant.Upsert(id, vector, payload{text: chunk})
```

### Data flow: URL indexing

```
CLI --> IndexURL(url)
         --> web.WebFetch(url)       — fetch + HTML cleanup -> Markdown
         --> chunk.Split(content, cfg)
         --> (same as for file)
```

### Data flow: chat

```
CLI --> Run() --> REPL
         --> ask(user_input)
                --> [if token threshold reached]
                      --> summarizer.Compact(history)
                            --> system message is preserved
                            --> last save_tail messages are preserved
                            --> LLM: compress older messages -> one summary message
                --> trimMessages(history) — trimming to fit context window
                --> LLM (system prompt + history + user_input)
                      --> tool_call: rag_search(query)
                            --> embed(query) -> []float32
                            --> qdrant.Query(vector, limit=N, score_threshold=T)
                            --> return top-N texts
                      --> [if rag_search returned results]
                            --> LLM composes answer, web_fetch is not called
                      --> [if rag_search empty]
                            --> tool_call: web_fetch(url) -> Markdown
                            --> LLM composes answer
                --> output answer to stdout
```

## Chunking (`internal/chunk`)

The `internal/chunk` package implements chunking optimized for RAG:

- **Paragraph boundaries** — splits on blank lines (`\n\n`).
- **Markdown headings** — each ATX heading (`# … ######`) starts a new chunk so section titles stay with their content.
- **Word-boundary snapping** — chunk and overlap boundaries are aligned to word edges; words are never cut in the middle.
- **UTF-8 safe** — all size accounting is in Unicode runes, not bytes. Cyrillic, CJK, emoji all work correctly.
- **Newline normalization** — `\r\n` and `\r` are normalized to `\n` before processing.

## Logging

All logs are written to `stderr` in `slog` text format. Levels:

| Level | When |
|---|---|
| `DEBUG` | Tool call details (name, arguments); search scores and result previews; number of trimmed/compressed messages |
| `INFO` | Start/end of file indexing, collection creation, history compression start |
| `WARN` | Overwriting an existing chunk in Qdrant |
| `ERROR` | File read errors, embedding/Qdrant failures, LLM errors; compression failure (non-fatal) |

Set `log.level: debug` in the config for verbose output.

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/alecthomas/kong` | CLI parser |
| `github.com/sashabaranov/go-openai` | OpenAI API client (LLM + embeddings) |
| `github.com/qdrant/go-client` | Qdrant client (gRPC) |
| `github.com/google/uuid` | UUID v5 for deterministic chunk IDs |
| `github.com/knadh/koanf/v2` | YAML config loading |
| `github.com/lmittmann/tint` | Colorized slog handler |
| `github.com/charmbracelet/glamour` | Markdown rendering in terminal |
| `log/slog` | Structured logging (stdlib) |
