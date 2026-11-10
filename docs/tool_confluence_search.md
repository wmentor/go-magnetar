# Confluence Search Tool

The `confluence_search` tool allows you to search Confluence pages using CQL (Confluence Query Language) and return results in Markdown format.

## Configuration

To use the `confluence_search` tool, you need to configure Confluence integration in your config file:

```yaml
confluence:
  enable: true
  base_url: "https://your-domain.atlassian.net/wiki"
  access_key: $env:CONFLUENCE_ACCESS_KEY
```

## Usage

The tool accepts the following parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | string | Search query string in CQL syntax (e.g., `text ~ "API" and type=page`) |
| `limit` | int | Maximum number of results to return (default: 25) |

## Example Queries

### Search for pages containing a specific term
```cql
text ~ "API"
```

### Search for pages containing "API" in a specific space
```cql
text ~ "API" and space = "DEV"
```

### Search for pages and order by last modified date
```cql
text ~ "API" and type=page order by lastmodified desc
```

### Search for pages with "performance" in the title
```cql
title ~ "performance"
```

### Combine multiple conditions
```cql
(text ~ "API" or text ~ "documentation") and type=page
```

## Response Format

The tool returns results in Markdown format with the following information for each result:

- Title
- Full URL (with base URL prepended)
- Excerpt
- Page ID
- Space
- Type
- Status
- Last Modified date
- Container title

## Examples

```go
// Search for pages containing "API"
tool.Search(ctx, &SearchParams{
    Query: "text ~ \"API\"",
    Limit: 25,
})

// Search for pages in a specific space
tool.Search(ctx, &SearchParams{
    Query: "text ~ \"API\" and space = \"DEV\"",
    Limit: 10,
})

// Search for pages with "performance" in the title
tool.Search(ctx, &SearchParams{
    Query: "title ~ \"performance\" and type=page",
    Limit: 25,
})
```

## CQL Syntax Reference

Confluence Query Language (CQL) supports various operators:

### Text Operators
- `text ~ "query"` - Search for text (case-insensitive)
- `text = "query"` - Exact text match

### Type Operators
- `type=page` - Search only pages
- `type=blogpost` - Search only blog posts
- `type=attachment` - Search only attachments

### Space Operators
- `space = "SPACEKEY"` - Search in specific space
- `space ~ "keyword"` - Search in spaces matching keyword

### Date Operators
- `lastmodified > "2024-01-01"` - Modified after date
- `lastmodified < "2024-12-31"` - Modified before date
- `created = "today"` - Created today

### Sorting
- `order by lastmodified desc` - Sort by last modified (newest first)
- `order by lastmodified asc` - Sort by last modified (oldest first)
- `order by title asc` - Sort by title alphabetically

### Grouping
- `(text ~ "API" or text ~ "documentation") and type=page` - Group conditions

See the [Confluence CQL documentation](https://confluence.atlassian.com/display/CONFLUENCE/Advanced+Searching+in+Confluence) for more details.
