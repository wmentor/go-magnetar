# Architecture

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
