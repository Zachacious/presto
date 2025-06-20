#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Configuration
MAKEFILE="cmd/presto/Makefile"
DIST_DIR="dist"
REPO="github.com/Zachacious/presto"

# Defaults
BUMP="patch"
NOTES=""

# Parse arguments and flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    --major) BUMP="major"; shift ;;
    --minor) BUMP="minor"; shift ;;
    --patch) BUMP="patch"; shift ;;
    -m|--message)
      NOTES="$2"
      shift 2
      ;;
    *)
      echo "❌ Unknown option: $1"
      echo "Usage: ./release.sh [--major|--minor|--patch] -m \"Changelog message\""
      exit 1
      ;;
  esac
done

# Ensure GitHub CLI is available
if ! command -v gh >/dev/null 2>&1; then
  echo "❌ GitHub CLI (gh) is required. Install it: https://cli.github.com/"
  exit 2
fi

# Ensure working directory is clean
if ! git diff-index --quiet HEAD --; then
  echo "❌ Uncommitted changes found. Commit or stash before releasing."
  exit 1
fi

# Get latest version tag (default to v0.0.0 if none)
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION_REGEX="^v([0-9]+)\.([0-9]+)\.([0-9]+)$"

if [[ $LATEST_TAG =~ $VERSION_REGEX ]]; then
  MAJOR="${BASH_REMATCH[1]}"
  MINOR="${BASH_REMATCH[2]}"
  PATCH="${BASH_REMATCH[3]}"
else
  echo "❌ Failed to parse latest version tag: $LATEST_TAG"
  exit 1
fi

# Calculate next version
case "$BUMP" in
  major)
    ((MAJOR++)); MINOR=0; PATCH=0 ;;
  minor)
    ((MINOR++)); PATCH=0 ;;
  patch)
    ((PATCH++)) ;;
esac

NEW_VERSION="v$MAJOR.$MINOR.$PATCH"

echo "🔖 Latest tag: $LATEST_TAG"
echo "📈 Bumping $BUMP → $NEW_VERSION"

if [[ -z "$NOTES" ]]; then
  echo "⚠️  No changelog message provided. Use: -m \"your message\""
  read -p "Enter changelog notes: " NOTES
fi

# Tag and push
echo "🏷️  Tagging release ${NEW_VERSION}..."
git tag "${NEW_VERSION}"
git push origin main
git push origin "${NEW_VERSION}"

# Build release
echo "🔨 Building release artifacts..."
make -f "$MAKEFILE" release

# Upload to GitHub
echo "🚀 Creating GitHub release and uploading artifacts..."
gh release create "${NEW_VERSION}" "${DIST_DIR}"/* \
  --title "${NEW_VERSION}" \
  --notes "${NOTES}"

# Notify Go Proxy
echo "📣 Notifying pkg.go.dev..."
go list -m "$REPO@$NEW_VERSION" || true
curl -sSf "https://proxy.golang.org/$REPO/@v/$NEW_VERSION.info" > /dev/null || true

echo "✅ Release ${NEW_VERSION} published!"
echo "🌐 Visit: https://pkg.go.dev/$REPO@$NEW_VERSION"
