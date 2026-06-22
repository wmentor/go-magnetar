# Plugin Architecture

This document describes the plugin system design and migration plan for go-magnetar.

## Motivation

The current codebase has three hardcoded extension points:

| Extension point | Current implementation | Problem |
|---|---|---|
| CLI subcommands | `indexer`, `agent` hardwired in `CLI` struct via kong tags | Adding a subcommand requires editing `cmd.go` |
| LLM tools | `switch name { case "rag_search": ... }` in `ask()` | Adding a tool requires editing `agent.go` |
| Chat commands | `handleCommand()` switch on strings | Adding `/foo` requires editing `agent.go` |

The plugin system replaces all three with a single registry-based approach modelled
after the `database/sql` driver pattern.

## Design Principles

1. **Minimal plugin interface.** A plugin implements only `Init(State, Hub) error`.
   Everything else — tools, CLI commands, chat commands, background goroutines —
   is registered by the plugin itself through the `Hub`.
2. **Global registry, `init()`-based.** Each plugin package calls `plugin.Register`
   in its `init()` function. `main.go` activates plugins via blank imports, exactly
   like `database/sql` drivers.
3. **Three-phase lifecycle.**
   - `Register` — at `init()` time, before config is loaded; records the plugin.
   - `InitAll` — in `main()`, after config is loaded; calls `Init` on every plugin
     in registration order. Goroutines queued via `Hub.Go` are **not started yet**.
   - `Start` — called by `InitAll` after all plugins have initialised; launches all
     queued goroutines. This guarantees that every goroutine sees a fully
     initialised system.
4. **Hub owns the lifecycle.** The `Hub` is the single point of contact between a
   plugin and the rest of the system. It accepts registrations during `Init` and
   manages the lifecycle of background goroutines.
   `Hub.Stop()` signals all goroutines to stop and waits for them to exit.
5. **Static linking.** Plugins are regular Go packages compiled into the binary.
   No `.so` files.

---

## Plugin Interface

```go
// internal/plugin/plugin.go

// Plugin is the only interface a plugin package must implement.
type Plugin interface {
    // Init is called once after the config is loaded.
    // The plugin registers its contributions (tools, commands, goroutines)
    // through hub and initialises any internal state from s.
    Init(s State, hub Hub) error
}

// State carries shared infrastructure injected into every plugin at Init time.
// Named State (not Context) to avoid confusion with context.Context.
type State struct {
    Config *config.Config
    Root   *os.Root     // sandboxed filesystem; nil in the indexer context
    Log    *slog.Logger
}

// Hub is the registration and lifecycle interface passed to Plugin.Init.
// A plugin calls the Register* methods to contribute functionality,
// and Go to start managed background goroutines.
type Hub interface {
    // RegisterTool adds an LLM tool to the agent tool-use loop.
    RegisterTool(tool LLMTool)

    // RegisterChatCommand adds a slash command to the chat REPL.
    RegisterChatCommand(cmd ChatCommand)

    // RegisterCLICommand adds a subcommand to the CLI.
    // cmd must be a pointer to a kong-annotated struct with a Run() method.
    RegisterCLICommand(cmd any)

    // Go enqueues a background goroutine to be started after all plugins
    // have finished Init (i.e. after InitAll returns).
    // Each goroutine receives a context.Context that is cancelled when
    // Hub.Stop() is called. Hub.Stop() blocks until all goroutines return.
    // Calling Go after Start has no effect and panics in debug builds.
    Go(f func(ctx context.Context))

    // Stop cancels all goroutine contexts and waits for them to finish.
    // Called once by main when the program is shutting down.
    Stop()
}
```

### Supporting types

```go
// LLMTool is a single tool exposed to the LLM in the tool-use loop.
type LLMTool struct {
    // Definition returns the OpenAI-compatible tool schema.
    Definition func() openai.Tool
    // Execute is called when the LLM invokes this tool.
    // args is the raw JSON arguments string.
    Execute func(ctx context.Context, args string) (string, error)
    // IsSearchTool marks tools that count toward the per-request search call
    // limit (e.g. rag_search, web_fetch).
    IsSearchTool bool
}

// ChatCommand is a slash command available in the chat REPL.
type ChatCommand struct {
    // Name is the primary command name without "/", e.g. "stat".
    // Matching is case-insensitive.
    Name    string
    Aliases []string
    // Help is a one-line description shown in /help output.
    Help    string
    // Execute is called when the command is matched.
    // args contains any text following the command name (trimmed).
    Execute func(ctx context.Context, agent AgentHandle, args string) error
}

// AgentHandle is passed to ChatCommand.Execute.
// It exposes the minimal API that slash commands need without giving
// plugins direct access to ChatAgent internals.
type AgentHandle interface {
    // Messages returns a read-only snapshot of conversation history.
    Messages() []openai.ChatCompletionMessage
    // SetMessages replaces conversation history.
    SetMessages([]openai.ChatCompletionMessage)
    // Config returns the agent's effective configuration.
    Config() *config.Config
    // Compact runs history compression via the summarizer and replaces
    // the history with the compacted version.
    // Used by the /compact chat command.
    Compact() error
    // Reset clears conversation history, keeping only the system prompt.
    // Used by the /new chat command. The agent owns the system prompt
    // text; the plugin does not need to know it.
    Reset()
}
```

