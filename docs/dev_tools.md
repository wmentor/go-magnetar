# Development Tools

Prefer installing all of these at once with:

```bash
make setup/tools
```

That target runs `go install` for gotestsum, golangci-lint v2.14.0 (matching CI), govulncheck, mockery, goreleaser, lefthook, gci, and gofumpt into `$(go env GOPATH)/bin`. Manual installs below remain available if you only need one tool.

## gotestsum

gotestsum provides clearer test output. It is used by `make test` instead of plain `go test` when present on `PATH`.

Install gotestsum:

```bash
go install gotest.tools/gotestsum@latest
```

## govulncheck

govulncheck scans for known Go vulnerabilities. Run via `make check/vuln`. CI runs it on every PR.

Install govulncheck:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Mockery

Mockery is used to generate mock objects for testing. Run via `make generate`.

Install mockery:

```bash
go install github.com/vektra/mockery/v3@v3.8.1
```

## golangci-lint

golangci-lint is used for linting Go code. Run via `make check/lint`.

Install golangci-lint v2.14.0 (matches CI `GOLANGCI_LINT_VERSION`):

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
```

Or download from [GitHub Releases](https://github.com/golangci/golangci-lint/releases).

## GoReleaser

GoReleaser is used to build and publish releases in GitHub Flow. Releases are triggered by pushing a signed tag (format `vX.Y.Z`) to the repository, which triggers the GitHub Actions release workflow that runs `goreleaser release`.

Install GoReleaser:

```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

Or download from [GitHub Releases](https://github.com/goreleaser/goreleaser/releases).

See [docs/release.md](./release.md) for release process documentation.

## gofumpt

gofumpt is a stricter Go formatter that enforces consistent code style. Run via `make fmt/sources`.

Install gofumpt:

```bash
go install mvdan.cc/gofumpt@latest
```

## gci

gci is a tool for pinning and ordering Go imports. Run via `make fmt/imports`.

Install gci:

```bash
go install github.com/daixiang0/gci@latest
```

## lefthook

lefthook is a git hooks manager used to enforce code quality before commits and pushes. It is configured via `lefthook.yaml`.

Install lefthook:

```bash
go install github.com/evilmartians/lefthook/v2@v2.2.2
```
