# Makefile for presto

# Project metadata
BINARY      := presto
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      := $(shell git rev-parse --short HEAD)
DATE        := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Target platforms
PLATFORMS   := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

# Output directory
DIST        := dist

# ldflags for version injection
LDFLAGS := -s -w -buildid= -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

.PHONY: all build clean install test deps release version run-example

# Default goal
.DEFAULT_GOAL := build

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) cmd/presto/main.go

all: $(PLATFORMS:%=$(DIST)/$(BINARY)-%)

# Cross-compile
$(DIST)/$(BINARY)-%:
	@platform="$*"; \
	os=$${platform%-*}; arch=$${platform#*-}; \
	outfile="$(DIST)/$(BINARY)-$$os-$$arch"; \
	[ "$$os" = "windows" ] && outfile="$$outfile.exe"; \
	mkdir -p $(DIST); \
	GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	go build -trimpath -ldflags="$(LDFLAGS)" -o "$$outfile" cmd/presto/main.go

# Release: build + zip + checksum
release: clean all
	@for platform in $(PLATFORMS); do \
		os=$${platform%-*}; arch=$${platform#*-}; \
		base="$(DIST)/$(BINARY)-$$os-$$arch"; \
		[ "$$os" = "windows" ] && bin="$$base.exe" || bin="$$base"; \
		zipfile="$$base.zip"; \
		zip -j "$$zipfile" "$$bin"; \
	done
	@cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum * > SHA256SUMS || shasum -a 256 * > SHA256SUMS)

install: build
	sudo mv $(BINARY) /usr/local/bin/

clean:
	rm -rf $(DIST) $(BINARY)*

test:
	go test ./...

deps:
	go mod download
	go mod tidy

version:
	@echo "Version:   $(VERSION)"
	@echo "Commit:    $(COMMIT)"
	@echo "BuildDate: $(DATE)"

run-example:
	./$(BINARY) --cmd add-docs --dry-run --verbose
