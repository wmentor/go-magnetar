.PHONY: all build clean install run-agent check/lint tidy fix format setup-tools

ifeq ($(shell command -v gotestsum 2> /dev/null),)
    TEST_RUNNER := go test
else
    TEST_RUNNER := gotestsum --
endif

all: clean fix format generate tidy test check/lint build install

build:
	go build -o bin/go-magnetar ./cmd/go-magnetar

clean:
	rm -rf bin/

run: build
	./bin/go-magnetar

format:
	go fmt ./...

fix:
	go fix ./...

check/lint:
	@if [ -x "$$(command -v golangci-lint)" ]; then echo "run golangci-lint..." ; golangci-lint run ./... ; else echo "golangci-lint not found" ; fi

check/tidy:
	go mod tidy && git diff --exit-code -- go.mod go.sum

check/vuln:
	@if [ -x "$$(command -v govulncheck)" ]; then echo "run govulncheck..." ; govulncheck -show verbose ./... ; else echo "govulncheck not found" ; fi

vet:
	go vet ./...

tidy:
	go mod tidy

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

setup-tools:
	@echo "Installing development tools..."
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/vektra/mockery/v3@v3.8.1
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install github.com/evilmartians/lefthook/v2@v2.2.2
	@echo "Done. Tools installed to $$(go env GOPATH)/bin (ensure it is on your PATH)."
