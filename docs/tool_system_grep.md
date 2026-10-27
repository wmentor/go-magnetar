# system_grep Tool

The `system_grep` tool executes system grep or ripgrep command to search for patterns in files and directories.

## Overview

The tool provides a safe way to search for text patterns in files using system grep or ripgrep (rg). It automatically detects which tool is available and uses it for searching.

## Features

- **Pattern Search**: Search for regular expression patterns in files or directories
- **Recursive Search**: Automatically searches recursively through directories
- **Case Insensitive**: Searches are case-insensitive by default
- **Line Numbers**: Results include line numbers for matching lines
- ** Ripgrep Support**: Automatically uses ripgrep if available (faster than grep)
- **Sandboxed Execution**: Runs within the agent's sandboxed filesystem

## Configuration

The tool is automatically enabled when `grep` or `ripgrep` is available on the system.

## Usage

The tool is invoked by the LLM when it needs to search for patterns in files. The tool parameters are:

| Parameter | Type | Description |
|-----------|------|-------------|
| `filename` | string | Path to the file or directory to search (relative to the sandbox root) |
| `pattern` | string | Regular expression pattern to search for |

## Examples

### Search for a pattern in a file

```json
{
  "filename": "docs/guide.md",
  "pattern": "system_grep"
}
```

### Search for a pattern in a directory

```json
{
  "filename": "configs",
  "pattern": "api_key"
}
```

### Case insensitive pattern matching

The tool automatically performs case-insensitive searches, so all of these patterns match:

```json
{
  "filename": "README.md",
  "pattern": "System_grep"
}
```

## Output Format

The tool returns the search results as plain text, with each matching line on a new line, prefixed by the line number:

```
12:This is a line with system_grep in it
45:Another line mentioning System_grep
```

If no matches are found, the tool returns:

```
No match found
```

## Error Handling

- **File not found**: If the specified file or directory doesn't exist, the tool returns an error
- **Invalid pattern**: If the regular expression is invalid, the tool returns an error
- **Permission denied**: If the tool cannot read the file due to permissions, an error is returned

## Implementation Details

The tool uses the following system commands:

- **grep**: `grep -n -i -r -E <pattern> <filename>`
- **ripgrep**: `rg -n -i <pattern> <filename>`

Both tools are configured with:
- `-n`: Show line numbers
- `-i`: Case-insensitive search
- `-r` (grep only): Recursive search (ripgrep is recursive by default)
- `-E` (grep only): Extended regex support
