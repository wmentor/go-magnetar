.PHONY: all build clean install run-agent lint tidy fix format

all: clean fix format tidy test lint build install

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

test:
	go clean -testcache
	go test -race ./... -cover

install: build
	mkdir -p ${HOME}/.local/bin
	mv bin/go-magnetar ${HOME}/.local/bin/

