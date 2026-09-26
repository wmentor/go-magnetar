# CHANGELOG

## [1.3.0] - 2026-09-26

### 🚀 Features

- Add /profile chat command for profile switching
- Add full HTML support across all tools (file_read, indexing, web_fetch)
- Web_search tool integration

### 🐛 Bug Fixes

- *(profile)* Reconfigure OpenAI client when switching profiles (#47)
- Fix release script

### ⚙️ Miscellaneous Tasks

- *(templates)* Bug and feature request templates
- Fix changelog script

### 💼 Other

- *(release)* Using git cliff to generate CHANGELOG.md and a release page
- *(release)* Build release script
## [1.2.0] - 2026-09-21

### 🚀 Features

- Add clipboard copy command and refactor plugin system (#34)
- Add config versioning, include directive support, fix rag indexer, change config location (#37)

### 💼 Other

- Bump external dependencies
## [1.1.0] - 2026-09-18

### 🚀 Features

- Add CSV and XLSX file support
- Add SSH tool with optional user and dir parameters
- Add TSV file support in file reading

### 🚜 Refactor

- Disable external integrations by default in config

### 💼 Other

- Integrating a build project via goreleaser when submitting a tag (#30)
## [1.0.0] - 2026-09-13

### 🚀 Features

- Add -p/--profile flag to override configuration profile
- *(config)* Add $file:filename support for file content substitution
- Add GitHub Issues support to web_fetch and github_repo tool
- Add GitHub Milestones support to web_fetch and github_milestone tool
- Add .pptx support via internal/common package
- Add environment variable support to text preprocessor
- *(github)* Add security advisory support
- Add CVE lookup tool and update agent tools documentation
- Add session loading support and refactor chat commands documentation

### 🐛 Bug Fixes

- Potential infinite loop

### 🚜 Refactor

- Replace strings.Builder.WriteString with fmt.Fprintf for better efficiency
- Categorize tools as search vs lookup and improve loop protection
## [0.1.6] - 2026-09-01

### 🚀 Features

- Add -f/--file flag for non-interactive mode
- Add llm.reasoning_effort configuration parameter
- *(config)* Profile params
- Implement profile-based configuration with ProfileParam* helpers
- Add environment variable substitution support ($env:VAR_NAME)

### 🐛 Bug Fixes

- Bump go 1.27.0 . bump dependencies. go fix

### 📚 Documentation

- Add docs/configuration.md
- Update README.md and docs/user_manual.md
## [0.1.5] - 2026-07-29

### 🚀 Features

- Add DOCX file support to file_read tool, indexer, and preprocessor
- Add PDF support to file_read tool, indexer, and preprocessor
- Add ODT file support to file_read, indexer, and preprocessor
- Add ODT file support to documentation
## [0.1.4] - 2026-07-06

### 🚀 Features

- *(generic)* Add search_replace tool for regex-based file text replacement
- Refactor exec tool with security restrictions and stdin support
- Add security guard for exec commands with read-only mode support
- Add security hardening improvements
- Add /fetch chat command for URL content retrieval
- Add guard configuration options (disable, ask)

### 🐛 Bug Fixes

- Handle float64 args in file_read tool

### 📚 Documentation

- Add warning about stdin input in AGENTS.md

### 🚜 Refactor

- Remove ask tool from codebase

### 🛡️ Security

- Harden command execution with root prevention and expanded guard

### 💼 Other

- Add support for fetching child issues in Epic tasks
- Add allowed system commands
- Type cast
## [0.1.3] - 2026-06-30

### 🚀 Features

- Add jira_task_search tool with JQL support and pagination
- Add GitHub integration with repo/file/tree tools

### 🚜 Refactor

- Replace printer.Debug with printer.ToolCall in gitlab and rag
- Replace printer.Info with printer.Print for indexed file/URL logging

### 💼 Other

- Replace tput with golang.org/x/term for width detection
## [0.1.2] - 2026-06-28

### 🚀 Features

- Add GitLab merge request fetching support
- Add version command with enhanced display and timeout adjustment
- Add ask tool plugin for clarifying user questions
- Add JIRA plugin with jira_task_get LLM tool

### 🚜 Refactor

- *(config)* Rewrite Config to use koanf.Koanf directly
- Replace slog with internal/printer package

### ⚙️ Miscellaneous Tasks

- Update documentation to reflect codebase changes
## [0.1.1] - 2026-06-25

### 🚀 Features

- Add /readonly chat command
- Add read-only mode to prevent file modifications
- *(web)* Add JIRA issue support to web_fetch tool
- *(generic)* Add line-based file_read with limit/offset support
- *(generic)* Add system_grep tool with safety checks and /less availability check
- Add /idxtab command for batch indexing from JSON lines file
- *(system_exec)* Add system_exec tool with mode-based command permissions
- Add system_exec with mode-based permissions and system_date tool

### 📚 Documentation

- Update AGENTS.md, README.md, docs/user_manual.md

### 🚜 Refactor

- *(chat)* Replace inline history with internal/history package
- *(cli)* Replace subcommands with /index chat command

### 💼 Other

- Pass State by pointer instead of by value
- Add install target and improve version command
- Add /less command to view last answer with less
- Optimize New() for empty filename and add Records() method
- Add tests for Storage with t.Parallel
## [0.1.0] - 2026-06-23

### 🚀 Features

- Add Confluence page fetching support to web_fetch tool
## [0.0.3] - 2026-06-22

### 🚜 Refactor

- Plugin architecture and search enhancements
## [0.0.2] - 2026-06-21

### 🚀 Features

- *(indexer)* Add -m/--message flag to prepend custom text to chunks
- Add maximum search tool call limit to prevent infinite loops
- *(chat)* Add command history navigation with arrow keys

### 📚 Documentation

- Translate all docs to English and add webfetch config description

### ⚙️ Miscellaneous Tasks

- *(indexer)* Remove directory indexing mode, keep only file and URL
## [0.0.1] - 2026-06-20

### 🚀 Features

- Add /new command to start a new session and clear context
- Add file_write tool to generic tools
- Add file_exists tool and load AGENTS.md into system prompt

### 🐛 Bug Fixes

- Allow file_list extension filter without leading dot
