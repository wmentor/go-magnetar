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
| `/copy [index]` | `/c` | Copy an assistant answer to clipboard (default: the last one) |
| `/revert` | `/r` | Revert to the previous assistant answer and display it again |
| `/readonly` | — | Toggle read-only mode (blocks all modification operations) |
| `/fetch` | `/f` | Fetch content from a URL, optionally save to file |
| `/exec` | `/e` | Execute a shell command, redirect stdout/stderr, print exit code on failure |
| `/confluence.edit` | — | Edit a Confluence page using Markdown content from a file |
| `/session.save` | — | Save current conversation session to a file |
| `/profile` | — | Show current profile or switch to a different profile |
| `/skill` | `/sk` | List available skills or load/activate a skill |

## /fetch Command

Fetch content from a URL, optionally save to file:

```
> /fetch https://example.com/article
> /fetch https://example.com/article output.md
```

The command cleans HTML content and converts it to Markdown. If a filename is provided, the content is saved to that file; otherwise, it's displayed in the terminal (using `less` if available).

## /exec Command

Execute a shell command and redirect stdout/stderr to the console:

```
> /exec ls -la
> /exec grep pattern file.txt
> /exec go build ./...
```

**Usage:**
```
/exec <command> [args...]
```

**Parameters:**
- `command` — Shell command to execute (e.g., `ls`, `grep`, `cat`)
- `args` — Additional arguments for the command

**Output:**
- `stdout` and `stderr` are printed directly to the console
- If the command fails (exit code != 0), the exit code is printed at the end: `[Exit code: N]`
- For non-command errors (e.g., command not found), an `[Error: <error message>]` line is printed

**Security:**
The command is blocked if it contains dangerous patterns:
- `rm -rf /` — recursive deletion of root directory
- `rm -rf *` — recursive deletion of all files in current directory
- `dd` with `if=` parameter — direct memory access (e.g., `dd if=/dev/zero`)
- `mkfs` — create filesystem commands
- `fdisk` — disk partitioning tool
- `bash -c`, `sh -c`, `zsh -c` — execute command from string argument
- `curl ... | bash`, `wget ... | sh` — download and pipe to shell

**Examples:**

Display files in current directory:
```
> /exec ls -la
total 123
drwxr-xr-x  10 user  staff   320 Oct 18 12:00 .
drwxr-xr-x  10 user  staff   320 Oct 17 10:30 ..
...
```

Search for pattern in file:
```
> /exec grep "TODO" *.go
file1.go:42:    // TODO: implement this
file2.go:15:    // TODO: fix this bug
```

Run Go build:
```
> /exec go build -v ./...
github.com/wmentor/go-magnetar
```

Command with failure (exit code printed):
```
> /exec cat nonexistent.txt
cat: nonexistent.txt: No such file or directory

[Exit code: 1]
```

Command not found (error printed):
```
> /exec nonexistentcommand
zsh:1: command not found: nonexistentcommand

[Error: exit status 127]
```

## /confluence.edit Command

Edit a Confluence page by uploading Markdown content:

```
> /confluence.edit <url> <markdown_file>
```

**Parameters:**
- `url` — Confluence page URL (supports full or short links)
- `markdown_file` — Path to local Markdown file containing the new page content

**Example:**
```
> /confluence.edit https://confluence.example.com/spaces/TEAM/pages/123456 /path/to/content.md
```

The command:
1. Extracts page ID and space key from the URL
2. Reads Markdown content from the specified file
3. Converts Markdown to HTML for Confluence storage format
4. Updates the page via Confluence REST API
5. Increments page version number automatically

**Configuration:**

Enable Confluence editing in your config:

```yaml
confluence:
  enable: true
  base_url: https://confluence.example.com
  api_key: $env:CONFLUENCE_API_KEY
```

See [docs/security.md](./security.md) for Confluence security considerations.

## /copy Command

Copy an assistant answer from the current conversation to the clipboard:

```
> /copy
Answer copied to clipboard
```

The optional `index` selects which answer to copy, counting back from the most recent one:

| Index | Answer |
|---|---|
| `0` (default) | The last answer |
| `1` | The answer before the last one |
| `n` | The answer `n` steps back |

```
> /copy 1
Answer copied to clipboard
```

Only answers shown to the user are counted. Intermediate assistant messages that request tool calls are skipped, so they never take up an index.

If the index is not a non-negative integer, the command prints its usage. If the index is larger than the number of answers in the history, nothing is copied:

```
> /copy 5
Nothing to copy: only 2 assistant answer(s) in history
```

## /revert Command

**/revert — the time machine for your conversation**

Revert to the previous assistant answer and display it again:

```
> /revert
Reverted to previous answer:
This is the previous answer
```

The `/revert` command acts as a **time machine** for your conversation. It undoes the last turn of the conversation by removing all messages after the last assistant answer (including the most recent user message), effectively rolling back the conversation to the previous state and displaying the previous answer. This is useful for:

- **Reviewing past responses**: See what the assistant previously said without navigating through history
- **Trying a different approach**: Go back to a previous point and change your question or request
- **Correcting course**: If the conversation took an unexpected turn, use `/revert` to go back and start fresh
- **Comparing answers**: Review previous responses side-by-side with your current context

The command:
- Identifies the last assistant answer (skipping tool calls)
- Removes all subsequent messages (including the last user message)
- Re-displays the previous answer

If there is only one assistant answer or no assistant answer in the conversation history, the command displays an appropriate message.

## /profile Command

Display available profiles and their status:

```
> /profile
Available profiles:
 *> default
    production
    development

Current profile: default
LLM model: gpt-4o
Total messages: 15
```

Switch to a different profile:

```
> /profile production
Switched to profile: production
```

## Command Syntax

Commands are entered in the REPL and always start with `/`:

```
> /help
> /exit
> /index docs/guide.md
> /session.save /tmp/session.json
```

## Command History

The chat agent maintains command history in `~/.go-magnetar-history.json`. Use **↑/↓** arrows to navigate through previously entered commands.

### History Storage

- **Location**: `~/.go-magnetar-history.json`
- **Format**: JSON lines file (one command per line)
- **Limit**: 200 entries (oldest entries are removed when limit is exceeded)

### Usage

- **↑** — navigate to previous command
- **↓** — navigate to next command

### Example

```
> ./bin/go-magnetar
> /index docs/guide.md
Indexed 15 chunks from docs/guide.md
> /fetch https://example.com/article
> # Press ↑ to see previous commands
```

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
go-magnetar --session /path/to/session.json
```

The conversation history is restored, and you can continue from where you left off.

### Non-interactive mode with session

You can also use saved sessions with the `-f` flag:

```bash
go-magnetar -f commands.txt --session session.json
```

In this mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available.

## Text Preprocessing

See [docs/preprocessor.md](./preprocessor.md) for a complete reference on text preprocessors and available placeholders.
