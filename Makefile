.PHONY: build clean run-agent lint tidy

build:
	go build -o bin/go-magnetar ./cmd/go-magnetar

clean:
	rm -rf bin/

run-agent: build
	./bin/go-magnetar agent -c configs/config.yaml

lint:
	go vet ./...

tidy:
	go mod tidy

test:
	go clean -testcache
	go test -race ./... -cover
