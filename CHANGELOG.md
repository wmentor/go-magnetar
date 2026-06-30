# Changelog

All notable changes to go-magnetar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.3] - 2026-06-30

### Added

- **GitHub integration** - Full GitHub API integration with three new tools:
  - `github_repo` - Fetch repository information and README
  - `github_file` - Fetch file content from any branch
  - `github_tree` - List files and directories in current path
  - Support for URLs: `https://github.com/{owner}/{repo}`, `https://github.com/{owner}/{repo}/blob/{branch}/{file}`, `https://github.com/{owner}/{repo}/tree/{branch}/{path}`
  - Configurable via `github` block in config (base_url, access_key)
  -Integrated with `web_fetch` for automatic URL resolution
- **Documentation** - Comprehensive documentation for GitHub integration
  - Example URLs and configuration
  - Command examples in user_manual.md and README.md
  - Tool signatures and parameters

### Changed

- **Refactor gitlab fetch** - Move common functionality to internal/tools/gitlab/fetch.go
  - Improved code organization
  - Better testability
- **Refactor web fetch** - Extract URL routing to separate handlers
  - Cleaner separation between web, confluence, jira, gitlab, github
  - Easier to add new URL handlers

### Fixed

- Removed TODO.md from commit history (all tasks completed in v0.1.3)

## [v0.1.2] - 2026-06-28

### Added

- **GitLab MR fetching** - Fetch GitLab merge requests with file changes
  - Fetch MR details via `/api/v4/projects/{project}/merge_requests/{mr_id}`
  - Additional call to `/api/v4/projects/{project}/merge_requests/{mr_id}/changes`
  - Parse and include file diffs list in output
  - Support MR URLs in format `https://gitlab.example.com/group/project/-/merge_requests/123`
  - Configurable via `gitlab` block in config (base_url, api_key)
  - Examples in user_manual.md and README.md
- **version command** - Enhanced version command with timeout adjustment
  - Improved display format
  - Timeout adjustment support
- **ask tool** - Clarifying questions tool for the agent
  - Allows agent to ask user for clarification during conversation
  - Returns user's answer as string
- **JIRA plugin** - JIRA task fetching via LLM tool
  - Fetch JIRA issues directly via `jira_task_get` tool
  - Configurable via `jira` block in config
  - Based on JIRA REST API

### Changed

- **Config refactoring** - Rewrite Config to use `koanf.Koanf` directly
  - Simplified configuration structure
  - Better support for dynamic config loading
  - Optimized memory usage for shared structs
- **Plugin architecture** - Optimized plugin initialization
  - State passed by pointer instead of value
- **History management** - Refactored to use `internal/history` package
  - Added `Records()` method for history iteration
  - Optimized `New()` for empty filename
- **Logging** - Replaced `slog` with internal `printer` package
  - Unified logging system across the codebase
  - Consistent output format with icons and timestamps
- **Documentation** - Updated documentation with new features
  - Added GitLab MR support examples
  - Updated user_manual.md with new commands
  - Updated README.md with all supported features

### Fixed

- Multiple query processing in search with better deduplication
- Improved error handling for web fetch operations
- URL detection for Confluence, JIRA, and GitLab resources

### Removed

- Separate indexer CLI subcommand (`go-magnetar index`)
- CLI agent subcommand (functionality merged into unified agent)

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

[Unreleased]: https://github.com/wmentor/go-magnetar/compare/v0.1.3...HEAD
[v0.1.3]: https://github.com/wmentor/go-magnetar/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/wmentor/go-magnetar/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/wmentor/go-magnetar/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/wmentor/go-magnetar/releases/tag/v0.1.0
