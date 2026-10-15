# JIRA Tools

go-magnetar provides the following tools for interacting with JIRA:

All JIRA tools use the [JIRA REST API v2](https://docs.atlassian.com/software/jira/docs/api/rest/) to interact with JIRA Server/Data Center and Cloud instances.

## jira_task_get

Fetches a JIRA issue and returns its details in Markdown format.

### Parameters

| Parameter | Type | Description |
|---|---|---|
| `issue_key` | string | JIRA issue key (e.g., GOARCH-60, TASK-123) |

### Response

Returns a detailed Markdown representation of the issue including:
- Issue key and ID
- Summary and description
- Status, assignee, reporter
- Project and issue type
- Created/updated timestamps
- Labels, parent issue
- Comments (if any)
- Related issues (for Epics)

## jira_task_search

Searches JIRA issues using a JQL query and returns an array of issue keys and summaries with pagination support (using JIRA REST API v2 `/rest/api/2/search` endpoint).

### Parameters

| Parameter | Type | Description |
|---|---|---|
| `jql` | string | JIRA Query Language (JQL) search query (e.g., `project = GOARCH AND status = Open`) |
| `start_at` | integer | Starting index for pagination (default: 0) |
| `max_results` | integer | Maximum number of results to return (default: 100) |

### Response

Returns a JSON array of issues with the following structure:

```json
[
  {
    "key": "TASK-123",
    "summary": "Issue summary text"
  },
  {
    "key": "TASK-124",
    "summary": "Another issue summary"
  }
]
```

## jira_issue_create

Creates a new JIRA issue with the specified project, summary, description, and issue type. Description is converted from Markdown to Jira format automatically.

### Parameters

| Parameter | Type | Description |
|---|---|---|
| `project_key` | string | JIRA project key (e.g., TASK, GOARCH) |
| `summary` | string | Issue summary (short description) |
| `description` | string | Issue description in Markdown format |
| `issue_type` | string | Issue type name (e.g., Bug, Task, Story, Epic) |

### Response

Returns the created issue details in the following format (using JIRA REST API v2 `/rest/api/2/issue` endpoint):

```json
{
  "id": "163355",
  "key": "TASK-54",
  "self": "https://jira.postgrespro.ru/rest/api/2/issue/163355"
}
```

### Markdown to Jira Conversion

The description is automatically converted from Markdown to Jira markup format:

| Markdown | Jira |
|---|---|
| `**bold**` | `*bold*` |
| `*italic*` | `_italic_` |
| `# Heading 1` | `h1. Heading 1` |
| `## Heading 2` | `h2. Heading 2` |
| `- item` | `* item` |
| `1. item` | `# item` |
| `[text](url)` | `[text\|url]` |
| ```go\nfunc()``` | `{code:go}\nfunc()\n{code}` |
| `> quote` | `{quote}\nquote\n{quote}` |
| `| Header |` | `||Header||` |

See the [full conversion reference](https://github.com/wmentor/md2jira) for more details.

## Configuration

Enable JIRA integration in your config file:

The JIRA tools use the REST API v2 endpoints:
- Issue creation: `POST /rest/api/2/issue`
- Issue fetch: `GET /rest/api/2/issue/{id}`
- Search: `POST /rest/api/2/search`

```yaml
jira:
  enable: true
  base_url: https://jira.example.com
  api_key: $env:JIRA_API_KEY
  readonly: false  # Set to true to disable issue creation
```

## Usage Examples

### Fetch a specific issue

```bash
# LLM tool call
jira_task_get('GOARCH-60')
```

### Search issues with JQL

```bash
# LLM tool call
jira_task_search('project = TASK AND status = "In Progress"', 0, 50)
```

### Create a new issue

```bash
# LLM tool call
jira_issue_create({
  "project_key": "TASK",
  "summary": "Implement new feature",
  "description": "## Task\n\nImplement a new feature in the project.\n\n### Steps\n1. Analyze requirements\n2. Design solution\n3. Implement and test",
  "issue_type": "Task"
})
```
