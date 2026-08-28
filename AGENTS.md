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

```
cmd/go-magnetar/main.go          — entry point; blank plugin imports
internal/
  config/config.go               — YAML config loading, printer initialization
  chunk/chunk.go                 — text chunking (UTF-8, paragraph/heading boundaries)
  plugin/
    plugin.go                    — Plugin, Hub, CLIPlugin, AgentHandle interfaces;
                                   State, LLMTool, ChatCommand, ErrExit types
    registry.go                  — global registry; Register, InitAll, Stop,
                                   KongPlugins, LLMTools, ChatCommands, SetRoot, Reset
  plugins/
    rag/plugin.go                — rag_search LLM tool (init → Register)
    web/plugin.go                — web_fetch LLM tool (init → Register)
    generic/plugin.go            — file_read/list/write/exists/system_grep/LLM tools (init → Register)
      indexcmd/plugin.go           — /index chat command plugin
    chatcmd/
      help/plugin.go             — /help  (assembles output from plugin.ChatCommands())
      exit/plugin.go             — /exit  (returns plugin.ErrExit)
      new/plugin.go              — /new   (calls AgentHandle.Reset())
      compact/plugin.go          — /compact (calls AgentHandle.Compact())
      stat/plugin.go             — /stat  (reads AgentHandle.Messages() + Config())
      less/plugin.go             — /less  (view last answer with less)
      write/plugin.go            — /write (write content to file)
      version/plugin.go          — /version (print version)
      readonly/plugin.go         — /readonly (toggle readonly mode)
      fetch/plugin.go            — /fetch (fetch URL content and display/save)
    jira/plugin.go               — jira_task_get LLM tool (init → Register)
  cmd/
    cmd.go                       — root CLI (kong); config load; plugin.InitAll; defer Stop
  tools/
    rag/rag.go                   —     rag_save and rag_search tools — removed rag_save from dispatch
    web/fetch.go                 — web_fetch tool; HTML fetching and cleaning
    generic/generic.go           — file_read, file_list, file_write, file_exists, system_grep, tools
  agent/
    guard/agent.go               — security guard for exec commands
    indexer/indexer.go           — indexer agent (used by /index command)
    chat/agent.go                — chat agent, REPL, tool-use loop, agentHandle adapter
    summarizer/summarizer.go     — history compression agent
    markdown/preprocessor.go     — AI-based Markdown cleaner (webfetch post-processing)
```

### Plugin lifecycle

```
main()
  └── blank imports → each plugin's init() runs
        └── plugin.Register("name", &Plugin{})
  cmd.Execute()
  ├── kong.Parse(cli, ...)             — parses flags; -c resolved here
  ├── config.Load(path)                — loads YAML
  ├── printer.New(verbose)             — initializes printer
  ├── plugin.InitAll(State{Config})
  │     ├── Plugin.Init(s, hub) × N   — tools/commands/goroutines registered
  │     └── hub.start()               — goroutines launched
  ├── defer plugin.Stop()
  └── agent.Run()                      — starts REPL
```

### Data flow: file indexing

```
/index path/to/document.md [-m <message>]
         --> os.ReadFile(filename)
         --> chunk.Split(content, cfg)
               --> splitParagraphs   — paragraph and Markdown heading boundaries
               --> greedy pack       — greedy packing up to MaxSize runes
               --> forceSplit        — for paragraphs longer than MaxSize
         --> for each chunk:
               --> rag.RagSave(chunk, prepend)
                     --> prepend + "\n" + chunk (if prepend set)
                     --> contentUUID(chunk) -> UUID v5 (deterministic)
                     --> embed(chunk)        -> []float32
                     --> qdrant.Upsert(id, vector, payload{text: chunk})
```

### Data flow: URL indexing

```
/index https://example.com/article [-m <message>]
         --> web.WebFetch(url)       — fetch + HTML cleanup -> Markdown
         --> chunk.Split(content, cfg)
         --> (same as for file)
```

### Data flow: URL fetching (via /fetch)

```
/fetch https://example.com [output.md]
         --> web.WebFetch(url)       — fetch + HTML cleanup -> Markdown
         --> if filename specified:  → os.WriteFile(filename, content)
         --> else:                   → display in terminal (with less if available)
```

### Data flow: chat

```
REPL --> user_input
          --> [if token threshold reached]
               --> summarizer.Compact(history)
         --> trimMessages(history) — trimming to fit context window
          --> build toolMap from plugin.LLMTools()
         --> LLM (system prompt + history + user_input + tools)
               --> tool_call dispatched via toolMap[name].Execute(ctx, args)
                     --> rag_search:
                           --> expandQuery (LLM) -> N extra phrasings
                           --> parallel: embed+query for each phrasing
                           --> merge by chunk ID, keep best score
                           --> trim to search.limit
                           --> dedup by cosine similarity
                           --> return joined top-N texts
                     --> web_fetch:  fetch -> HTML clean -> Markdown
                     --> file_*:     sandboxed filesystem ops
         --> output answer to stdout
```

## Chunking (`internal/chunk`)

The `internal/chunk` package implements chunking optimized for RAG:

