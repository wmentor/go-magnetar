# go-magnetar

An AI chat agent with an integrated knowledge base. Chat with any OpenAI-compatible LLM, index documents and web pages into a local vector database, and get answers grounded in your own content.

## What it does

- **Chat** — interactive REPL with multi-turn conversation and automatic context compaction
- **Index** — `/index <file|url>` adds documents to your RAG knowledge base (optional)
- **Fetch** — `/fetch <url>` retrieves and displays web content as Markdown
- **Integrations** — Confluence, JIRA, GitHub, GitLab (all optional, disabled by default)

Supported file formats: `.md`, `.txt`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, `.html`

## Requirements

- Go 1.27.2 (or download a pre-built binary)
- API key for any OpenAI-compatible provider (OpenAI, Azure OpenAI, Ollama, etc.)

**Note:** Qdrant is **no longer required**. The default vector store is **Chromem** (built-in, in-memory with optional persistence). Qdrant is available as an optional external store for production deployments.

## Vector Stores

go-magnetar supports two vector storage backends:

### Chromem (default)

- **Built-in**: No external dependencies required
- **In-memory by default**: Store exists only during session (when `rag.chromem.data_dir` is empty)
- **Persistent mode**: Set `rag.chromem.data_dir` to persist vectors to disk
- **Use case**: Local development, testing, edge deployments

### Qdrant (external)

- **External server**: Requires running Qdrant container
- **Full persistence**: Automatic disk persistence with advanced filtering
- **Use case**: Production, large-scale deployments

Switch between stores via config:
```yaml
rag:
  store:
    type: chromem  # or qdrant
```

See [docs/architecture.md](./docs/architecture.md) for vector store architecture details.

## Quick start

### 1. Install

**Option A — pre-built binary:**

Download from [GitHub Releases](https://github.com/wmentor/go-magnetar/releases/) and place it in your `$PATH`.

**Option B — `go install`:**

```bash
go install github.com/wmentor/go-magnetar/cmd/go-magnetar@latest
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
| `web_fetch` | Fetch a URL and return its content as Markdown (transparently handles Confluence, JIRA, GitHub, GitLab, RSS feeds when corresponding plugins are enabled). See [docs/tool_web_fetch.md](./docs/tool_web_fetch.md) for complete documentation |
| `search` | Execute web and knowledge base search in parallel and return merged results |
| `ssh` | Execute commands on a remote server via SSH (when enabled) |
| `cve` | Look up vulnerability info from the OSV database |
| `github_*` | Read GitHub repos, files, issues, milestones |
| `confluence_page_edit` | Edit a Confluence page by ID with Markdown content |

The agent picks tools automatically based on your question. You don't need to specify which tool to use.

## Using the knowledge base (RAG)

The RAG feature is **optional**. Enable it when you want the agent to answer questions based on your own documents.

### Enable RAG in config

By default, RAG uses Chromem (built-in, in-memory). For persistent storage, set `rag.chromem.data_dir`:

```yaml
rag:
  enable: true
  store:
    type: chromem  # or qdrant
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
    model: text-embedding-3-small
    vector_size: 1536
  chromem:
    data_dir: ~/.go-magnetar/store  # optional: set for persistence
    collection: documents  # collection name
```

The main `~/.go-magnetar/config.yml` already includes `plugins/rag.yml` via the `include` directive — no changes to it are needed.

For Qdrant (external store), use:

```yaml
rag:
  enable: true
  store:
    type: qdrant
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
    model: text-embedding-3-small
    vector_size: 1536
  qdrant:
    connstr: http://localhost:6333
    collection: documents
```

### Start Qdrant (optional, only if using Qdrant store)

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

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
| `/confluence.edit <page-id> <content>` | — | Edit a Confluence page by ID with Markdown content |
| `/idxtab <file>` | — | Index multiple documents from a JSON lines file |
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

## Data Residency

go-magnetar offers **100% local processing** options:

- **Chromem** (built-in vector store) — no external dependencies
- **Local LLMs** (Ollama, lm-studio, etc.) — run on your machine
- **Zero data exfiltration** — everything stays on your computer

When configured with Chromem + local LLM, **your documents, embeddings, and conversations never leave your machine**.

See [docs/data_residency.md](./docs/data_residency.md) for complete details.

## Security

See [docs/security.md](./docs/security.md) for complete security information.

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
| `--read-only` | Run in read-only mode (disable file writes and shell modifications) |

```bash
# Non-interactive (scripting / batch)
go-magnetar -f questions.txt

# Load a saved session
go-magnetar --session session.json

# Use a named profile
go-magnetar -p production

# Run in read-only mode
go-magnetar --read-only
```

> **Note:** In `-f` mode, chat commands (`/index`, `/fetch`, etc.) are not available. Text preprocessor placeholders like `{{date}}`, `{{uuid}}`, `{{file:path}}` are expanded.

## Documentation

- [Configuration reference](./docs/configuration.md)
- [Chat commands](./docs/chat_command.md)
- [Agent tools](./docs/agent_tools.md)
- [File format support](./docs/file_codecs.md)
- [Text preprocessor](./docs/preprocessor.md)
- [Security](./docs/security.md)
- [Data Residency](./docs/data_residency.md)
- [User manual](./docs/user_manual.md)

## License

See [LICENSE](LICENSE).
