# File Codecs

go-magnetar supports reading multiple file formats through a unified codec interface. The codec system replaces the older `internal/common/filetypes.go` approach with a more flexible and extensible design.

## Supported Formats

| Extension | Codec | Description | Read | Write |
|---|---|---|---|---|
| `.csv` | `csv.Codec` | CSV files (comma-separated values) | ✅ | ✅ |
| `.tsv` | `tsv.Codec` | TSV files (tab-separated values) | ✅ | ✅ |
| `.docx` | `docx.Codec` | Microsoft Word documents | ✅ | ✅ |
| `.html` | `html.Codec` | HTML files (cleaned and converted to Markdown) | ✅ | ✅ |
| `.pdf` | `pdf.Codec` | PDF documents | ✅ | ❌ |
| `.odt` | `odt.Codec` | OpenOffice Writer documents | ✅ | ❌ |
| `.pptx` | `pptx.Codec` | PowerPoint presentations | ✅ | ❌ |
| `.xlsx` | `excel.Codec` | Excel files | ✅ | ❌ |
| `.eml` | `eml.Codec` | EML files | ✅ | ❌ |
| `.pages` | `pages.Codec` | Apple Pages documents (iWork '13 and newer) | ✅ | ❌ |

Additionally, all text-based formats are supported with read and write capabilities: `.txt`, `.md`, `.pl`, `.py`, `.go`, `.java`, `.c`, `.cpp`, `.h`, `.hpp`, `.js`, `.ts`, `.rs`, `.rb`, `.php`, `.swift`, `.kt`, `.scala`, `.r`, `.m`, `.cs`, `.vb`, `.sh`, `.bash`, `.zsh`, `.fish`, `.yaml`, `.yml`, `.json`, `.xml`, `.toml`, `.ini`, `.cfg`, `.conf`, `.properties`, `.sql`, `.graphql`, `.proto`, `.tf`, `.hcl`, `.rst`, `.asciidoc`, `.org`, `.lua`, `.tex`, `.ltx`, `.bib`, `.log`, `.env`, `.gitignore`, `.dockerignore`, `.gitattributes`, `.editorconfig`, `.license`, `.notice`, `.readme`, `.changelog`, `.contributing`, `.code-of-conduct`, `.security`, `.authors`, `.maintainers`, `.contributors`.

## Codec Interface

All codecs implement the `Codec` interface:

```go
type Codec interface {
    ReadFile(filename string) (string, error)
}
```

**Note:** For binary formats like `.pages`, the codec uses the [github.com/wmentor/iwaread](https://github.com/wmentor/iwaread) library to extract text content from Apple Pages documents (iWork '13 and newer).

## Using Codecs

Use `codec.GetCodec()` to retrieve a codec for a given file extension:

```go
import "github.com/wmentor/go-magnetar/internal/codec"

reader, err := codec.GetCodec(".csv")
if err != nil {
    // handle error
}
text, err := reader.ReadFile("data.csv")
```

File paths can be absolute or use `~/` for the home directory prefix.

## Indexer Support

The indexer automatically uses codecs when reading files for indexing. Supported formats can be indexed via:

```bash
./bin/go-magnetar
> /index path/to/document.csv
> /index path/to/document.tsv
> /index path/to/document.docx
> /index path/to/document.pdf
> /index path/to/document.odt
> /index path/to/document.pptx
> /index path/to/document.xlsx
> /index path/to/document.html
> /index path/to/document.pages
```

## File Content Substitution

The `{{file:filename}}` placeholder supports all codec formats, including `.pages` documents:

```yaml
llm:
  api_key: $file:$env:API_KEY_FILE
```

When reading a `.pages` file, the content is extracted using the iwaread library and converted to plain text.

## Adding a New Codec

To add support for a new file format:

1. Create a new directory under `internal/codec/` with the format name (e.g., `internal/codec/newformat/`)
2. Implement the `Codec` interface in `internal/codec/newformat/newformat.go`:
   ```go
   package newformat

   type Codec struct{}

   func (c *Codec) ReadFile(filename string) (string, error) {
       // Read and convert file to text
       return text, nil
   }
   ```
   For binary formats like `.pages`, use appropriate libraries (e.g., `github.com/wmentor/iwaread` for Apple Pages documents).
3. Register the codec in `internal/codec/codec.go`:
   ```go
   import "github.com/wmentor/go-magnetar/internal/codec/newformat"

   var (
       codecs = map[string]Codec{
           ".newformat": &newformat.Codec{},
           "newformat":  &newformat.Codec{},
       }
   )
   ```
