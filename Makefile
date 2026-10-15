.PHONY: all clean run/fix run/tidy fmt/sources fmt/imports generate test check/vet check/lint build install setup/tools

ifeq ($(shell command -v gotestsum 2> /dev/null),)
    TEST_RUNNER := go test
else
    TEST_RUNNER := gotestsum --
endif

all: clean run/fix run/tidy fmt/sources fmt/imports generate test check/vet check/lint build install

build:
	@echo "build go-magnetar"
	@go build -o bin/go-magnetar ./cmd/go-magnetar

clean:
	@echo "clean binaries"
	@rm -rf bin/

run: build
	./bin/go-magnetar

check/lint:
	@if [ -x "$$(command -v golangci-lint)" ]; then echo "run golangci-lint..." ; golangci-lint run ./... ; else echo "golangci-lint not found" ; fi

check/tidy:
	@echo "check go mod tidy"
	@go mod tidy && git diff --exit-code -- go.mod go.sum

check/vuln:
	@if [ -x "$$(command -v govulncheck)" ]; then echo "run govulncheck..." ; govulncheck -show verbose ./... ; else echo "govulncheck not found" ; fi

check/vet:
	@echo "run go vet"
	@go vet ./...

run/fix:
	@echo "run go fix"
	@go fix ./...

run/tidy:
	@echo "run go mod tidy"
	@go mod tidy

fmt/imports:
	@if [ -x "$$(command -v gci)" ]; then echo "run gci" ; gci write --skip-generated -s standard -s default -s "prefix(github.com/wmentor/go-magnetar)" . ; else echo "gci not found" ; fi

fmt/sources:
	@if [ -x "$$(command -v gofmpt)" ]; then echo "run gofmpt" ; gofumpt -l -w . ; else echo "gofmpt not found" ; fi

generate:
	@if [ -x "$$(command -v mockery)" ]; then echo "run mockery..." ; mockery ; else echo "mockery not found" ; fi

test:
	@echo "clean cache"
	@go clean -testcache
	@echo "run tests"
	@$(TEST_RUNNER) -race -cover -coverprofile=coverage.out ./...
	@CURRENT_COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); echo "Code coverage: $$CURRENT_COVERAGE%"
	@rm -f coverage.out

install: build
	mkdir -p ${HOME}/.local/bin
	mv bin/go-magnetar ${HOME}/.local/bin/

setup/tools:
	@echo "Installing development tools..."
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/vektra/mockery/v3@v3.8.1
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install github.com/evilmartians/lefthook/v2@v2.2.2
	@go install github.com/daixiang0/gci@latest
	@go install mvdan.cc/gofumpt@latest
	@echo "Done. Tools installed to $$(go env GOPATH)/bin (ensure it is on your PATH)."
