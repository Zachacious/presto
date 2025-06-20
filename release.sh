#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# Paths
MAKEFILE="cmd/presto/Makefile"
DIST_DIR="dist"
REPO="github.com/Zachacious/presto"

# Inputs
VERSION=""
BUMP=""
NOTES=""

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    v*.*.*) VERSION="$1"; shift ;;
    --major) BUMP="major"; shift ;;
    --minor) BUMP="minor"; shift ;;
    --patch) BUMP="patch"; shift ;;
    -m|--message) NOTES="$2"; shift 2 ;;
    *)
      echo "❌ Unknown argument: $1"
      echo "Usage:"
      echo "  ./release.sh v1.2.4 -m \"Release notes...\""
      echo "  ./release.sh --minor -m \"Release notes...\""
      exit 1
      ;;
  esac
done

# Require gh CLI
if ! command -v gh >/dev/null 2>&1; then
  echo "❌ GitHub CLI (gh) required: https://cli.github.com/"
  exit 1
fi

# Ensure clean working directory
if ! git diff-index --quiet HEAD --; then
  echo "❌ Uncommitted changes! Please commit or stash first."
  exit 1
fi

# Determine version if not explicitly provided
if [[ -z "$VERSION" ]]; then
  LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
  echo "🔍 Latest tag: $LATEST"

  if [[ $LATEST =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
  else
    echo "❌ Invalid latest tag: $LATEST"
    exit 1
  fi

  case "$BUMP" in
    major)   ((MAJOR++)); MINOR=0; PATCH=0 ;;
    minor)   ((MINOR++)); PATCH=0 ;;
    patch|"") ((PATCH++)) ;;
    *)
      echo "❌ Unknown bump type: $BUMP"
      exit 1
      ;;
  esac

  VERSION="v$MAJOR.$MINOR.$PATCH"
fi

# Validate version format
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "❌ Invalid version format: $VERSION"
  exit 1
fi

# Get release notes
if [[ -z "$NOTES" ]]; then
  echo "✏️  Enter release notes (end with Ctrl+D):"
  NOTES=$(</dev/stdin)
fi

if [[ -z "$NOTES" ]]; then
  echo "❌ Release notes are required."
  exit 1
fi

# Tag + push
echo "🏷️  Tagging $VERSION..."
git tag "$VERSION"
git push origin main
git push origin "$VERSION"

# Build
echo "🔨 Building release artifacts..."
make -f "$MAKEFILE" release

# Create GitHub release
echo "🚀 Creating GitHub release..."
gh release create "$VERSION" "$DIST_DIR"/* \
  --title "$VERSION" \
  --notes "$NOTES"

# Trigger Go proxy indexing
echo "📣 Notifying Go proxy..."
go list -m "$REPO@$VERSION" || true
curl -sSf "https://proxy.golang.org/$REPO/@v/$VERSION.info" > /dev/null || true

echo "✅ Release $VERSION completed!"
echo "🌍 https://pkg.go.dev/$REPO@$VERSION"
