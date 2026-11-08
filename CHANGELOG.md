# CHANGELOG

## [1.9.0] - 2026-11-08

### 🚀 Features

- *(codecs)* Add Apple Pages document support (#300)
- *(tools)* Add docx write support to file_write tool (#265)
- *(indexer)* Index directories recursively with /index -r command (#244)
- *(agents)* Added subagents system (#279)
- *(chat)* Return list enabled agents when /agent has no arguments (#283)
- *(agents)* Implement synchronous agent execution mode (#287)
- *(config)* Add environment variable to config parameter mapping (#290)
- *(agents)* Add profile support for agents (#294)
- *(agents)* Agent-specific tools (#295)
- *(stat)* Add statistics tracking for file operations (chat command /stat) (#299)
- *(chat)* Add "Good bye!" message to exit command response (#259)

### 🐛 Bug Fixes

- *(chat)* Panic /confluence.edit chat command (#264)
- *(config)* Fix environment variable mapping for fields with underscores (#292)

### 🛡️ Security

- *(codecs)* Add path traversal protection for file write codec (#271)
- *(dependencies)* Bumps github.com/xuri/excelize/v2 to v2.11.1 (fixed GO-2026-6684, GO-2026-6685, GO-2026-6688, GO-2026-6689) (#272)
- *(agents)* Add directory access restrictions and improve security constraints (#293)
- *(guard)* Block GitHub and GitLab CLI commands (#297)

### 📚 Documentation

- *(tools)* Update file_write tool description (#261)
- *(codecs)* Update file formats table with read/write support status (#270)
- *(common)* Added CODE_OF_CONDUCT.md (#277)
- *(agents)* Add documentation for subagents feature (#288)
- *(common)* Update architecture.md with new interfaces and components (#301)

### 🧹 Refactor

- *(agents)* Improve agent tool logging and output formatting (#289)

### 🎨 Styling

- *(icons)* Replace generic tool icon with context-specific icons (#298)

### 🧪 Testing

- *(plugins)* Add unit tests for GitHub plugin (#268)
- *(plugins)* Add confluence plugin unit tests (#269)
- *(plugins)* Add unit tests for file plugin initialization and registration (#274)
- *(tools)* Add comprehensive unit tests for github formatTreeMarkdown (#278)
- *(plugins)* Add unit tests for jira plugin (#286)
- *(plugins)* Added rag plugin unit tests (#291)
- *(plugins)* Fix rag plugin test (#296)
- *(plugins)* Add gitlab plugin unit tests (#302)
- *(plugins)* Add unit tests for CVE plugin (#303)
- *(plugins)* Add plugin unit test for /new chat command (#304)

### ⚙️ Miscellaneous Tasks

- *(utils)* Add .rgignore to ignore generated go files in testdata directories (#276)

### 👥 Contributors

<div align="left" style="padding-left: 20px;">
<a href="https://github.com/wmentor" data-hovercard-type="user" data-hovercard-url="/users/wmentor/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/wmentor?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@wmentor" title="@wmentor" style="border-radius: 50% !important;"></a> <a href="https://github.com/sshnox" data-hovercard-type="user" data-hovercard-url="/users/sshnox/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/sshnox?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@sshnox" title="@sshnox" style="border-radius: 50% !important;"></a> <a href="https://github.com/spayder" data-hovercard-type="user" data-hovercard-url="/users/spayder/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/spayder?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@spayder" title="@spayder" style="border-radius: 50% !important;"></a> <a href="https://github.com/Abhishek8841" data-hovercard-type="user" data-hovercard-url="/users/Abhishek8841/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/Abhishek8841?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@Abhishek8841" title="@Abhishek8841" style="border-radius: 50% !important;"></a> 
</div><br/><br/>

## [1.8.0] - 2026-11-01

### 🚀 Features

- *(search)* Add ripgrep (rg) support as preferred search tool if available
- *(codec)* Add EML file support
- *(summarizer)* Support configurable compact.mode strategy (#216)
- *(security)* Add NIST adapter support for CVE identifiers (#220)
- *(security)* Add rd, rmdir, reboot, shutdown, del, assoc to command blocklists (#207)
- *(web)* Add rate limiting for NIST and web search APIs (#242)
- *(stat)* Add HTTP request metrics (#250)

### 🐛 Bug Fixes

- *(cve)* Cover nil and markdown formatting (#208)
- *(web)* Remove stray fmt.Println on MIME codec decode error (#213)
- *(web)* Add https to scheme-less URLs (#224)
- *(chat)* Prevent infinite loop by truncating message history (#243)
- *(skills)* Enhance commit-message skill instructions (#241)

### 📚 Documentation

- *(docs)* Update README with new features and tool documentation (#218)

### 🧹 Refactor

- *(security)* Move security patterns to interceptor package
- *(rag)* Centralize embedding client initialization (#215)
- *(tools)* Introduce unified tool architecture with plugin registration (#217)
- *(tools)* Reorganize web tools into separate packages
- *(tools)* Split monolithic Jira tools into separate tool types (#225)
- *(tools)* Extract tools and remove ConfluenceTools legacy code (#226)
- *(tools)* Split GitLab tools into separate packages and support new architecture (#227)
- *(tools)* Migrate github tools to tool store API (#228)
- *(tools)* Replace RAGTools with Tool interface for rag_search (#239)
- *(tools)* Refactor last_response plugin (#240)
- *(tools)* Remove legacy string APIs from websearch (#246)
- *(tools)* Remove deprecated file_edit tool schema functions (#247)
- *(tools)* Remove unused OpenAI tool definitions in github tools (#248)
- *(agent)* Simplify tool execution logic and remove old LLMTool abstraction (#249)
- *(tools)* Remove IsSearchTool flag from tool structure (#252)
- *(tools)* Remove static tool definitions from confluence and web packages (#253)

### 🧪 Testing

- *(generic)* Add unit tests for FileList method (#214)
- *(web)* Update tests to use new web tool structure
- *(web)* Avoid assuming tool map order (#223)

### 💼 Other

- *(security)* Bumps Go version to 1.27.3 (#229)
- *(security)* Update dependencies (#230)

### 👥 Contributors

<div align="left" style="padding-left: 20px;">
<a href="https://github.com/wmentor" data-hovercard-type="user" data-hovercard-url="/users/wmentor/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/wmentor?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@wmentor" title="@wmentor" style="border-radius: 50% !important;"></a> <a href="https://github.com/duongynhi000005-oss" data-hovercard-type="user" data-hovercard-url="/users/duongynhi000005-oss/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/duongynhi000005-oss?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@duongynhi000005-oss" title="@duongynhi000005-oss" style="border-radius: 50% !important;"></a> <a href="https://github.com/Vamsi-2581" data-hovercard-type="user" data-hovercard-url="/users/Vamsi-2581/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/Vamsi-2581?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@Vamsi-2581" title="@Vamsi-2581" style="border-radius: 50% !important;"></a> <a href="https://github.com/udayx-dev" data-hovercard-type="user" data-hovercard-url="/users/udayx-dev/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/udayx-dev?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@udayx-dev" title="@udayx-dev" style="border-radius: 50% !important;"></a> <a href="https://github.com/SatriaBagusNurhadianto" data-hovercard-type="user" data-hovercard-url="/users/SatriaBagusNurhadianto/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/SatriaBagusNurhadianto?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@SatriaBagusNurhadianto" title="@SatriaBagusNurhadianto" style="border-radius: 50% !important;"></a> <a href="https://github.com/dannyward630" data-hovercard-type="user" data-hovercard-url="/users/dannyward630/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/dannyward630?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@dannyward630" title="@dannyward630" style="border-radius: 50% !important;"></a> 
</div><br/><br/>

## [1.7.0] - 2026-10-25

### 🚀 Features

- *(chat)* Fallback to CLAUDE.md when AGENTS.md is not found
- *(tools)* Add file_edit and file_edit_multiple tools
- *(chat)* Add /exec command for shell command execution
- *(tools)* Drop unused tool file_exists
- *(chat)* Add /revert command to undo last conversation turn
- *(tools)* Add last_response tool
- *(github)* Add support for fetching GitHub releases
- *(chat)* Command /copy now supports both LLM responses and chat command results
- *(guard)* Add allow utilities configuration for security checks
- *(chat)* Add /timing and --enable-timing to show execution time
- *(cli)* Add --skill flag to apply skills at startup
- *(tools)* Replace unified search tool with independent web_search and rag_search tools
- *(tools)* Add transparent gzip decompression to file_read
- *(web)* Add web.disable.fetch and web.disable.search settings
- *(confluence)* Add support for Confluence macros and fix HTML handling

### 🐛 Bug Fixes

- *(help)* Return markdown through command output
- *(chat)* Render command /profile output as markdown
- *(tools)* Register system_date without requiring the date utility
- *(build)* Correct gofmpt to gofumpt typo in Makefile
- *(tools)* Validate file size before reading contents
- *(config)* Update the profile key in SetProfile
- *(printer)* Make the printer and the default printer safe for concurrent use
- *(tools)* Added text about gzip support in file_read tool description
- *(chat)* Return /readonly command output in chat response
- *(network)* Fix OpenAI client memory leaks
- *(chat)* Write returns diagnostics as markdown in ChatCommandResponse.Stdout, with tests

### 📚 Documentation

- Added CONTRIBUTING.md
- Added faq.md

### 🚜 Refactor

- Add NoLog flag to ChatCommandResponse
- *(tools)* Add output showing what file_edit changed
- *(chat)* Command /stat outputs a Markdown table
- Flexible types for tool calls
- New tools storage prototype

### 💼 Other

- *(release)* Constrain GoReleaser to v2

### 👥 Contributors

<div align="left" style="padding-left: 20px;">
<a href="https://github.com/wmentor" data-hovercard-type="user" data-hovercard-url="/users/wmentor/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/wmentor?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@wmentor" title="@wmentor" style="border-radius: 50% !important;"></a> <a href="https://github.com/bercedev" data-hovercard-type="user" data-hovercard-url="/users/bercedev/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/bercedev?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@bercedev" title="@bercedev" style="border-radius: 50% !important;"></a> <a href="https://github.com/Abhishek8841" data-hovercard-type="user" data-hovercard-url="/users/Abhishek8841/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/Abhishek8841?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@Abhishek8841" title="@Abhishek8841" style="border-radius: 50% !important;"></a> <a href="https://github.com/yuefdev" data-hovercard-type="user" data-hovercard-url="/users/yuefdev/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/yuefdev?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@yuefdev" title="@yuefdev" style="border-radius: 50% !important;"></a> <a href="https://github.com/rootsiemen" data-hovercard-type="user" data-hovercard-url="/users/rootsiemen/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/rootsiemen?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@rootsiemen" title="@rootsiemen" style="border-radius: 50% !important;"></a> <a href="https://github.com/montionugera" data-hovercard-type="user" data-hovercard-url="/users/montionugera/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/montionugera?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@montionugera" title="@montionugera" style="border-radius: 50% !important;"></a> <a href="https://github.com/masyarakatai" data-hovercard-type="user" data-hovercard-url="/users/masyarakatai/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/masyarakatai?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@masyarakatai" title="@masyarakatai" style="border-radius: 50% !important;"></a> <a href="https://github.com/LetMeSleep8h" data-hovercard-type="user" data-hovercard-url="/users/LetMeSleep8h/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/LetMeSleep8h?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@LetMeSleep8h" title="@LetMeSleep8h" style="border-radius: 50% !important;"></a> <a href="https://github.com/Jah-yee" data-hovercard-type="user" data-hovercard-url="/users/Jah-yee/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/Jah-yee?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@Jah-yee" title="@Jah-yee" style="border-radius: 50% !important;"></a> <a href="https://github.com/Doomguy-BFG9000" data-hovercard-type="user" data-hovercard-url="/users/Doomguy-BFG9000/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/Doomguy-BFG9000?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@Doomguy-BFG9000" title="@Doomguy-BFG9000" style="border-radius: 50% !important;"></a> <a href="https://github.com/bxanedot" data-hovercard-type="user" data-hovercard-url="/users/bxanedot/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/bxanedot?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@bxanedot" title="@bxanedot" style="border-radius: 50% !important;"></a> <a href="https://github.com/abdulmuhaimin-abualuruq" data-hovercard-type="user" data-hovercard-url="/users/abdulmuhaimin-abualuruq/hovercard" data-octo-click="hovercard-link-click" data-octo-dimensions="link_type:self" aria-keyshortcuts="Alt+ArrowUp"><img src="https://avatars.githubusercontent.com/abdulmuhaimin-abualuruq?v=4&s=64" width="32" height="32" data-view-component="true" class="avatar-user" alt="@abdulmuhaimin-abualuruq" title="@abdulmuhaimin-abualuruq" style="border-radius: 50% !important;"></a> 
</div><br/><br/>

## [1.6.0] - 2026-10-18

### 🚀 Features

- *(cli)* Support `-` as stdin input for --file / -f flag
- *(codec)* Add MIME type support for web fetch and document decoding
- *(chat)* Allow /copy [index] to select a prior assistant answer
- *(skill)* Add commit-message skill
- *(jira)* Add Jira issue creation capability
- *(gitlab)* Add support for GitLab repositories, files, and tree structures
- *(skill)* Add bug-hunter skill for detecting common bugs
- *(confluence)* Add support for tables in posts
- *(github)* Add support for fetching GitHub pull requests

### 🐛 Bug Fixes

- *(confluence)* Properly escape horizontal rules in Markdown-to-HTML conversion
- *(confluence)* Decode short page IDs with URL-safe characters
- *(web)* Extract short GitLab /-/mr/ merge request URLs
- *(tools)* Stop replaceAllPatterns from looping on unchanged placeholders
- *(jira)* Add unit tests and fix HTML conversion regex patterns

### 🚜 Refactor

- *(chat)* Introduce ChatCommandInput struct for command execution
- *(printer)* Convert Printer to interface and add ConsolePrinter implementation
- *(tools)* Use time package in SystemDate instead of the date command
- *(build)* Reorganize Makefile targets and improve code formatting
- *(chat)* Centralize answer printing and add stdout support

### 🧪 Testing

- *(web)* Cover and simplify JIRA URL extraction
- *(tools)* Add unit tests for processFiles
- *(tools)* Cover unreadable files and empty names in processFiles

### ⚙️ Miscellaneous Tasks

- *(build)* Bumps mockery to v3.8.1
- *(release)* Use archives.formats for GoReleaser v2
- *(skill)* Improve make-issue skill instructions
- *(ci)* Add lefthook to dev tools setup
- *(ci)* Enable errorlint and use errors.Is for io.EOF checks
- *(skill)* Add code style and unit testing guidelines to AGENTS.md
- *(release)* Append contributor list with avatars to CHANGELOG.md

### 👥 Contributors

<div align="left" style="padding-left: 20px;">
<a href="https://github.com/wmentor"><img src="https://avatars.githubusercontent.com/wmentor?v=4&s=64" width="32" height="32" alt="@wmentor" title="@wmentor" style="border-radius:50%"></a> <a href="https://github.com/spayder"><img src="https://avatars.githubusercontent.com/spayder?v=4&s=64" width="32" height="32" alt="@spayder" title="@spayder" style="border-radius:50%"></a> <a href="https://github.com/Detoy"><img src="https://avatars.githubusercontent.com/Detoy?v=4&s=64" width="32" height="32" alt="@Detoy" title="@Detoy" style="border-radius:50%"></a> <a href="https://github.com/pollychen-lab"><img src="https://avatars.githubusercontent.com/pollychen-lab?v=4&s=64" width="32" height="32" alt="@pollychen-lab" title="@pollychen-lab" style="border-radius:50%"></a> <a href="https://github.com/himanshu748"><img src="https://avatars.githubusercontent.com/himanshu748?v=4&s=64" width="32" height="32" alt="@himanshu748" title="@himanshu748" style="border-radius:50%"></a> 
</div><br/><br/>

## [1.5.0] - 2026-10-11

### 🚀 Features

- Chat command /idxtab now supports configuration filenames starting with tilde
- Add Chromem vector store as default for RAG
- Add unified search tool that combines WEB and RAG search
- Implementation skills storage, loading skill logic and /skill chat command

### 🐛 Bug Fixes

- Expand ~ in /fetch output filename
- *(confluence_page_edit)* Correct HTML body generation
- Invert ssh.enable check in SSHExec
- Show ssh and guard in enabled plugins on startup

### 📚 Documentation

- Add Data Residency documentation

### 🚜 Refactor

- *(rag)* Restructure architecture with interface-based design
- Changed ChatCommand.Execute to return response object

### 💼 Other

- Update Go version to 1.27.2
## [1.4.0] - 2026-10-02

### 🚀 Features

- Add Confluence page editing support via /confluence.edit command and confluence_page_edit tool
- Add confluence.readonly config parameter for disabling page editing
- Add --read-only CLI flag to active read-only mode
- Add RSS feed support to web_fetch tool

### 🚜 Refactor

- Extract Confluence tools to dedicated package

### ⚙️ Miscellaneous Tasks

- Add golangci-lint configuration and code refactor
- Added pull request flow
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
