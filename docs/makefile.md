# Makefile Commands

This document describes all available Makefile targets for go-magnetar.

## Available Commands

### make setup-tools

Install all development tools required for working on go-magnetar into `$(go env GOPATH)/bin`:

- **gotestsum** (`gotest.tools/gotestsum@latest`) — improved test output used by `make test`
- **golangci-lint** (`github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`) — matches CI `GOLANGCI_LINT_VERSION`, used by `make lint`
- **govulncheck** (`golang.org/x/vuln/cmd/govulncheck@latest`) — vulnerability scanner used in CI
- **mockery** (`github.com/vektra/mockery/v3@v3.7.4`) — mock generation used by `make generate`
- **goreleaser** (`github.com/goreleaser/goreleaser/v2@latest`) — optional release builder

Ensure `$(go env GOPATH)/bin` is on your `PATH` after running this target. Tools are installed with `go install` and are not vendored into the repository.

### make all

Run the full build pipeline: clean, fix, format, generate, tidy, test, lint, and build.

This is the default target and should be used for CI/CD or pre-commit validation.

### make build

Build the binary and place it in bin/go-magnetar.

### make clean

Remove the bin/ directory and all built binaries.

### make run

Build and run the application.

### make format

Run go fmt on all packages to format code according to Go standards.

### make fix

Run go fix on all packages to apply automatic code fixes.

### make lint

Run go vet to identify suspicious constructs in the code.

### make tidy

Run go mod tidy to update go.mod and go.sum files.

### make generate

Run mockery to generate mock objects for testing.

This command checks if mockery is installed and runs it if available.

### make test

Run all tests with race detection and coverage reporting.

This command:
- Cleans the test cache
- Runs all tests with the -race flag for race condition detection
- Reports code coverage statistics

## Usage Notes

- Most targets depend on others (e.g., run depends on build)
- The all target runs targets in a specific order: clean → fix → format → generate → tidy → test → lint → build
- Run make without arguments to execute make all
- Individual targets can be run independently for faster development cycles
