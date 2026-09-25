# go-magnetar

An AI chat agent with an integrated knowledge base. Chat with any OpenAI-compatible LLM, index documents and web pages into a local vector database, and get answers grounded in your own content.

## What it does

- **Chat** — interactive REPL with multi-turn conversation and automatic context compaction
- **Index** — `/index <file|url>` adds documents to your RAG knowledge base (optional)
- **Fetch** — `/fetch <url>` retrieves and displays web content as Markdown
- **Integrations** — Confluence, JIRA, GitHub, GitLab (all optional, disabled by default)

Supported file formats: `.md`, `.txt`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, `.html`

## Requirements

- Go 1.27+ (or download a pre-built binary)
- API key for any OpenAI-compatible provider (OpenAI, Azure OpenAI, Ollama, etc.)
- [Qdrant](https://qdrant.tech/) — only if you want to use the RAG knowledge base (`rag.enable: true`)

## Quick start

### 1. Install

**Option A — pre-built binary:**

Download from [GitHub Releases](https://github.com/wmentor/go-magnetar/releases/) and place it in your `$PATH`.

**Option B — `go install`:**

```bash
go install github.com/wmentor/go-magnetar/cmd/go-magnetar@v1.2.0
```

**Option C — build from source:**

```bash
git clone https://github.com/wmentor/go-magnetar.git
cd go-magnetar
make build          # binary → bin/go-magnetar
```

### 2. Configure

On first run, go-magnetar creates `~/.go-magnetar/config.yml` automatically.

Edit it to set your LLM API key and model:

```yaml
version: "1.0"
language: english
verbose: true
profile: default

profiles:
  default:
    llm:
      base_url: https://api.openai.com/v1
      api_key: $env:OPENAI_API_KEY   # or paste the key directly
      model: gpt-4o
      context: 128000
      temperature: 0.9
```

For a complete list of options see [docs/configuration.md](./docs/configuration.md).

### 3. Run

```bash
go-magnetar
```

This opens the interactive chat REPL. Type a question and press Enter:

```
> What is the capital of France?
Paris is the capital of France.

> /help
...

> ^D   ← Ctrl+D or /exit to quit
```

## What the agent can do

Beyond answering questions, the agent has access to tools it can call automatically:

| Tool | Description |
|---|---|
| `file_read` | Read local files (`.md`, `.txt`, `.pdf`, `.docx`, `.xlsx`, and more) |
| `file_list` | List files in the current directory by glob pattern |
| `file_write` | Write content to a file |
| `exec` | Run shell commands (with a built-in safety guard) |
| `system_grep` | Search file contents with grep |
| `web_fetch` | Fetch a URL and return its content as Markdown |
| `web_search` | Search the web and return results |
| `rag_search` | Search your indexed knowledge base (when RAG is enabled) |
| `ssh` | Execute commands on a remote server via SSH (when enabled) |
| `cve` | Look up vulnerability info from the OSV database |
| `github_*` | Read GitHub repos, files, issues, milestones |

The agent picks tools automatically based on your question. You don't need to specify which tool to use.

## Using the knowledge base (RAG)

The RAG feature is **optional**. Enable it when you want the agent to answer questions based on your own documents.

### Start Qdrant

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Enable RAG in config

Edit `~/.go-magnetar/plugins/rag.yml` (created automatically on first run):

```yaml
rag:
  enable: true
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
    model: text-embedding-3-small
    vector_size: 1536
  qdrant:
    connstr: http://localhost:6333
    collection: documents
```

The main `~/.go-magnetar/config.yml` already includes `plugins/rag.yml` via the `include` directive — no changes to it are needed.

### Index documents

```bash
go-magnetar
> /index docs/guide.md
> /index https://example.com/article
> /index https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456
> /index https://jira.example.com/browse/PROJECT-123
```

The agent will now search the knowledge base before answering any question.

## Chat commands

| Command | Alias | Description |
|---|---|---|
| `/help` | `/h` | Show available commands |
| `/exit` | `/quit` | Exit the program |
| `/index <path\|url>` | `/i` | Index a file or URL into the knowledge base |
| `/fetch <url> [file]` | `/f` | Fetch URL content, display or save to file |
| `/copy` | `/c` | Copy the last answer to clipboard |
| `/write <file>` | `/w` | Write last answer to a file |
| `/new` | — | Clear conversation history and start fresh |
| `/compact` | — | Compress conversation history immediately |
| `/stat` | — | Show context stats (tokens, messages, models) |
| `/profile [name]` | — | Show or switch profile |
| `/readonly` | — | Toggle read-only mode |
| `/session.save <file>` | — | Save conversation to a JSON file |

Use **↑/↓** arrow keys to navigate command history (stored in `~/.go-magnetar-history.json`).

## Sessions

Save the current conversation to a file and restore it later:

```
> /session.save /tmp/my-session.json
```

```bash
# Resume later
go-magnetar --session /tmp/my-session.json
```

## Security

The agent can execute shell commands via the `exec` tool. To stay safe:

- **Safety guard** — dangerous commands (`rm -rf /`, `sudo`, `git push`, package managers, etc.) are blocked automatically. Enable with `guard.enable: true` in config.
- **Read-only mode** — toggle with `/readonly` to block all file writes and shell modifications for the session.
- **Root user** — go-magnetar refuses to run as root.
- **Sensitive env vars** — variables matching patterns like `TOKEN`, `KEY`, `PASS`, `SECRET` are stripped from the command environment before execution.

See [docs/security.md](./docs/security.md) for details.

## CLI flags

| Flag | Description |
|---|---|
| `-p <profile>` | Use a specific configuration profile |
| `-f <file>` | Non-interactive mode: read input from file, print answer, exit |
| `--session <file>` | Load a previously saved conversation session |

```bash
# Non-interactive (scripting / batch)
go-magnetar -f questions.txt

# Load a saved session
go-magnetar --session session.json

# Use a named profile
go-magnetar -p production
```

> **Note:** In `-f` mode, chat commands (`/index`, `/fetch`, etc.) are not available. Text preprocessor placeholders like `{{date}}`, `{{uuid}}`, `{{file:path}}` are expanded.

## Documentation

- [Configuration reference](./docs/configuration.md)
- [Chat commands](./docs/chat_command.md)
- [Agent tools](./docs/agent_tools.md)
- [File format support](./docs/file_codecs.md)
- [Text preprocessor](./docs/preprocessor.md)
- [Security](./docs/security.md)
- [User manual](./docs/user_manual.md)

## License

See [LICENSE](LICENSE).