### Global registry functions

```go
// internal/plugin/registry.go

// Register records a plugin under the given name.
// Intended to be called from a plugin package's init() function.
// Panics if name is already registered.
func Register(name string, p Plugin)

// InitAll initialises every registered plugin in registration order and then
// starts all goroutines that plugins queued via Hub.Go.
// The two steps happen in strict sequence:
//  1. Init(s, hub) is called on every plugin.
//  2. All queued goroutines are launched.
// Must be called once in main(), after the config is loaded.
func InitAll(s State) error

// LLMTools returns all LLM tools registered by plugins.
func LLMTools() []LLMTool

// ChatCommands returns all chat commands registered by plugins.
func ChatCommands() []ChatCommand

// KongPlugins returns a kong.Plugins slice of all CLI command structs
// registered by plugins, ready to be embedded in the root CLI struct.
func KongPlugins() kong.Plugins

// Stop cancels all managed goroutine contexts and waits for them to finish.
// Must be called in main() before exit (e.g. via defer).
func Stop()
```

### Hub implementation sketch

```go
// internal/plugin/registry.go (internal type)

type hub struct {
    tools      []LLMTool
    commands   []ChatCommand
    cli        []any

    goroutines []func(ctx context.Context) // queued during Init, launched by start()

    cancel context.CancelFunc
    wg     sync.WaitGroup
}

// Called by each plugin during Init to enqueue a goroutine.
func (h *hub) Go(f func(ctx context.Context)) {
    h.goroutines = append(h.goroutines, f)
}

// start is called by InitAll after all Init calls have returned.
// It creates a single shared context and launches every queued goroutine.
func (h *hub) start() {
    ctx, cancel := context.WithCancel(context.Background())
    h.cancel = cancel
    for _, f := range h.goroutines {
        h.wg.Add(1)
        go func(f func(context.Context)) {
            defer h.wg.Done()
            f(ctx)
        }(f)
    }
}

func (h *hub) Stop() {
    if h.cancel != nil {
        h.cancel()
    }
    h.wg.Wait()
}
```

And `InitAll` ties the two phases together:

```go
func InitAll(s State) error {
    for _, entry := range registry { // registration-order slice
        if err := entry.plugin.Init(s, globalHub); err != nil {
            return fmt.Errorf("plugin %q: %w", entry.name, err)
        }
    }
    globalHub.start() // launch all queued goroutines only after every Init
    return nil
}
```

---

## Examples

### RAG plugin (LLM tool)

```go
// internal/plugins/rag/plugin.go
package ragplugin

func init() {
    plugin.Register("rag", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(s plugin.State, hub plugin.Hub) error {
    tools, err := rag.New(s.Config)
    if err != nil {
        return err
    }
    hub.RegisterTool(plugin.LLMTool{
        Definition:   tools.DefinitionSearch,
        IsSearchTool: true,
        Execute: func(_ context.Context, args string) (string, error) {
            return tools.Dispatch("rag_search", args), nil
        },
    })
    return nil
}
```

### Indexer plugin (CLI subcommand)

```go
// internal/plugins/cli/indexer/plugin.go
package indexerplugin

func init() {
    plugin.Register("cli.indexer", &Plugin{})
}

type Plugin struct{ command indexer.Cmd }

func (p *Plugin) Init(_ plugin.State, hub plugin.Hub) error {
    hub.RegisterCLICommand(&p.command)
    return nil
}
```

### `/stat` plugin (chat command)

```go
// internal/plugins/chatcmd/stat/plugin.go
package statplugin

func init() {
    plugin.Register("chatcmd.stat", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(_ plugin.State, hub plugin.Hub) error {
    hub.RegisterChatCommand(plugin.ChatCommand{
        Name: "stat",
        Help: "Print context statistics (messages, tokens, models).",
        Execute: func(_ context.Context, a plugin.AgentHandle, _ string) error {
            cfg := a.Config()
            msgs := a.Messages()
            // ... format and print stats
            _ = cfg
            _ = msgs
            return nil
        },
    })
    return nil
}
```

### Plugin with a background goroutine

```go
func (p *Plugin) Init(s plugin.State, hub plugin.Hub) error {
    hub.Go(func(ctx context.Context) {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // periodic background work
            }
        }
    })
    return nil
}
```

