#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Configuration
MAKEFILE_PATH="cmd/presto/Makefile"
DIST_DIR="dist"
REPO="github.com/Zachacious/presto"

# Usage message
usage() {
  echo "Usage: $0 v1.0.2 \"Release notes or changelog\""
  exit 1
}

# Check inputs
if [[ $# -lt 2 ]]; then usage; fi

VERSION="$1"
NOTES="$2"

if [[ -z "$VERSION" ]]; then usage; fi

# Ensure GitHub CLI is available
if ! command -v gh >/dev/null 2>&1; then
  echo "❌ GitHub CLI (gh) is required. Install it: https://cli.github.com/"
  exit 2
fi

# Clean and validate Go modules
echo "📦 Tidying modules..."
go mod tidy

# Ensure clean working directory
if ! git diff-index --quiet HEAD --; then
  echo "❌ Uncommitted changes found. Commit or stash before releasing."
  exit 1
fi

# Tag and push
echo "🏷️  Tagging release ${VERSION}..."
git tag "${VERSION}"
git push origin main
git push origin "${VERSION}"

# Build release artifacts
echo "🔨 Building release artifacts..."
make -f "$MAKEFILE_PATH" release

# Upload to GitHub
echo "🚀 Creating GitHub release and uploading artifacts..."
gh release create "${VERSION}" "${DIST_DIR}"/* \
  --title "${VERSION}" \
  --notes "${NOTES}"

echo "✅ Release ${VERSION} published!"

# Notify pkg.go.dev
echo "📣 Requesting pkg.go.dev to fetch new version..."
go list -m "${REPO}@${VERSION}" || true
curl -sSf "https://proxy.golang.org/${REPO}/@v/${VERSION}.info" > /dev/null || true

echo "🌐 Visit https://pkg.go.dev/${REPO}@${VERSION} to verify indexing."
