# Text Preprocessor

The agent applies text preprocessors to user input before processing. Text preprocessors transform the input text by expanding placeholders and performing other transformations.

## Built-in placeholders

The built-in generic plugin provides the following placeholders that are automatically expanded:

| Placeholder | Description |
|---|---|
| `{{home}}` | Your home directory path |
| `{{uuid}}` | A random UUID v4 |
| `{{date}}` | Current date in `YYYY-MM-DD` format |
| `{{now}}` | Current date and time in `YYYY-MM-DD HH:MM:SS` format |
| `{{file:filename}}` | Reads file content and replaces placeholder with the file contents. Supports `.md`, `.txt`, `.docx`, `.pdf`, `.odt`, and `.pptx` files, absolute paths and `~/` home directory prefix |
| `{{env:VARIABLE}}` | Value of environment variable `VARIABLE` |

## File content substitution

The `{{file:filename}}` placeholder supports reading file contents. Supported formats:

| Format | Description |
|---|---|
| `.txt` | Plain text files (via `os.ReadFile`) |
| `.md` | Markdown files (via `os.ReadFile`) |
| `.docx` | Microsoft Word documents (via `internal/docx.ReadFile`) |
| `.pdf` | PDF documents (via `internal/pdf.ReadFile`) |
| `.odt` | OpenOffice Writer documents (via `internal/odt.ReadFile`) |
| `.pptx` | PowerPoint presentations (via `internal/pptx.ReadFile`) |

File paths can be absolute or use `~/` for the home directory prefix.

## Environment variable substitution

The `{{env:VARIABLE}}` placeholder substitutes the value of an environment variable. If the variable is not set, an empty string is returned.

Example: `{{env:OPENAI_API_KEY}}` will be replaced with the value of the `OPENAI_API_KEY` environment variable.

## Usage in non-interactive mode

In `-f/--file` mode, text preprocessors are applied but chat commands (e.g., `/readonly`, `/fetch`, `/index`) are not available. The text preprocessor expands all placeholders listed above.

## Adding a custom preprocessor

Plugins can register custom preprocessors using `Hub.RegisterPreprocessor()`. The preprocessor function signature is:

```go
func(ctx context.Context, text string) (string, error)
```

Preprocessors are applied in registration order.