### `main.go`

The order of calls in `main()` is strict and must not be changed:

1. `config.Load` — load config
2. `plugin.InitAll` — initialise all plugins (registers CLI commands, tools, chat
   commands, enqueues goroutines, then starts them)
3. `defer plugin.Stop()` — ensure goroutines are stopped on exit
4. `cmd.Execute()` — parse CLI and run; calls `plugin.KongPlugins()` internally

`plugin.KongPlugins()` panics if called before `InitAll` to make mistakes
immediately visible.

```go
// cmd/go-magnetar/main.go
package main

import (
    _ "go-magnetar/internal/plugins/rag"
    _ "go-magnetar/internal/plugins/web"
    _ "go-magnetar/internal/plugins/generic"
    _ "go-magnetar/internal/plugins/cli/indexer"
    _ "go-magnetar/internal/plugins/cli/agent"
    _ "go-magnetar/internal/plugins/chatcmd/help"
    _ "go-magnetar/internal/plugins/chatcmd/exit"
    _ "go-magnetar/internal/plugins/chatcmd/new"
    _ "go-magnetar/internal/plugins/chatcmd/compact"
    _ "go-magnetar/internal/plugins/chatcmd/stat"
)

func main() {
    cfg, err := config.Load(...)
    if err != nil { ... }

    // InitAll must be called before cmd.Execute so that CLI commands,
    // tools and chat commands are registered before kong parses os.Args.
    if err := plugin.InitAll(plugin.State{Config: cfg, ...}); err != nil { ... }
    defer plugin.Stop()

    cmd.Execute()
}
```

### `internal/cmd/cmd.go`

`KongPlugins()` panics if the registry has not been initialised yet (i.e. if
`InitAll` was not called before `Execute`):

```go
// internal/plugin/registry.go
func KongPlugins() kong.Plugins {
    if !globalHub.initialised {
        panic("plugin: KongPlugins called before InitAll")
    }
    return globalHub.cliPlugins()
}
```

```go
// internal/cmd/cmd.go
func Execute() error {
    cli := &struct{ kong.Plugins }{}
    cli.Plugins = plugin.KongPlugins()
    k := kong.Must(cli, kong.Name("go-magnetar"), ...)
    ctx, err := k.Parse(os.Args[1:])
    if err != nil {
        return err
    }
    return ctx.Run()
}
```

---

## Directory Structure After Migration

```
internal/
  plugin/
    plugin.go     <- Plugin, Hub, AgentHandle interfaces;
                     State, LLMTool, ChatCommand types
    registry.go   <- global registry; hub implementation;
                     Register, InitAll, Stop, LLMTools,
                     ChatCommands, KongPlugins functions
  plugins/
    rag/plugin.go
    web/plugin.go
    generic/plugin.go
    cli/
      indexer/plugin.go
      agent/plugin.go
    chatcmd/
      help/plugin.go
      exit/plugin.go
      new/plugin.go
      compact/plugin.go
      stat/plugin.go
```

Packages `internal/tools/*`, `internal/agent/*`, `internal/cmd/*` are preserved —
plugins delegate to them.

---

## Migration Plan

The migration is split into 5 stages. Each stage is a separate commit;
the code compiles and existing tests pass after every stage.

### Stage 1 — Create `internal/plugin/` (infrastructure)

No existing code is modified; only new files are added.

**Create:**
- `internal/plugin/plugin.go` — `Plugin`, `Hub`, `AgentHandle` interfaces;
  `State`, `LLMTool`, `ChatCommand` types
- `internal/plugin/registry.go` — global plugin list, `hub` implementation
  (with `Go`/`Stop` goroutine management via `sync.WaitGroup` + `context.WithCancel`),
  and all public accessor functions

**Verification:** `make lint` + `make build` — compiles cleanly.

---

### Stage 2 — Wrap existing tools in plugins

Create thin wrapper packages. Original `internal/tools/*` packages are untouched.

**Create:**

| Package | Registers |
|---|---|
| `internal/plugins/rag/` | `rag_search` LLM tool |
| `internal/plugins/web/` | `web_fetch` LLM tool |
| `internal/plugins/generic/` | `file_read`, `file_list`, `file_write`, `file_exists` LLM tools |

**Verification:** `make lint`.

---

### Stage 3 — Wrap chat commands in plugins

**Create:**

| Package | Command |
|---|---|
| `internal/plugins/chatcmd/help/` | `/help` |
| `internal/plugins/chatcmd/exit/` | `/exit` |
| `internal/plugins/chatcmd/new/` | `/new` |
| `internal/plugins/chatcmd/compact/` | `/compact` |
| `internal/plugins/chatcmd/stat/` | `/stat` |

**Verification:** `make lint`.

---

