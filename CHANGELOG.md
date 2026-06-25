# Changelog

All notable changes to go-magnetar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.1] - 2026-06-25

### Added

- **system_exec tool** - Execute system commands with arguments
- **system_date tool** - Execute the `date` command to get current system time
- **Read-only mode** - Prevent file modifications with configurable allowed commands
- **/readonly chat command** - Toggle read-only mode (`/readonly on|off`)
- **/index command** - Unified document indexing from REPL (replaces separate CLI subcommands)
  - Auto-detects file vs URL
  - Supports `-m <message>` flag for prepending context to chunks
  - Works with Confluence pages and JIRA issues
- **/idxtab command** - Batch indexing from JSON lines file
  - Format: `{"source":"path|url","message":"text"}`
- **JIRA issue support** - Fetch JIRA issues directly by URL via `web_fetch` tool

### Changed

- **CLI structure** - Simplified to single command approach
  - Removed `internal/cmd/indexer/cmd.go`
  - Removed `internal/plugins/cli/agent/plugin.go`
  - Removed `internal/plugins/cli/indexer/plugin.go`
  - Single point of entry: `go-magnetar [flags]`
- **Plugin architecture** - State passed by pointer instead of value
  - Optimizes memory usage for shared structs
- **History management** - Refactored to use `internal/history` package
  - Added `Records()` method for history iteration
  - Optimized `New()` for empty filename

### Fixed

- Refactored CLI to use `/index` chat command instead of separate indexer command
- Replaced `kong.Plugins` with direct chat command registration for simplicity

### Removed

- Separate indexer CLI subcommand (`go-magnetar index`)
- CLI agent subcommand (functionality merged into unified agent)

## [v0.1.0] - 2026-06-23

### Added

- Initial release
- RAG-based knowledge base with document indexing
- Interactive chat agent with multi-turn conversation
- Web page fetching with HTML cleanup
- Confluence page fetching support
- `/save <filename>` command to save last assistant answer
- `/version` command to display program version
- Plugin architecture with dynamic registration
- Search engine with multi-query and deduplication support

### Changed

- Plugin architecture refactoring
- Search enhancements

[Unreleased]: https://github.com/wmentor/go-magnetar/compare/v0.1.1...HEAD
[v0.1.1]: https://github.com/wmentor/go-magnetar/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/wmentor/go-magnetar/releases/tag/v0.1.0
