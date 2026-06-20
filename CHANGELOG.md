# Changelog

All notable changes to go-magnetar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2026-06-20

### Added

- **RAG Indexer (`indexer` subcommand)**
  - Index single files (`.md`, `.txt`) via `-f` flag
  - Index entire directories recursively via `-d` flag  
  - Index web pages via `-u` flag (HTML cleaned and converted to Markdown)
  - Deterministic chunk IDs using UUID v5 (content-based, idempotent)
  - Paragraph and Markdown heading boundary-aware chunking
  - Configurable chunk size and overlap
  - Automatic Qdrant collection creation
  - Proper error handling (continues on single-file failures during directory indexing)

- **Chat Agent (`agent` subcommand)**
  - Interactive REPL with multiline input support
  - Multi-turn conversation history
  - Automatic history compaction when token threshold is reached
  - `/help` command — show available chat commands
  - `/exit` / `exit` commands — end session and exit
  - `/compact` command — manually trigger history summarization
  - `/new` command — start new session and clear context
  - `/stat` command — display context statistics (messages, tokens, bytes, models)

- **Tool Calling System**
  - Built-in tools exposed to LLM:
    - `rag_save` — save content chunks to knowledge base
    - `rag_search` — search knowledge base for relevant information
    - `file_read` — read files from filesystem
    - `file_list` — recursively list files with optional filters
    - `file_write` — write content to files
    - `file_exists` — check file existence
    - `web_fetch` — fetch and preprocess web pages (fallback when RAG has no results)
  - Smart tool selection strategy:
    - Always tries `rag_search` first, even for seemingly known facts
    - Uses `web_fetch` only as fallback when RAG returns no relevant results
    - Never hallucinates — answers strictly based on tool results

- **HTML Preprocessing Pipeline**
  - HTML noise removal (ads, navigation, cookie banners, etc.)
  - Readability extraction (article content extraction)
  - HTML to Markdown conversion
  - Markdown cleanup (removes meta-info, subscribe calls, related articles)

- **History Compaction (Summarizer)**
  - Automatic compaction when token threshold reached (default: 80% of context)
  - Preserves system message verbatim
  - Configurable tail preservation (`save_tail`)
  - Manual compaction via `/compact` command

- **Configuration System**
  - YAML config file support
  - OpenAI-compatible LLM endpoint support
  - Qdrant connection via gRPC
  - Embedding model configuration
  - Chunk size and overlap settings
  - Search threshold and limit configuration
  - Log level configuration (debug, info, warn, error)
  - Path defaults (expand `~` in config paths)

- **Logging**
  - Structured logging via `log/slog`
  - Colorised output to stderr using `tint`
  - Detailed debug logging for tool calls and search results
  - Progress indicators for indexing operations

- **Chunking Algorithm**
  - UTF-8 safe (operates on Unicode runes, not bytes)
  - Word-boundary snapping (no mid-word cuts)
  - Smart overlap with word boundary alignment
  - Automatic handling of oversized paragraphs (force-split)
  - Default: 512 runes chunk size, 64-rune overlap (~12.5%)

- **Search Features**
  - Cosine similarity threshold filtering
  - Configurable result limit
  - Search preview in debug logs
  - Score reporting for debugging

- **CLI Infrastructure**
  - `kong`-based CLI parser
  - Two subcommands: `indexer` and `agent`
  - Proper exit codes and error handling
  - Configurable config file paths with tilde expansion

### Technical Details

- Built with Go 1.26.4+
- Dependencies:
  - `github.com/alecthomas/kong` — CLI parsing
  - `github.com/sashabaranov/go-openai` — OpenAI API client
  - `github.com/qdrant/go-client` — Qdrant gRPC client
  - `github.com/google/uuid` — Deterministic UUID v5 generation
  - `github.com/knadh/koanf/v2` — YAML configuration loading
  - `github.com/lmittmann/tint` — Colorised logging
  - `github.com/charmbracelet/glamour` — Markdown rendering in terminal
  - `github.com/PuerkitoBio/goquery` — HTML cleaning
  - `codeberg.org/readeck/go-readability/v2` — Article extraction
  - `github.com/JohannesKaufmann/html-to-markdown/v2` — HTML to Markdown conversion

- Architecture:
  - Command-layer (`cmd/`) — CLI argument parsing
  - Config-layer (`config/`) — YAML loading and validation
  - Chunk-layer (`chunk/`) — Text chunking algorithm
  - Tool-layer (`tools/`) — LLM tool implementations
  - Agent-layer (`agent/`) — Indexer and Chat agents
  - Summarizer-layer (`agent/summarizer/`) — History compression
