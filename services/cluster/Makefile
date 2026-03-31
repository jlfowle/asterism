.PHONY: run build test lint fmt clean

BINARY=dist/server

run:
	go run ./cmd/server

build:
	mkdir -p dist
	go build -o $(BINARY) ./cmd/server

test:
	go test ./... -coverprofile=coverage.out -covermode=atomic

lint:
	gofmt -w $(shell find . -name '*.go' -type f)
	go vet ./...

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

clean:
	rm -rf dist coverage.out
