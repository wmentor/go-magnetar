# Agent Tools

go-magnetar provides the following tools for the chat agent:

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem; supports `.txt`, `.md`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, `.html`, `.htm`; `limit` and `offset` specify line range (0 = read all). See [docs/file_codecs.md](./file_codecs.md) for complete format support and codec details |
| `file_list` | `(filter: string) -> []string` | Recursively lists files in the current directory using glob pattern (e.g. `*.go`) |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem (blocked in read-only mode) |
| `file_edit` | `(filename: string, old_text: string, new_text: string) -> string` | Edits a **plain-text** file by replacing text. **CRITICAL**: `old_text` must appear EXACTLY ONCE in the file. Returns error if `old_text` appears multiple times. Case-sensitive search. Not supported for `.docx`, `.pdf`, `.odt`, `.xlsx`, `.pptx`, `.csv`, `.tsv`, `.html`, `.htm` — use `file_write` for those. |
| `file_edit_multiple` | `(filename: string, edits: [{old_text, new_text}], strategy: "first"\|"all") -> string` | Edits a **plain-text** file by applying multiple text replacements. Strategy `"all"` (default) replaces all occurrences; `"first"` replaces only the first. Not supported for codec-handled formats (same as `file_edit`). |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment, current working directory, and built-in safety guard |
| `ssh` | `(command: string, stdin: string, addr: string, user: string, dir: string) -> string` | Execute a shell command on a remote server via SSH. Connection parameters loaded from config. `user` and `dir` are optional; if not provided, uses `ssh.user` and `ssh.remote_dir` from config |
| `system_date` | `() -> string` | Returns the current date in the "Mon Jan _2 15:04:05 MST 2006" format |
| `system_grep` | `(filename: string, pattern: string) -> string` | Executes system grep command with safe parameters: -n (always), -i (case-insensitive), -r (recursive), -E (extended regex) |
| `web_fetch` | `(url: string) -> string` | Fetches a web page and returns Markdown content (also fetches Confluence pages, JIRA issues, GitHub repositories/Issues/milestones, GitLab merge requests, RSS feeds). See [tool_web_fetch.md](./tool_web_fetch.md) for complete documentation |
| `web_search` | `(query: string) -> string` | Execute web search and return results in Markdown format (e.g., for current events, news, popular topics) |
| `rag_search` | `(query: string) -> string` | Search the knowledge base for relevant information (e.g., for domain-specific questions) |
| `cve` | `(id: string) -> string` | Fetches vulnerability information from OSV database; supports all OSV database identifiers (CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, and 50+ more). See [vulnerability_lookup.md](./vulnerability_lookup.md) for complete documentation |
| `github_repo` | `(repo: string) -> string` | Fetches GitHub repository information and returns its details in Markdown format |
| `github_file` | `(repo: string, branch: string, file: string) -> string` | Fetches a file from GitHub repository and returns its content |
| `github_tree` | `(repo: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |
| `github_issue` | `(repo: string, issue: string) -> string` | Fetches a GitHub issue and its comments, returns issue details in Markdown format |
| `github_milestone` | `(repo: string, milestone: string) -> string` | Fetches a GitHub milestone and returns its details in Markdown format |
| `github_pull_request` | `(repo: string, pr: string) -> string` | Fetches a GitHub pull request with all details including files changed, comments, and reviews |
| `github_release_info` | `(repo: string, tag: string) -> string` | Fetches a GitHub release by tag name and returns its details in Markdown format, including changelog and assets |
| `gitlab_fetch_mr` | `(project_path: string, mr_id: string) -> string` | Fetches a GitLab merge request and returns its details in Markdown format. See [plugin_gitlab.md](./plugin_gitlab.md) for complete documentation |
| `gitlab_repo` | `(project_path: string) -> string` | Fetches GitLab repository information and returns its details in Markdown format. See [plugin_gitlab.md](./plugin_gitlab.md) for complete documentation |
| `gitlab_file` | `(project_path: string, branch: string, file: string) -> string` | Fetches a file from GitLab repository and returns its content. See [plugin_gitlab.md](./plugin_gitlab.md) for complete documentation |
| `gitlab_tree` | `(project_path: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path. See [plugin_gitlab.md](./plugin_gitlab.md) for complete documentation |
| `confluence_page_edit` | `(page_id: string, markdown: string) -> string` | Edits a Confluence page by converting Markdown to HTML and updating via REST API |
| `jira_issue_create` | `(project_key: string, summary: string, description: string, issue_type: string) -> object` | Creates a new JIRA issue with Markdown-to-Jira conversion (blocked in read-only mode). See [tool_jira.md](./tool_jira.md) for complete documentation |

## File editing tools

### file_edit behavior

The `file_edit` tool performs exact string replacement with strict requirements:

- **Plain-text only**: Works only with plain-text files (`.go`, `.py`, `.js`, `.ts`, `.md`, `.yaml`, `.json`, `.txt`, etc.). Files with a registered codec — `.docx`, `.pdf`, `.odt`, `.xlsx`, `.pptx`, `.csv`, `.tsv`, `.html`, `.htm` — are rejected with an error because their binary/structured format cannot be round-tripped through text replacement. Use `file_write` or a dedicated tool for those formats.
- **Uniqueness**: `old_text` must appear exactly once in the file. If it appears multiple times, the tool returns an error asking you to use `file_edit_multiple` or provide more context to identify a unique occurrence.
- **Case-sensitive**: The search is case-sensitive. "Hello" will NOT match "hello".
- **Exact match**: All whitespace, special characters, and formatting must match exactly.

Use `file_edit` when you need to make a single, precise replacement in an existing plain-text file.

### file_edit_multiple behavior

The `file_edit_multiple` tool applies multiple text replacements in one call:

- **Plain-text only**: Same restriction as `file_edit` — codec-handled formats (`.docx`, `.pdf`, `.odt`, `.xlsx`, `.pptx`, `.csv`, `.tsv`, `.html`, `.htm`) are not supported.
- **"all" strategy** (default): Replaces ALL occurrences of each `old_text`.
- **"first" strategy**: Replaces only the FIRST occurrence of each `old_text`.

Use `file_edit_multiple` when:
- You need to make multiple different replacements in one file.
- The string you want to replace appears multiple times and you want to replace all of them.
- You want to replace only the first occurrence of multiple strings.

### Sequential execution guarantee

`file_write`, `file_edit`, and `file_edit_multiple` are marked `Sequential: true` in the agent's tool execution loop. This means:

- They are **never executed in parallel** with each other or with other tools in the same LLM response.
- They run **one at a time**, in the order the LLM listed them, after all parallel tools (reads, searches) in the same batch have finished.
- Each call sees the fully up-to-date filesystem state written by the previous call.

This prevents the lost-write race condition where two concurrent edits both read the original file and the second overwrites the result of the first.

See [chat_agent_ask.md](./chat_agent_ask.md) for details on tool execution categories.

### Guidelines

1. Use `file_read` first to see the file contents before editing
2. Use `file_edit` for single, unique replacements
3. Use `file_edit_multiple` with "all" for replacing multiple occurrences
4. Use `file_edit_multiple` with "first" for replacing only the first occurrence of multiple strings
5. Use `file_write` when you need to create a new file or completely rewrite an existing one

## Search Tools

go-magnetar provides two independent search tools: `web_search` and `rag_search`. Each tool can be called separately or together based on the query requirements.

### web_search

Execute web search and return results in Markdown format. Use `web_search` for:
- Time-sensitive queries (current events, news, popular topics)
- General knowledge questions
- Information not in the knowledge base

### rag_search

Search the knowledge base for relevant information. Use `rag_search` for:
- Domain-specific questions
- Questions about documents in the knowledge base
- Historical or archived information

### Search Strategy

The agent can choose to call either `web_search`, `rag_search`, or both tools based on the query requirements:

- For time-sensitive or general knowledge queries: use `web_search`
- For domain-specific questions: use `rag_search`
- For comprehensive answers: use both tools in parallel

When results from either tool are relevant, the answer is formed exclusively based on those results. If both tools provide no relevant results, the agent explicitly states this.

### Search Tool Call Limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls. Each search tool (`web_search` and `rag_search`) counts as 1 search call independently. By default, the limit is 20 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.

See [chat_agent_ask.md](./chat_agent_ask.md) for complete documentation on the Ask method, including tool categorization, protection mechanisms, and full algorithm description.
