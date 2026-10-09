.PHONY: all build clean install run-agent lint tidy fix format

all: clean fix format generate tidy test lint build install

build:
	go build -o bin/go-magnetar ./cmd/go-magnetar

clean:
	rm -rf bin/

run-agent: build
	./bin/go-magnetar agent -c configs/config.yaml

format:
	go fmt ./...

fix:
	go fix ./...

lint:
	@if [ -x "$$(command -v golangci-lint)" ]; then echo "run golangci-lint..." ; golangci-lint run ./... ; else echo "golangci-lint not found" ; fi

vet:
	go vet ./...

tidy:
	go mod tidy

generate:
	#@if [ -x "$$(command -v mockery)" ]; then echo "run mockery..." ; mockery ; else echo "mockery not found" ; fi

test:
	@echo "clean cache"
	@go clean -testcache
	@echo "run tests"
	@go test -race ./... -cover -coverprofile=coverage.out
	@CURRENT_COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); echo "Code coverage: $$CURRENT_COVERAGE%"
	@rm -f coverage.out

install: build
	mkdir -p ${HOME}/.local/bin
	mv bin/go-magnetar ${HOME}/.local/bin/
