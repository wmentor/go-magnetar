# Agent Tools

go-magnetar provides the following tools for the chat agent:

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem; supports `.txt`, `.md`, `.csv`, `.tsv`, `.docx`, `.pdf`, `.odt`, `.pptx`, `.xlsx`, `.html`, `.htm`; `limit` and `offset` specify line range (0 = read all). See [docs/file_codecs.md](./file_codecs.md) for complete format support and codec details |
| `file_list` | `(filter: string) -> []string` | Recursively lists files in the current directory using glob pattern (e.g. `*.go`) |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem (blocked in read-only mode) |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment, current working directory, and built-in safety guard |
| `ssh` | `(command: string, stdin: string, addr: string, user: string, dir: string) -> string` | Execute a shell command on a remote server via SSH. Connection parameters loaded from config. `user` and `dir` are optional; if not provided, uses `ssh.user` and `ssh.remote_dir` from config |
| `system_date` | `() -> string` | Executes the date command to get the current system time |
| `system_grep` | `(filename: string, pattern: string) -> string` | Executes system grep command with safe parameters: -n (always), -i (case-insensitive), -r (recursive), -E (extended regex) |
| `web_fetch` | `(url: string) -> string` | Fetches a web page and returns Markdown content (also fetches Confluence pages, JIRA issues, GitHub repositories/Issues/milestones, GitLab merge requests, RSS feeds). See [tool_web_fetch.md](./tool_web_fetch.md) for complete documentation |
| `search` | `(query: string) -> string` | Execute web and knowledge base search in parallel and return merged results |
| `cve` | `(id: string) -> string` | Fetches vulnerability information from OSV database; supports all OSV database identifiers (CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, and 50+ more). See [vulnerability_lookup.md](./vulnerability_lookup.md) for complete documentation |
| `github_repo` | `(repo: string) -> string` | Fetches GitHub repository information and returns its details in Markdown format |
| `github_file` | `(repo: string, branch: string, file: string) -> string` | Fetches a file from GitHub repository and returns its content |
| `github_tree` | `(repo: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |
| `github_issue` | `(repo: string, issue: string) -> string` | Fetches a GitHub issue and its comments, returns issue details in Markdown format |
| `github_milestone` | `(repo: string, milestone: string) -> string` | Fetches a GitHub milestone and returns its details in Markdown format |
| `confluence_page_edit` | `(page_id: string, markdown: string) -> string` | Edits a Confluence page by converting Markdown to HTML and updating via REST API |

## Search strategy

The agent calls the unified `search` tool, which executes `web_search` and `rag_search` in parallel and merges the results. When the `search` tool returns relevant results, the answer is formed exclusively based on those results. If the `search` tool provides no relevant results, the agent explicitly states this.

### Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls. The unified `search` tool counts as 1 search call (not 2 separate calls). By default, the limit is 20 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.

See [chat_agent_ask.md](./chat_agent_ask.md) for complete documentation on the Ask method, including tool categorization, protection mechanisms, and full algorithm description.
