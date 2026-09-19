# Dependencies

## Go Libraries

| Package | Purpose |
|---|---|
| `github.com/alecthomas/kong` | CLI parser |
| `github.com/sashabaranov/go-openai` | OpenAI API client (LLM + embeddings) |
| `github.com/qdrant/go-client` | Qdrant client (gRPC) |
| `github.com/google/uuid` | UUID v5 for deterministic chunk IDs |
| `github.com/knadh/koanf/v2` | YAML config loading |
| `github.com/charmbracelet/glamour` | Markdown rendering in terminal |
| `github.com/melbahja/goph/v2` | SSH client for remote command execution |
| `github.com/razvandimescu/gopdf` | PDF document generation and manipulation |
| `github.com/xuri/excelization/v2` | Excel (.xlsx) file reading and writing |
| `github.com/PuerkitoBio/goquery` | HTML parsing and DOM manipulation (used by webfetch) |
| `github.com/atotto/clipboard` | Clipboard operations (used by /copy command) |
| `github.com/stretchr/testify` | Testing framework (test mocks and assertions) |
| `github.com/vektra/mockery` | Mock generation for testing |

## External Tools

### Mockery

Mockery is used to generate mock objects for testing. It is invoked via `make generate` and requires `mockery` to be installed in your `$PATH`.

Install mockery:

```bash
go install github.com/vektra/mockery/v3@v3.7.4
```

### GoReleaser

GoReleaser is used to build and publish releases in GitHub Flow. It is configured via `.goreleaser.yaml` and used in CI/CD pipelines.

Install GoReleaser:

```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

Or download from [GitHub Releases](https://github.com/goreleaser/goreleaser/releases).

See [docs/release.md](./release.md) for release process documentation.
