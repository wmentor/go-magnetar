# Changelog

All notable changes to go-magnetar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.5] - 2026-07-29

### Added

- **DOCX file support** — full support for Microsoft Word documents
  - Add `internal/docx` package with `ReadFile` function for parsing `.docx` documents
  - Update `file_read` tool to support `.docx` files via `internal/docx.ReadFile`
  - Update indexer to index `.docx` files
  - Update `{{file:filename}}` preprocessor to read `.docx` files
  - Update documentation in README.md, AGENTS.md, docs/user_manual.md
- **PDF file support** — full support for PDF documents
  - Add `internal/pdf` package with `ReadFile` function for parsing `.pdf` documents
  - Update `file_read` tool to support `.pdf` files via `internal/pdf.ReadFile`
  - Update indexer to index `.pdf` files
  - Update `{{file:filename}}` preprocessor to read `.pdf` files
  - Update documentation in README.md, AGENTS.md, docs/user_manual.md
- **ODT file support** — full support for OpenOffice Writer documents
  - Add `internal/odt` package with `ReadFile` function for parsing `.odt` documents
  - Update `file_read` tool to support `.odt` files via `internal/odt.ReadFile`
  - Update indexer to index `.odt` files
  - Update `{{file:filename}}` preprocessor to read `.odt` files
  - Update documentation in README.md, AGENTS.md, docs/user_manual.md
- **User input preprocessors** — modular text preprocessing pipeline
  - Add `hub.RegisterPreprocessor()` API for plugins to transform user input
  - Preprocessors run after user input is displayed but before command matching or LLM processing
  - Used for simple placeholder expansion (`{{file:filename}}`)
- **Serial placeholders** — support for multiple `{{file:filename}}` placeholders
  - Expand all `{{file:filename}}` placeholders in user input
  - Reuse expansion result across multiple tool calls to avoid redundant file reads
- **parallel tool calling** — concurrent execution of LLM tool calls
  - Execute multiple tools in parallel when supported by LLM response
  - Improve performance for requests requiring multiple search operations
- **Go version bump** — updated to Go 1.26.5
  - Update minimum version requirement in README.md and AGENTS.md
  - Update GitHub Actions workflows
  - Update build instructions

### Changed

- **exec tool** — simplification and hardening
  - Simplified `system_grep` tool: removed optional parameters
  - Always use `-n -i -r -E` flags for consistent behavior
  - Updated documentation in AGENTS.md, README.md
- **file_list filter** — unified glob pattern interface
  - Add `filter` parameter to `file_list` tool for glob pattern matching
  - Replaces legacy directory listing behavior
  - Update documentation in AGENTS.md

### Fixed

- **Makefile improvements** — build process enhancements
  - `go fix` and `go fmt` integration
  - Automatic code formatting and fixes before build
  - Updated `make tidy`, `make build` targets
- **Go module fixes** — dependency management improvements
  - Resolve `go.mod` version conflicts
  - Update dependency constraints

## [v0.1.4] - 2026-07-06

### Added

- **Language parameter** — new `language` parameter for agent responses
  - Default: `english`
  - Configurable via `language` field in config
  - Injected into system prompt instead of hardcoded English
  - Updated in configs/config.yaml, README.md, AGENTS.md, docs/user_manual.md
- **llm.temperature and llm.top_p** — new LLM parameters
  - Default values: 0.5 and 0.95
  - Passed to OpenAI API in CreateChatCompletion request
  - Documented in all configuration examples
- **/fetch chat command** — new command for URL content retrieval
  - Alias: `/f`
  - Fetches and displays content from URLs
  - HTML cleanup and Markdown conversion via webfetch
  - Terminal display with `less` if available, or save to file
  - Example: `/fetch https://example.com/article [output.md]`
- **JIRA Epic child issues** — fetch child issues in Epic tasks
  - Fetch child issues via JQL search when issue type is "Epic"
  - Add labels and parent issue fields to output
  - Include child count and summary in Markdown output

