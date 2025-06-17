.PHONY: build clean install test run-example

BINARY_NAME=presto
VERSION=1.0.0
BUILD_FLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) cmd/presto/main.go

build-all:
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64 cmd/presto/main.go
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64 cmd/presto/main.go
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-arm64 cmd/presto/main.go
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe cmd/presto/main.go

install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

clean:
	rm -f $(BINARY_NAME)*

test:
	go test ./...

deps:
	go mod download
	go mod tidy

run-example:
	./$(BINARY_NAME) --cmd add-docs --dry-run --verbose

.DEFAULT_GOAL := build
