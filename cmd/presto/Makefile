# Makefile for Presto CLI

# Metadata
BINARY      := presto
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      := $(shell git rev-parse --short HEAD)
DATE        := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Platforms to build for (OS-ARCH)
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

# Output directory
DIST := dist

# ldflags
LDFLAGS := -s -w -buildid= -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

.PHONY: all build clean release version

# Default target builds for current OS
build: $(DIST)/$(BINARY)

# Local build to dist/presto
$(DIST)/$(BINARY):
	@mkdir -p $(DIST)
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY) cmd/presto/main.go

# Cross-platform builds
all: $(PLATFORMS:%=$(DIST)/$(BINARY)-%)

$(DIST)/$(BINARY)-%:
	@platform="$*"; \
	os=$${platform%-*}; arch=$${platform#*-}; \
	outfile="$(DIST)/$(BINARY)-$$os-$$arch"; \
	[ "$$os" = "windows" ] && outfile="$$outfile.exe"; \
	mkdir -p $(DIST); \
	GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	go build -trimpath -ldflags="$(LDFLAGS)" -o "$$outfile" cmd/presto/main.go

# Clean dist directory
clean:
	rm -rf $(DIST)

# Release: build all, zip, and checksum
release: clean all
	@for platform in $(PLATFORMS); do \
		os=$${platform%-*}; arch=$${platform#*-}; \
		base="$(DIST)/$(BINARY)-$$os-$$arch"; \
		out="$$base"; [ "$$os" = "windows" ] && out="$$base.exe"; \
		zipfile="$$base.zip"; \
		zip -j "$$zipfile" "$$out"; \
	done
	@cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum * > SHA256SUMS || shasum -a 256 * > SHA256SUMS)

# Show version info
version:
	@echo "Version:   $(VERSION)"
	@echo "Commit:    $(COMMIT)"
	@echo "BuildDate: $(DATE)"
