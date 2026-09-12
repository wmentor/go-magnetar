# Chat Commands

go-magnetar provides several built-in chat commands accessible from the interactive REPL.

## Built-in Chat Commands

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
| `/readonly` | — | Toggle read-only mode (blocks all modification operations) |
| `/fetch` | `/f` | Fetch content from a URL, optionally save to file |
| `/session.save` | — | Save current conversation session to a file |

## Command Syntax

Commands are entered in the REPL and always start with `/`:

```
> /help
> /exit
> /index docs/guide.md
> /session.save /tmp/session.json
```

## Command History

Use **↑/↓** arrows to navigate through previously entered commands. History is persisted in `~/.go-magnetar-history.json` and limited to 200 entries.

## Command Execution Flow

1. Input is stripped of leading `/` and split into `name` + `args` on the first space
2. Command matching is case-insensitive against `Name` and `Aliases`
3. Commands are never added to the message history
4. Commands are dispatched by iterating over `plugin.ChatCommands()`

## Saving and Loading Sessions

### Save a session

To save your current conversation session:

```
/session.save /path/to/session.json
```

The session is saved in OpenAI-compatible JSON format with the following structure:

```json
[
  {
    "role": "system",
    "content": "You are a helpful assistant..."
  },
  {
    "role": "user",
    "content": "Hello"
  },
  {
    "role": "assistant",
    "content": "Hi there! How can I help you?"
  }
]
```

### Load a session on startup

To load a previously saved session:

```bash
go-magnetar -c config.yaml --session /path/to/session.json
```

The conversation history is restored, and you can continue from where you left off.

### Non-interactive mode with session

You can also use saved sessions with the `-f` flag:

```bash
go-magnetar -c config.yaml -f commands.txt --session session.json
```

In this mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available.

## Text Preprocessing

See [docs/preprocessor.md](./preprocessor.md) for a complete reference on text preprocessors and available placeholders.