- **Paragraph boundaries** — splits on blank lines (`\n\n`).
- **Markdown headings** — each ATX heading (`# … ######`) starts a new chunk so section titles stay with their content.
- **Word-boundary snapping** — chunk and overlap boundaries are aligned to word edges; words are never cut in the middle.
- **UTF-8 safe** — all size accounting is in Unicode runes, not bytes. Cyrillic, CJK, emoji all work correctly.
- **Newline normalization** — `\r\n` and `\r` are normalized to `\n` before processing.

## File reading with line range (`internal/tools/generic`)

The `file_read` tool supports reading file contents by line range to avoid loading large files into memory:

| Parameter | Default | Description |
|---|---|---|
| `limit` | `0` | Maximum number of lines to read (`0` = read all lines) |
| `offset` | `0` | Number of lines to skip from the beginning (`0` = start from beginning) |

When both `limit` and `offset` are `0`, the entire file is read using the optimized `ReadFile` path. When either parameter is non-zero, the tool uses `bufio.Scanner` to read line-by-line without loading the entire file into memory, then returns the specified range.

### Supported file formats

The `file_read` tool supports the following file formats:

| Format | Description |
|---|---|
| `.txt` | Plain text files |
| `.md` | Markdown files |
| `.docx` | Microsoft Word documents (via `internal/docx.ReadFile`) |
| `.pdf` | PDF documents (via `internal/pdf.ReadFile`) |
| `.odt` | OpenOffice Writer documents (via `internal/odt.ReadFile`) |

### File preprocessor

The preprocessor supports reading file contents via the `{{file:filename}}` syntax. This is useful for injecting file contents directly into user prompts. The preprocessor automatically detects file type and uses the appropriate reader:

| Format | Description |
|---|---|
| `.txt` | Plain text files (via `os.ReadFile`) |
| `.md` | Markdown files (via `os.ReadFile`) |
| `.docx` | Microsoft Word documents (via `internal/docx.ReadFile`) |
| `.pdf` | PDF documents (via `internal/pdf.ReadFile`) |
| `.odt` | OpenOffice Writer documents (via `internal/odt.ReadFile`) |

## Search replace functionality (`internal/tools/generic`)

| Parameter | Default | Description |
|---|---|---|
| `filename` | — | Path to the file to modify |
| `operations` | — | Array of search-and-replace operations |

Each operation can contain the following fields:

| Field | Type | Description |
|---|---|---|
| `search` | string | Regex pattern to search for (required) |
| `replace` | string | Replacement string (supports $1, $2, etc.) (required) |
| `before` | string | Optional context before the match (plain text, not regex) |
| `after` | string | Optional context after the match (plain text, not regex) |
| `max_len` | integer | Maximum file length allowed for this operation (0 = unlimited) |

The tool validates regex patterns, checks context constraints, verifies file size limits, and applies all successful operations sequentially. It returns success status, modified content, and a list of any errors encountered.

## Plugin System

go-magnetar uses a plugin architecture modelled after `database/sql` drivers.

### Adding a new LLM tool

1. Create `internal/plugins/mytool/plugin.go`
2. Call `plugin.Register("mytool", &Plugin{})` in `init()`
3. Implement `Init(s plugin.State, hub plugin.Hub) error` — call `hub.RegisterTool(...)`
4. Add a blank import in `cmd/go-magnetar/main.go`

### Adding a new chat command

Same pattern but call `hub.RegisterChatCommand(plugin.ChatCommand{...})` in `Init`.
Return `plugin.ErrExit` from `Execute` to signal the REPL to quit.

### Adding a text preprocessor

Text preprocessors allow plugins to transform user input before it's processed by the agent (commands or LLM). 
Preprocessors are applied sequentially in registration order.

1. Create a preprocessor function with signature: `func(ctx context.Context, text string) (string, error)`
2. Call `hub.RegisterPreprocessor(fn)` in your plugin's `Init()` method
3. Preprocessors run after user input is displayed but before command matching or LLM processing

Example:
```go
func (p *MyPlugin) Init(s plugin.State, hub plugin.Hub) error {
    hub.RegisterPreprocessor(func(ctx context.Context, text string) (string, error) {
        // Transform text (e.g., normalize, expand abbreviations, etc.)
        return strings.ToUpper(text), nil
    })
    return nil
}
```

### Key interfaces

```go
type Plugin interface {
    Init(s State, hub Hub) error
}

type Hub interface {
    RegisterTool(tool LLMTool)
    RegisterChatCommand(cmd ChatCommand)
    RegisterCLICommand(cmd any)
    RegisterPreprocessor(fn PreprocessorFunc)
    Go(f func(ctx context.Context))  // background goroutine; started after all Init calls
    Stop()
}

type AgentHandle interface {
    Messages() []openai.ChatCompletionMessage
    SetMessages([]openai.ChatCompletionMessage)
    Config() *config.Config
    Compact() error
    Reset()
}
```

## Logging

All logs are written to `stdout` in text format using the `internal/printer` package.

Set `verbose: true` in the config for verbose output.

### Logging in Tools

For logging within tools, use `printer.ToolCall()` with the appropriate icon:

- **Success normal case**: `printer.IconTool`
- **Error case**: `printer.IconError`

**Signature**: `printer.ToolCall(icon, message, param1, value1, param2, value2, ...)`

Example:
```go
printer.ToolCall(printer.IconTool, "rag_search", "query", query, "results", len(results))
printer.ToolCall(printer.IconError, "rag_save failed", "id", id, "err", err)
```

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
