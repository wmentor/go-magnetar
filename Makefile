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
	go vet ./...

tidy:
	go mod tidy

generate:
	@if [ -x "$$(command -v mockery)" ]; then echo "run mockery..." ; mockery ; else echo "mockery not found" ; fi

test:
	go clean -testcache
	go test -race ./... -cover

install: build
	mkdir -p ${HOME}/.local/bin
	mv bin/go-magnetar ${HOME}/.local/bin/