### Changed

- **Refactor exec tool** — security-focused refactoring
  - Renamed `system_exec` to `exec` with simplified API (command + stdin)
  - Execute via `sh -c` with clean environment and current working directory
  - Added 1-minute timeout and 64KB output size limit
  - Implement blocklist: rm -rf, sudo, mkfs, dd, git ops, shell pipes (bash/sh/zsh)
  - Remove allowedCommands and read-only mode support
- **Security hardening** — comprehensive security improvements
  - Block execution when running as root user
  - Expand blocklist with privileged commands: su, chmod, chown, fdisk, format, brew, apt, dpkg, npm, systemctl, useradd/userdel, passwd, etc.
  - Add environment variable filtering to prevent credential leakage
  - Support read-only mode check in exec tool
  - Updated documentation and security guidelines

### FIXED

- **Search replace tool** — removed from generic plugin
  - Removed `search_replace` tool from internal/tools/generic
  - Updated documentation in AGENTS.md, README.md, docs/user_manual.md
- **Type cast bug** — fix float64 args in file_read tool
  - Changed limit/offset params from int to *float64 in generic tool args parsing
  - Support JSON numbers with decimal points (e.g., 100.0)
- **Ask tool** — removed from codebase
  - Delete internal/plugins/ask/plugin.go
  - Remove ask import from cmd/go-magnetar/main.go
  - Remove ask tool documentation from AGENTS.md

### Deprecated

- **Search replace tool** — removed from generic plugin (functionality replaced by internal/tools/generic)

### Security

- **Command safety guard** — LLM-based security analysis for exec commands
  - Implement guard agent with safety analysis
  - Block destructive commands (rm -rf, sudo, mkfs, dd, fdisk, etc.)
  - Block shell pipes (| bash, | sh, | zsh)
  - Block heredoc syntax to prevent unauthorized file writes
  - Block git operations (commit, push, rebase, etc.)
  - Add read-only mode support
  - Environment variable filtering to prevent credential leakage
- **Read-only mode** — toggle via `/readonly` chat command
  - Prevent all modifications when enabled
  - Block file writes and command execution that modifies state
  - Comprehensive protection via guard agent

### Removed

- **Ask tool** — removed from codebase
  - Delete internal/plugins/ask/plugin.go
  - Remove ask import from cmd/go-magnetar/main.go
  - Remove ask tool documentation from AGENTS.md
- **Search replace tool** — removed from generic plugin
  - Removed from internal/tools/generic/generic.go
  - Updated documentation in AGENTS.md, README.md, docs/user_manual.md
- **Guard ask flag** — simplified guard configuration
  - Removed ask flag from guard configuration (replaced with unified guard)

### Documentation

- **Security documentation** — new docs/security.md
  - Comprehensive security guidelines
  - Command safety guard details
  - Read-only mode explanation
  - Root user prevention
- **Updated documentation** — for new features
  - AGENTS.md, README.md, docs/user_manual.md updates
  - Command examples for /fetch and /readonly
  - Configuration parameters for language, temperature, top_p
  - Updated data flow diagrams
  - Added warning about stdin input limitations (Bubble Tea requires interactive terminal)

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

## [v0.1.1] - 2026-06-23

### Added

- **exec tool** - Execute shell commands via `sh -c` with clean environment and stdin support
- **Security restrictions** - Block list for dangerous commands (rm -rf, sudo, mkfs, dd, git operations, shell pipes)
- **system_date tool** - Execute the `date` command to get current system time
- **Read-only mode** - Prevent file modifications
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

[Unreleased]: https://github.com/wmentor/go-magnetar/compare/v0.1.4...HEAD
[v0.1.3]: https://github.com/wmentor/go-magnetar/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/wmentor/go-magnetar/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/wmentor/go-magnetar/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/wmentor/go-magnetar/releases/tag/v0.1.0