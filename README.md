# go-magnetar

An AI agent that combines retrieval capabilities with a powerful plugin system. The agent can index documents, fetch web content, and execute tools through a unified interactive REPL.

## How it works

**Indexing** — reads `.md`, `.txt`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, and `.html` files or web pages (via URL), splits content into overlapping chunks respecting paragraph and Markdown heading boundaries, computes embedding vectors and stores each chunk in Qdrant. Each chunk is identified by a deterministic UUID v5 derived from its content, making re-indexing idempotent: the same chunk is never stored twice.

**Chat agent** — an interactive REPL with multi-turn conversation support. The agent automatically decides which tools to call based on the user query. It always tries `rag_search` first; if relevant results are found in the knowledge base, the answer is based exclusively on those results. `web_fetch` and `web_search` are used as fallbacks when external or up-to-date information is needed. If no tool can provide an answer, the agent explicitly states this. Conversation history is automatically managed to stay within the configured context window: when history approaches the token threshold it is compacted by the built-in summarizer, which replaces older turns with a concise summary while keeping the most recent ones verbatim.

**Indexing via chat** — the `/index` command (alias `/i`) allows direct document indexing from the REPL. Simply type `/index <path|url>` to add documents to your knowledge base.

**Fetch content** — the `/fetch` command (alias `/f`) retrieves content from URLs, cleans HTML, and displays it in the terminal (using `less` if available) or saves it to a file.

**HTML Preprocessing** — web pages fetched via `web_fetch` are cleaned of ads, navigation, cookie banners, and other noise, then processed through readability extraction and converted to Markdown before being indexed or returned to the agent. Confluence URLs are also handled via the `confluence` block to fetch pages by ID, JIRA issues via the `jira` block, GitHub repositories via the `github` block to fetch repository information, files, and directory trees, and GitLab merge requests via the `gitlab` block to fetch MR details and file changes.

## Requirements

- Go 1.27.0
- An API key for any OpenAI-compatible provider (for the chat model, embedding model, and optionally for web page preprocessing)

## Quick start

### 1. Installation

#### Option 1: Download pre-built binary

Download the latest release for your platform and architecture from [GitHub Releases](https://github.com/wmentor/go-magnetar/releases/). Extract the archive and run the binary directly.

#### Option 2: Install via go install

The easiest way to install go-magnetar is via `go install`:

```bash
go install github.com/wmentor/go-magnetar/cmd/go-magnetar@v1.2.0
```

The binary will be placed in your `$GOPATH/bin` or `$GOBIN` directory.

#### Option 3: Build from source

Clone the repository and build manually:

```bash
git clone https://github.com/wmentor/go-magnetar.git
cd go-magnetar
make build
```

The binary will be placed at `${HOME}/.local/bin/go-magnetar`.

### 2. Configure

Configuration files are automatically created in `~/.go-magnetar/` on first run with default plugin configurations.

Plugin configurations are disabled by default. To enable a plugin (e.g., rag), set `enable: true` in its configuration file.

See [docs/configuration.md](./docs/configuration.md) for complete configuration options.

### 3. Ask questions

```bash
./bin/go-magnetar
```

This opens an interactive chat session with the AI agent. Enter your questions or chat commands to interact with the agent.

## Commands

Configuration file is always loaded from `~/.go-magnetar/config.yml`.

### `-p/--profile` — select configuration profile

```
go-magnetar -p <profile_name>
```

The `-p` flag overrides the profile specified in the configuration file. This allows switching between different configurations (e.g., `default`, `production`, `development`) without modifying the config file.

### `-f/--file` — non-interactive mode

```
go-magnetar -f <input-file>
```

Reads input from the specified file, sends it to the agent, prints the answer, and exits. This mode is useful for scripting and batch processing.

> **Note:** In `-f/--file` mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available. The text preprocessor expands placeholders like `{{home}}`, `{{uuid}}`, `{{date}}`, `{{now}}`, and `{{file:filename}}`.

```
go-magnetar
```

Run the interactive agent REPL. Press `Ctrl+D` to exit.

## License

See [LICENSE](LICENSE).
