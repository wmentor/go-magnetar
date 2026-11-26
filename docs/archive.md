# Archive Tools

Tools for working with archive files (ZIP, etc.).

## Tools

### archive_list

Lists all files within an archive with their metadata. Supports filtering by file name pattern.

**Parameters:**
- `filename` (string, required): Path to the archive file (absolute or relative)
- `filter` (string, optional): Glob pattern to filter files (e.g., `*.txt`, `docs/*.md`, `src/*.go`)

**Returns:**
JSON array of file information with the following fields:
- `name`: File path within the archive
- `size`: File size in bytes
- `is_dir`: Whether the entry is a directory
- `mod_time`: Last modification time

**Examples:**

List all files:
```json
{
  "filename": "/path/to/archive.zip"
}
```

List only text files:
```json
{
  "filename": "/path/to/archive.zip",
  "filter": "*.txt"
}
```

List only markdown files:
```json
{
  "filename": "/path/to/archive.zip",
  "filter": "*.md"
}
```

**Response:**
```json
[
  {
    "name": "file1.txt",
    "size": 1234,
    "is_dir": false,
    "mod_time": "2024-01-15T10:30:00Z"
  },
  {
    "name": "dir/file2.txt",
    "size": 5678,
    "is_dir": false,
    "mod_time": "2024-01-15T10:31:00Z"
  }
]
```

**Filter Pattern Syntax:**

The filter parameter supports glob patterns:
- `*` matches any sequence of characters
- `?` matches any single character
- `[abc]` matches any character in the set
- `[!abc]` or `[^abc]` matches any character not in the set

Examples:
- `*.txt` - all text files
- `*.md` - all markdown files
- `docs/*` - all files in the docs directory
- `src/*.go` - all Go files in the src directory
- `*.{go,md}` - not supported (use separate filters)

**Note:** The filter matches both the full path and the base name of files. For example, `*.md` will match both `readme.md` and `docs/guide.md`.

### archive_read

Reads the contents of a specific file within an archive.

**Parameters:**
- `archive_path` (string, required): Path to the archive file (absolute or relative)
- `file_path` (string, required): Path to the file within the archive

**Returns:**
The contents of the file as a string.

**Example:**
```json
{
  "archive_path": "/path/to/archive.zip",
  "file_path": "config.json"
}
```

**Response:**
```
{
  "setting": "value",
  "enabled": true
}
```

## Supported Formats

Currently supported archive formats:
- **ZIP** (`.zip`, MIME: `application/zip`)

## Features

- **Case-insensitive file matching**: File paths are matched case-insensitively as a fallback
- **Path normalization**: File paths are normalized using `path.Clean()`
- **Metadata extraction**: Full file metadata including size, modification time, and directory status
- **File filtering**: Filter files by glob patterns to handle large archives efficiently

## Error Handling

The tools return appropriate error messages for:
- Unsupported archive formats
- Files not found within the archive
- Corrupted or invalid archive files

## Architecture

The archive system follows the same pattern as the codec system:

1. **`internal/archive`**: Core interfaces and registry
   - `Archive` interface with `List()` and `ReadFile()` methods
   - `FileInfo` struct for file metadata
   - Registry for archive handlers

2. **`internal/archive/zip`**: ZIP implementation
   - Implements the `Archive` interface
   - Registers itself via `init()` function

3. **`internal/tools/archive`**: Tool implementations
   - `list`: Lists archive contents with optional filtering
   - `read`: Reads files from archives

4. **`internal/plugins/archive`**: Plugin registration
   - Registers tools with the agent
   - Imports archive implementations to trigger registration

## Adding New Archive Formats

To add support for a new archive format (e.g., TAR, RAR):

1. Create a new package in `internal/archive/<format>/`
2. Implement the `Archive` interface
3. Register the format in the `init()` function:
   ```go
   func init() {
       archive.Register(&Archive{}, []string{".tar"}, []string{"application/x-tar"})
   }
   ```
4. Import the package in `internal/plugins/archive/plugin.go`

The tools will automatically support the new format.

## Performance Considerations

For large archives with thousands of files:
- Use the `filter` parameter in `archive_list` to reduce the number of files returned
- Filter patterns are applied after listing all files, so the archive is still fully scanned
- Consider using specific patterns like `*.go` or `src/*.md` to narrow down results
