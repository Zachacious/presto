# Makefile for Presto CLI

# Metadata
BINARY      := presto
# The version is dynamically determined from the latest git tag.
# --always ensures a version is generated even with no tags.
# --dirty appends '-dirty' if you have uncommitted changes.
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      := $(shell git rev-parse --short HEAD)
DATE        := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Platforms to build for (OS-ARCH)
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

# Output directory
DIST := dist

# ldflags to inject version info into the binary
LDFLAGS := -s -w -buildid= -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

.PHONY: all build clean release version

# Default target builds for the current OS/architecture
build: $(DIST)/$(BINARY)

# Local build target
$(DIST)/$(BINARY):
	@mkdir -p $(DIST)
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY) cmd/presto/main.go

# Cross-platform builds for all defined platforms
all: $(PLATFORMS:%=$(DIST)/$(BINARY)-%)

$(DIST)/$(BINARY)-%:
	@platform="$*"; \
	os=$${platform%-*}; arch=$${platform#*-}; \
	outfile="$(DIST)/$(BINARY)-$$os-$$arch"; \
	[ "$$os" = "windows" ] && outfile="$$outfile.exe"; \
	mkdir -p $(DIST); \
	echo "--> Building for $$os/$$arch..."; \
	GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	go build -trimpath -ldflags="$(LDFLAGS)" -o "$$outfile" cmd/presto/main.go

# Clean the dist directory
clean:
	rm -rf $(DIST)

# The release target builds all platforms, zips the artifacts, and creates a checksum file.
release: clean all
	@echo "--> Zipping release artifacts..."; \
	for platform in $(PLATFORMS); do \
		os=$${platform%-*}; arch=$${platform#*-}; \
		base="$(DIST)/$(BINARY)-$$os-$$arch"; \
		out="$$base"; [ "$$os" = "windows" ] && out="$$base.exe"; \
		zipfile="$$base.zip"; \
		zip -j "$$zipfile" "$$out"; \
	done
	@echo "--> Generating checksums..."; \
	cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum *.zip > SHA256SUMS || shasum -a 256 *.zip > SHA256SUMS)

# Show version info
version:
	@echo "Version:   $(VERSION)"
	@echo "Commit:    $(COMMIT)"
	@echo "BuildDate: $(DATE)"