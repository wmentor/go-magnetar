# web_fetch Tool

The `web_fetch` tool fetches content from URLs and returns it in Markdown format. It supports standard web pages, Confluence pages, JIRA issues, GitLab merge requests, and GitHub repositories/issues/milestones.

## Overview

`web_fetch` is a search tool that intelligently handles different URL types by detecting the platform and using the appropriate API or content extraction method.

## Signature

```go
func WebFetch(url string) (string, error)
```

## Parameters

- `url` (string): The URL to fetch content from

## Supported URL Types

### 1. Standard Web Pages
Fetches HTML pages and converts them to clean Markdown:
- Removes navigation, headers, footers, ads, cookies banners
- Uses Readability to extract main content
- Converts HTML to Markdown

### 2. Confluence Pages
Detects Confluence URLs and fetches via API:
- Standard URL: `https://your-domain.atlassian.net/wiki/spaces/SPACE/pages/123456`
- Short link: `https://your-domain.atlassian.net/x/AgA5`
- Share link: `https://your-domain.atlassian.net/p/123456`

Returns: page title, space key, creation/update dates, author, version, and body content.

### 3. JIRA Issues
Detects JIRA URLs and fetches via API:
- URL pattern: `https://your-domain.atlassian.net/browse/PROJECT-123`
- Returns issue details, description, status, assignee, reporter, comments

### 4. GitLab Merge Requests
Fetches GitLab merge requests via API:
- URL pattern: `https://gitlab.com/namespace/project/-/merge_requests/123`
- Returns MR details, author, branches, labels, description
- Includes file changes with diffs

### 5. GitHub Repositories
Fetches repository information:
- URL pattern: `https://github.com/owner/repo`
- Returns: full name, default branch, license, language, stars, forks, open issues, topics, description, README

### 6. GitHub Files
Fetches file content from repositories:
- URL pattern: `https://github.com/owner/repo/blob/branch/path/to/file`
- Returns file content

### 7. GitHub Tree (Directory Listing)
Lists repository contents:
- URL patterns: `https://github.com/owner/repo/tree/branch/path`, `https://github.com/owner/repo/commits/branch`
- Returns file/directory listing with sizes

### 8. GitHub Issues
Fetches issues with comments:
- URL pattern: `https://github.com/owner/repo/issues/123`
- Returns issue details, labels, assignees, description, comments

### 9. GitHub Milestones
Fetches milestone with associated issues:
- URL pattern: `https://github.com/owner/repo/milestone/123`
- Returns milestone details, open/closed issue counts, list of issues

### 10. GitHub Security Advisories
Fetches repository security advisories:
- URL patterns: `https://github.com/owner/repo/security/advisories`, `https://github.com/owner/repo/advisories/GHSA-xxxx`
- Returns advisory details, CVE IDs, severity, affected packages

### 11. RSS Feeds
Detects and decodes RSS/Atom feeds:
- Content-Type: `application/rss+xml`, `text/xml`, `application/xml`
- Returns channel items as formatted text

## Processing Pipeline

For HTML content, `web_fetch` applies three preprocessing steps:

### 1. cleanHTML
Removes unwanted elements:
- **Tags**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<nav>`, `<header>`, `<footer>`, `<aside>`, `<button>`, `<svg>`, `<canvas>`, `<input>`, `<textarea>`
- **CSS selectors**: cookie banners, newsletter forms, social share buttons, advertisements, popups, modals, widgets, comments, navigation menus, tracking scripts

### 2. processReadability
Extracts main content using Readability algorithm:
- Identifies primary content area
- Removes boilerplate and clutter
- Preserves document structure

### 3. htmlToMarkdown
Converts HTML to Markdown:
- Uses html-to-markdown library
- Supports tables, lists, code blocks
- Falls back to plain text extraction if conversion fails

## Search Strategy

`web_fetch` is a **search tool** and follows the agent's search strategy:

1. **Unified search**: `search` tool calls both web_search and rag_search in parallel
2. **Fallback**: If `search` returns no relevant results, `web_fetch` may be used for external information
3. **Search limit**: Maximum 20 search tool calls per user request (`search` + `web_fetch`)

**Note:** When `search` is called, it counts as 1 search call (not 2 separate calls).

## Examples

### Fetch a web page
```go
web_fetch("https://example.com/article")
```

### Fetch a Confluence page
```go
web_fetch("https://mycompany.atlassian.net/wiki/spaces/DOC/pages/123456")
```

### Fetch a JIRA issue
```go
web_fetch("https://jira.example.com/browse/PROJECT-123")
```

### Fetch a GitLab merge request
```go
web_fetch("https://gitlab.com/group/project/-/merge_requests/456")
```

### Fetch a GitHub repository
```go
web_fetch("https://github.com/golang/go")
```

### Fetch a GitHub issue
```go
web_fetch("https://github.com/golang/go/issues/12345")
```

### Fetch a GitHub milestone
```go
web_fetch("https://github.com/golang/go/milestone/123")
```

### Fetch an RSS feed
```go
web_fetch("https://example.com/feed.xml")
```

## OpenAI Tool Schema

```json
{
  "type": "function",
  "function": {
    "name": "web_fetch",
    "description": "Fetch a web page and return clean Markdown content. Supports Confluence pages, JIRA issues, GitLab merge requests, and GitHub repositories, issues, and milestones.",
    "parameters": {
      "type": "object",
      "properties": {
        "url": {
          "type": "string",
          "description": "URL of the web page, Confluence page, JIRA issue, GitLab merge request, or GitHub repository to fetch"
        }
      },
      "required": ["url"]
    }
  }
}
```

## Error Handling

`web_fetch` returns errors for:
- Failed HTTP requests (network errors, timeouts)
- HTTP status codes other than 200
- Invalid URLs or malformed responses
- API authentication failures (for Confluence/JIRA/GitHub/GitLab)
- Unsupported URL types

Errors are logged and returned as strings prefixed with "error:".

## Timeout

Default timeout: 1 minute

## User Agent

Uses Chrome user agent:
```
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36
```

## Limitations

- GitHub/GitLab APIs may require authentication for rate limit limits
- Some sites may block automated scraping
- Very large files may be truncated
- JavaScript-heavy pages may not render correctly
