# Agent Tools

go-magnetar provides the following tools for the chat agent:

| Tool | Signature | Description |
|---|---|---|
| `file_read` | `(filename: string, limit: int, offset: int) -> string` | Reads file contents from the filesystem; supports `.txt`, `.md`, `.docx`, `.pdf`, `.odt`, `.pptx`; `limit` and `offset` specify line range (0 = read all) |
| `file_list` | `(filter: string) -> []string` | Recursively lists files in the current directory using glob pattern (e.g. `*.go`) |
| `file_write` | `(filename: string, content: string) -> bool` | Writes content to a file in the filesystem (blocked in read-only mode) |
| `exec` | `(command: string, stdin: string) -> string` | Executes a shell command via `sh -c` with clean environment, current working directory, and built-in safety guard |
| `system_date` | `() -> string` | Executes the date command to get the current system time |
| `system_grep` | `(filename: string, pattern: string) -> string` | Executes system grep command with safe parameters: -n (always), -i (case-insensitive), -r (recursive), -E (extended regex) |
| `rag_search` | `(query: string) -> string` | Returns top-N relevant fragments from Qdrant (N is set by `rag.search.limit`) |
| `web_fetch` | `(url: string) -> string` | Fetches and cleans a web page (fallback if RAG returns no results); also fetches Confluence pages, JIRA issues, and GitHub repositories, issues, and milestones |
| `cve` | `(id: string) -> string` | Fetches vulnerability information from OSV database; supports all OSV database identifiers (CVE-, GO-, GHSA-, OSV-, GSD-, ALPINE-, and 50+ more). See [vulnerability_lookup.md](./vulnerability_lookup.md) for complete documentation |
| `github_repo` | `(repo: string) -> string` | Fetches GitHub repository information and returns its details in Markdown format |
| `github_file` | `(repo: string, branch: string, file: string) -> string` | Fetches a file from GitHub repository and returns its content |
| `github_tree` | `(repo: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |
| `github_issue` | `(repo: string, issue: string) -> string` | Fetches a GitHub issue and its comments, returns issue details in Markdown format |
| `github_milestone` | `(repo: string, milestone: string) -> string` | Fetches a GitHub milestone and returns its details in Markdown format |

## Search strategy

The agent **always** first calls `rag_search`, even if it believes it already knows the answer. If `rag_search` returns relevant results, the answer is formed exclusively based on those results, `web_fetch` is not called. `web_fetch` is used only as a fallback: when `rag_search` returns no relevant results and the user needs external or up-to-date information. If neither tool provides a result, the agent explicitly states this.

### Search tool call limit

To prevent infinite loops, each user request is limited to a maximum number of search-related tool calls (`rag_search` + `web_fetch`). By default, the limit is 10 calls per request. When the limit is exceeded, an error message is sent to the LLM and no more search tools are invoked for that request.
