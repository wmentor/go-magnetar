# File Codecs

go-magnetar supports reading multiple file formats through a unified codec interface. The codec system replaces the older `internal/common/filetypes.go` approach with a more flexible and extensible design.

## Supported Formats

| Extension | Codec | Description |
|---|---|---|
| `.csv` | `csv.Codec` | CSV files (comma-separated values) |
| `.tsv` | `tsv.Codec` | TSV files (tab-separated values) |
| `.docx` | `docx.Codec` | Microsoft Word documents |
| `.pdf` | `pdf.Codec` | PDF documents |
| `.odt` | `odt.Codec` | OpenOffice Writer documents |
| `.pptx` | `pptx.Codec` | PowerPoint presentations |
| `.xlsx` | `excel.Codec` | Excel files |

## Codec Interface

All codecs implement the `Codec` interface:

```go
type Codec interface {
    ReadFile(filename string) (string, error)
}
```

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
./bin/go-magnetar -c my-config.yaml agent
> /index path/to/document.csv
> /index path/to/document.tsv
> /index path/to/document.docx
> /index path/to/document.pdf
> /index path/to/document.odt
> /index path/to/document.pptx
> /index path/to/document.xlsx
```

## File Content Substitution

The `{{file:filename}}` placeholder supports all codec formats:

```yaml
llm:
  api_key: $file:$env:API_KEY_FILE
```

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
