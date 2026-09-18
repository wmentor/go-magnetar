# Changelog

All notable changes to go-magnetar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.1.0] - 2026-09-18

### Added

- **Automated releases via goreleaser** — GitHub Actions workflow for CI/CD
- **SSH tool** — execute remote commands via SSH with key/password/agent auth
- **Spreadsheet support** — CSV, XLSX, and TSV file reading
- **Unified codec system** — single package for all file format readers

### Changed

- **Security by default** — all external integrations disabled unless explicitly enabled

## [v1.0.0] - 2026-09-13

### Added

- **Session management** — load/resume conversations with `--session` and `/save`
- **Security features** — GitHub security advisories, CVE lookup
- **GitHub integration** — fetch milestones and issues
- **Enterprise file formats** — DOCX, PDF, ODT, PPTX support
- **Config enhancements** — environment variables, profile-based settings
- **Chat commands** — `/index`, `/fetch`, `/idxtab`, `/readonly` with aliases
- **Tool categorization** — search vs lookup tools with loop protection

### Changed

- **Unified CLI** — single `agent` command replacing separate indexer/agent subcommands
- **Go 1.27.0** — minimum version requirement

### Removed

- **Indexer CLI** — functionality merged into `/index` chat command

## [v0.1.6] - 2026-09-01

### Added

- **Config profiles** — structured configuration via `profiles` block
- **Non-interactive mode** — `-f/--file` flag for scripting

### Changed

- **Default chunking** — optimized to 2048 runes with 256 overlap
- **Go 1.27.0** — minimum version requirement

## [v0.1.5] - 2026-07-29

### Added

- **Confluence/JIRA integration** — fetch issues and pages directly
- **Parallel tool calling** — concurrent LLM tool execution
- **Multiple preprocessor placeholders** — `{{file:filename}}` expansion

### Changed

- **Security hardening** — exec tool refactoring with blocklist

### Fixed

- **Build process** — `go fix`/`go fmt` integration

## [v0.1.4] - 2026-07-06

### Added

- **Language parameter** — configurable agent response language
- **LLM parameters** — temperature and top_p support
- **/fetch command** — URL content retrieval
- **JIRA Epic support** — fetch child issues

### Changed

- **Security** — comprehensive command blocklist and read-only mode

### Removed

- **Ask tool** and **search_replace** — replaced by unified guard agent

## [v0.1.3] - 2026-06-30

### Added

- **GitHub API integration** — repo/file/tree tools with URL detection

### Changed

- **Web fetch refactoring** — cleaner separation between handlers

## [v0.1.2] - 2026-06-28

### Added

- **GitLab MR fetching** — pull request changes via API
- **JIRA plugin** — issue fetching via REST API

### Changed

- **Config refactoring** — koanf-based configuration
- **Unified logging** — single output format across the codebase

### Removed

- **CLI subcommands** — indexer and agent merged

## [v0.1.1] - 2026-06-23

### Added

- **Security restrictions** — blocklist for dangerous commands
- **Read-only mode** — `/readonly` toggle
- **Unified /index** — document indexing from REPL

## [v0.1.0] - 2026-06-23

- Initial release with RAG, chat agent, web fetching, and plugin architecture

[Unreleased]: https://github.com/wmentor/go-magnetar/compare/v1.0.0...HEAD
[v1.1.0]: https://github.com/wmentor/go-magnetar/compare/v1.0.0...HEAD
[v1.0.0]: https://github.com/wmentor/go-magnetar/releases/tag/v1.0.0
