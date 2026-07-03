# go-magnetar

A knowledge base tool built on RAG (Retrieval-Augmented Generation). Combines a **chat agent** with an integrated **index command** for document ingestion — all in a unified interactive REPL.

## How it works

**Indexing** — reads `.md`, `.txt` files or web pages (via URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores each chunk in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content, making re-indexing idempotent: the same chunk is never stored twice.

**Chat** — an interactive REPL with multi-turn conversation support. The agent always tries `rag_search` first; if the knowledge base returns relevant results, the answer is based exclusively on those results. `web_fetch` is used only as a fallback when the knowledge base has no relevant information and the user needs external or up-to-date data. If neither source has an answer, the agent says so explicitly. Conversation history is automatically managed to stay within the configured context window: when history approaches the token threshold it is compacted by the built-in summarizer, which replaces older turns with a concise summary while keeping the most recent ones verbatim.

**Indexing via chat** — the `/index` command (alias `/i`) replaces the separate indexer CLI subcommand. Simply type `/index <path|url>` in the REPL to index documents directly.

**Fetch content** — the `/fetch` command (alias `/f`) retrieves content from URLs, cleans HTML, and displays it in the terminal (using `less` if available) or saves it to a file.

**Search tool call limit** — to prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is reached, an error message is sent to the LLM.

**HTML Preprocessing** — when `webfetch.*` parameters are configured, web pages fetched via `web_fetch` are cleaned of ads, navigation, cookie banners, and other noise using an AI agent before being converted to Markdown and indexed or returned to the agent. Confluence URLs are also handled via the `confluence` block to fetch pages by ID, JIRA issues via the `jira` block, GitHub repositories via the `github` block to fetch repository information, files, and directory trees, and GitLab merge requests via the `gitlab` block to fetch MR details and file changes.

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
  disable: false
  llm:
    base_url: https://api.openai.com/v1
    api_key: sk-...
    model: text-embedding-3-small
    vector_size: 1536
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

### 4. Index your documents

```bash
./bin/go-magnetar -c my-config.yaml agent
> /index docs/guide.md

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
| `rag.search.multi_query` | `2` | Number of extra query reformulations generated by the LLM to improve recall. `0` disables multi-query |
| `rag.search.dedup_threshold` | `0.95` | Cosine similarity above which two result chunks are considered near-duplicates; the lower-scoring one is dropped. `0` disables deduplication |
| `rag.qdrant.connstr` | — | Qdrant address, e.g. `http://localhost:6333` |
| `rag.qdrant.collection` | — | Collection name (created automatically if absent) |
| `webfetch.base_url` | — | Endpoint of the OpenAI-compatible API for web page preprocessing model |
| `webfetch.api_key` | — | API key for web page preprocessing model |
| `webfetch.model` | — | Model name for web page preprocessing (e.g. `gpt-4o`) |
| `webfetch.context` | — | Token limit of the model's context window for web preprocessing |
| `confluence.base_url` | — | Confluence instance base URL (e.g. `https://your-domain.atlassian.net`) |
| `confluence.api_key` | — | Confluence API key for fetching pages |
| `jira.base_url` | — | JIRA instance base URL |
| `jira.api_key` | — | JIRA API key for fetching issues |
| `gitlab.base_url` | — | GitLab instance base URL |
| `gitlab.api_key` | — | GitLab API key for fetching merge requests and file changes |
| `github.base_url` | — | GitHub API base URL (e.g. `https://api.github.com`) |
| `github.api_key` | — | GitHub access token for fetching repositories, files, and trees |
| `verbose` | `true` | Enable verbose output (tool calls, debug messages) |
| `compact.threshold` | `0` | Token count that triggers history summarization. `0` = auto: 80 % of `llm.context` |
| `compact.save_tail` | `6` | Number of most-recent messages kept verbatim during summarization |

**Vector sizes for common embedding models:**

| Model | `vector_size` |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-ada-002` | 1536 |
| `text-embedding-3-large` | 3072 |

## Commands

The `-c`/`--config` flag is global and must be placed **before** the command:

```
go-magnetar [-c <config>] <command> [flags]
```

If `-c` is omitted, `~/.go-magnetar.yaml` is used. The flag can also be set via the `GO_MAGNETAR_CONFIG` environment variable.

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
| `/compact` | — | Immediately compress the conversation history via the summarizer |
| `/new` | — | Start a new session and clear conversation history |
| `/stat` | — | Print context statistics: messages, estimated tokens, bytes, LLM model, RAG model, vector size |
| `/index` | `/i` | Index file or URL into RAG knowledge base (auto-detects URL vs file) |
| `/idxtab` | — | Index multiple files/URLs from a JSON lines file (one per line, format: `{"source":"path\|url","message":"text"}`) |
| `/write` | `/w` | Write content to a file |
| `/readonly` | — | Toggle read-only mode (blocks all modification operations) |
| `/fetch` | `/f` | Fetch content from a URL, optionally save to file |

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

```
cmd/go-magnetar/main.go          — entry point; blank plugin imports
internal/
  config/config.go               — YAML config loading, printer initialization
  chunk/chunk.go                 — text chunking (paragraph/heading boundaries, UTF-8 safe)
  plugin/
    plugin.go                    — Plugin, Hub, CLIPlugin, AgentHandle interfaces;
                                   State, LLMTool, ChatCommand types
    registry.go                  — global registry; Register, InitAll, Stop,
                                   KongPlugins, LLMTools, ChatCommands, SetRoot
  plugins/
    rag/plugin.go                — rag_search LLM tool plugin
    web/plugin.go                — web_fetch LLM tool plugin
    generic/plugin.go            — file_read/list/write/exists/system_grep LLM tool plugin
    indexcmd/plugin.go           — /index chat command plugin
    chatcmd/
      help/plugin.go             — /help chat command plugin
      exit/plugin.go             — /exit chat command plugin
      new/plugin.go              — /new chat command plugin
      compact/plugin.go          — /compact chat command plugin
      stat/plugin.go             — /stat chat command plugin
      less/plugin.go             — /less chat command plugin
      write/plugin.go            — /write chat command plugin
      version/plugin.go          — /version chat command plugin
      readonly/plugin.go         — /readonly chat command plugin
      fetch/plugin.go            — /fetch chat command plugin
  cmd/
    cmd.go                       — root CLI (kong); config load; plugin.InitAll; defer Stop
  tools/
    rag/rag.go                   —     rag_save and rag_search tools — removed rag_save from dispatch
    web/fetch.go                 — web_fetch tool (HTML preprocessing, URL fetching)
    generic/generic.go           — file_read, file_list, file_write, file_exists, system_grep, exec with guard
    agent/
      guard/agent.go             — security guard for exec commands
  agent/
    indexer/indexer.go           — indexer agent (used by /index command)
    chat/agent.go                — chat agent, REPL, tool-use loop
    summarizer/summarizer.go     — history compaction agent
    markdown/preprocessor.go     — AI-based Markdown cleaner (webfetch post-processing)
```

### Plugin lifecycle

```
main()
  └── blank imports trigger init() in each plugin package
        └── plugin.Register("name", &Plugin{})   — records plugin in global registry

cmd.Execute()
  ├── kong.Parse(...)          — parses os.Args; resolves -c/--config flag
  ├── config.Load(path)        — loads YAML config
  ├── printer.New(verbose)     — initializes printer
  ├── plugin.InitAll(State{Config: cfg})
  │     ├── Plugin.Init(s, hub) for every registered plugin  — tools/commands registered
  │     └── hub.start()        — launches all goroutines queued via Hub.Go
  ├── defer plugin.Stop()      — cancels goroutine contexts, waits for exit
  └── agent.Run()              — starts REPL
```

### Built-in plugins

| Plugin | Type | Description |
|---|---|---|
| `rag` | LLM tool | `rag_search` tool — searches the knowledge base for relevant information |
| `web` | LLM tool | `web_fetch` tool — fetches and cleans web pages, Confluence pages, JIRA issues, GitLab merge requests, GitHub repositories |
| `generic` | LLM tools | `file_read`, `file_list`, `file_write`, `file_exists`, `system_grep`, tools |
| `indexcmd` | Chat command | `/index` command for document indexing |
| `chatcmd/help` | Chat command | `/help` command |
| `chatcmd/exit` | Chat command | `/exit` command |
| `chatcmd/new` | Chat command | `/new` command |
| `chatcmd/compact` | Chat command | `/compact` command |
| `chatcmd/stat` | Chat command | `/stat` command |
| `chatcmd/less` | Chat command | `/less` command |
| `chatcmd/write` | Chat command | `/write` command |
| `chatcmd/version` | Chat command | `/version` command |
| `chatcmd/readonly` | Chat command | `/readonly` command |
| `chatcmd/fetch` | Chat command | `/fetch` command — fetch URL content and display/save |
| `guard` | LLM tool | Security guard for exec commands — analyzes shell commands before execution |
| `jira` | LLM tool | `jira_task_get` tool — fetches JIRA issues by issue key |

### Indexing flow

```
/index path/to/file.md [-m <message>]
  └── os.ReadFile(file.md)
        └── chunk.Split(content)       — paragraph/heading-aware, word-boundary snapping
              └── for each chunk:
                    └── rag.RagSave(chunk, prepend)
                          ├── prepend + "\n" + chunk (if prepend set)
                          ├── contentUUID(chunk) -> UUID v5 (deterministic, idempotent)
                          ├── embed(chunk)        -> []float32
                          └── qdrant.Upsert(id, vector, payload{text: chunk})

/index https://example.com [-m <message>]
  └── web.WebFetch(url)               — fetch + HTML cleanup -> Markdown
        └── chunk.Split(content)
              └── (same as above)
```

### Fetching flow

```
/fetch https://example.com [output.md]
  └── web.WebFetch(url)               — fetch + HTML cleanup -> Markdown
        └── if filename:  → os.WriteFile(filename, content)
        └── else:        → display in terminal (with less if available)
```

### Chat flow

```
agent
  └── REPL: reads question from stdin
        ├── handleCommand(line)  — matched against plugin.ChatCommands()
        └── ask(user_input)
              ├── [if token threshold reached]
              │     └── summarizer.Compact(history)
              ├── trimMessages(history) -> fits within context window
              └── LLM (system prompt + history + user_input)
                    │   tools built from plugin.LLMTools()
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

Four parameters control retrieval quality:

| Parameter | Default | Effect |
|---|---|---|
| `rag.search.limit` | `10` | How many candidate chunks Qdrant returns per query |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; lower = more results but more noise |
| `rag.search.multi_query` | `2` | Extra query reformulations generated by the LLM. Each adds one embedding call + one Qdrant call, run in parallel. Improves recall for queries that match different phrasings in the index |
| `rag.search.dedup_threshold` | `0.95` | Near-duplicate suppression: if two returned chunks have cosine similarity ≥ this value, the lower-scoring one is dropped. Keeps context window usage efficient |

### Recommendations

For a dense, single-topic knowledge base (technical docs):
- `threshold: 0.40–0.45`, `multi_query: 1–2`, `dedup_threshold: 0.95`

For mixed-topic or large bases:
- `threshold: 0.25–0.35`, `multi_query: 2–3`, `dedup_threshold: 0.92`

To disable the new features entirely (single-query, no dedup):
- `multi_query: 0`, `dedup_threshold: 0`

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

### Command safety guard

All commands executed via the `exec` tool are analyzed by a built-in security guard before execution. The guard uses an LLM to evaluate whether a command is safe based on several criteria:

- **Destructive commands**: `rm -rf /`, `sudo`, `mkfs`, `dd` with `/dev/`, `fdisk`, `chmod 777`
- **Shell pipes**: `| bash`, `| sh`, `| zsh`
- **Git modifications**: `git commit`, `push`, `rebase`, `pull`, `cherry-pick`, `reset`, `stash`, `clean`, `reflog`, `clone`
- **System-level operations**: `su`, `chmod`, `chown`, `dscl`, `fdisk`, `format`, `dseditgroup`, `brew`, `dpkg`, `apt`, `cargo`, `rpm`, `npm`, `apt-get`, `groupadd`, `usermod`, `gpasswd`, `useradd`, `adduser`, `userdel`, `deluser`, `passwd`, `systemctl`, `sysadminctl`
- **Environment variable filtering**: Sensitive environment variables containing patterns like `PASS`, `SEC`, `CRED`, `TOKEN`, `KEY`, `AUTH`, `PWD`, `CERT`, `SIGN`, `SALT`, `BEARER`, `AMQP`, `CONNECT` are filtered out before command execution
- **Untrusted script execution**: `curl ... | bash`, `wget ... | sh`

When the system is in read-only mode, **NO modification operations are allowed** — all such commands are automatically rejected.

### Read-only mode

The read-only mode can be toggled via the `/readonly` chat command. In this mode:
- File writes (`file_write`) are blocked
- Command execution that modifies files or system state is blocked
- Only read-only operations are permitted

No output is ever displayed from blocked commands — they return an error message instead.

### Root user prevention

The application **cannot be run as root user**. If the current user is `root` (username or UID 0), the application prints an error message and exits immediately to prevent accidental system-wide modifications.

## License

See [LICENSE](LICENSE).