### Stage 4 — Wrap CLI subcommands in plugins

**Create:**

| Package | Command |
|---|---|
| `internal/plugins/cli/indexer/` | `indexer` subcommand |
| `internal/plugins/cli/agent/` | `agent` subcommand |

**Adapt `internal/cmd/cmd.go`:** remove hardcoded `Indexer`/`Agent` fields;
`Execute()` reads `plugin.KongPlugins()` and populates `cli.Plugins`.

**Verification:** `make build` + smoke test of both subcommands.

---

### Stage 5 — Wire plugins into `chat/agent.go` and `main.go`

**`internal/agent/chat/agent.go`:**
- `New(cfg, root)` reads tools and chat commands from `plugin.LLMTools()` and
  `plugin.ChatCommands()` — no registry parameter needed (global accessors)
- Replace `switch name { ... }` dispatch with a `map[string]LLMTool` built from
  `plugin.LLMTools()`
- Search call limit: count calls where `tool.IsSearchTool == true`
- Replace `handleCommand()` switch with the following algorithm:
  ```go
  // 1. Strip leading "/" and split into name + args.
  //    "/help foo bar" → name="help", args="foo bar"
  //    "exit"         → name="exit", args=""  (alias without slash)
  trimmed := strings.TrimPrefix(strings.TrimSpace(line), "/")
  parts   := strings.SplitN(trimmed, " ", 2)
  name    := strings.ToLower(parts[0])
  args    := ""
  if len(parts) == 2 {
      args = strings.TrimSpace(parts[1])
  }

  // 2. Match against registered commands (name and aliases).
  for _, cmd := range plugin.ChatCommands() {
      if name == strings.ToLower(cmd.Name) || slices.Contains(lowered(cmd.Aliases), name) {
          err := cmd.Execute(ctx, agentHandle, args)
          ...
      }
  }
  ```
- Implement `AgentHandle` via a private `chatAgentHandle` adapter;
  `Compact()` delegates to `a.summarizer.Compact`, `Reset()` rebuilds
  `a.messages` from the system prompt stored in `a.messages[0]`
- Remove fields `a.rag`, `a.web`, `a.generic`

**`cmd/go-magnetar/main.go`:**
- Add blank plugin imports
- Call `plugin.InitAll(plugin.State{...})` after config load
- Add `defer plugin.Stop()`

**Verification:** `make build` + `make lint` + full smoke test.

---

## What Does Not Change

- `internal/tools/*` — library packages, independent of the plugin system
- `internal/chunk/*` — unchanged
- `internal/config/*` — unchanged
- `internal/agent/summarizer/` — unchanged
- `internal/agent/markdown/` — unchanged

---

## Constraints

**`InitAll` must be called exactly once** per process. It is not idempotent: calling
it twice registers tools and commands twice and starts goroutines twice. The
`initialised` flag on `hub` enforces this — a second call to `InitAll` panics.

---

## Testing

Plugins should be tested by injecting a fake `Hub` directly, without touching the
global registry:

```go
func TestRAGPlugin(t *testing.T) {
    h := &fakeHub{}
    p := &ragplugin.Plugin{}
    err := p.Init(plugin.State{Config: testConfig()}, h)
    require.NoError(t, err)
    require.Len(t, h.tools, 1)
    require.Equal(t, "rag_search", h.tools[0].Definition().Function.Name)
}
```

When a test does need the global registry (e.g. integration tests), call
`plugin.Reset()` in `TestMain` or per-test setup to clear it between runs.
`Reset()` panics if called after `InitAll` has already started goroutines (i.e.
after `hub.start()`) to prevent accidental use in production.

```go
// internal/plugin/registry.go

// Reset clears all registered plugins and resets the hub to its initial state.
// Intended for use in tests only. Panics if goroutines have already been started.
func Reset() {
    if globalHub.initialised {
        panic("plugin: Reset called after InitAll — not safe in production")
    }
    registry = nil
    globalHub = newHub()
}
```

---

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| `InitAll` called more than once | `hub.initialised` flag panics on second call |
| `KongPlugins` called before `InitAll` | Panics with a clear message |
| Global state makes testing harder | Inject fake `Hub` directly into the plugin under test; use `plugin.Reset()` for integration tests |
| `init()` order is non-deterministic across packages | Tool/command ordering is not semantic — any order is acceptable. CLI command order in `--help` follows registration order within a single binary |
| Goroutine leak if `Stop()` not called | `defer plugin.Stop()` in `main()` guarantees cleanup; Hub uses `context.WithCancel` + `sync.WaitGroup` |
| Circular dependency plugin → agent → plugin | `AgentHandle` interface breaks the cycle; plugins depend only on `internal/plugin` |
| Search tool call limit | Counter stays in `ask()`; plugins mark tools with `IsSearchTool: true` |
