#!/usr/bin/env bash

set -euo pipefail

# === CONFIGURATION ===
REPO="github.com/Zachacious/presto" # Your GitHub repo (user/repo)
MAIN_BRANCH="main"                  # Or "master"
INITIAL_VERSION="v0.1.0"            # The version for the very first release

# === SCRIPT LOGIC ===

# --- Initial Health Checks ---
if ! command -v gh >/dev/null 2>&1; then
    echo "❌ GitHub CLI (gh) is required but not found. Please install it: https://cli.github.com/"
    exit 1
fi

if ! git diff-index --quiet HEAD --; then
    echo "❌ Uncommitted changes detected. Please commit or stash them before running a release."
    git status --short
    exit 1
fi

# --- Git Synchronization ---
echo "🔄 Switching to '$MAIN_BRANCH' and pulling latest changes..."
git checkout "$MAIN_BRANCH"

if ! GIT_TERMINAL_PROMPT=0 git pull origin "$MAIN_BRANCH"; then
    echo "❌ Git Error: Failed to pull from origin. Please ensure your git credentials are configured correctly."
    exit 1
fi

echo "🔄 Fetching all tags from the 'origin' remote..."
if ! GIT_TERMINAL_PROMPT=0 git fetch origin --tags --force; then
    echo "❌ Git Error: Failed to fetch tags. Please ensure your git credentials are configured correctly."
    exit 1
fi

# --- Argument Parsing ---
BUMP="patch"
VERSION=""
NOTES=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    v*.*.*) VERSION="$1"; shift ;;
    --major) BUMP="major"; shift ;;
    --minor) BUMP="minor"; shift ;;
    --patch) BUMP="patch"; shift ;;
    -m|--message) NOTES="$2"; shift 2 ;;
    *) echo "❌ Unknown argument: $1"; exit 1 ;;
  esac
done

# --- Detect and Calculate Version ---
# This logic block is the corrected part.
if [[ -z "$VERSION" ]]; then
    # We check if the 'git describe' command SUCCEEDS. If it fails, it means no tags exist.
    if LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null); then
        # The command succeeded, so tags exist.
        echo "🔍 Latest tag found: $LATEST_TAG"
        if [[ $LATEST_TAG =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
            MAJOR="${BASH_REMATCH[1]}"
            MINOR="${BASH_REMATCH[2]}"
            PATCH="${BASH_REMATCH[3]}"
        else
            echo "❌ Invalid latest tag format: '$LATEST_TAG'. Expected vX.Y.Z"
            exit 1
        fi
        case "$BUMP" in
            major) ((MAJOR++)); MINOR=0; PATCH=0 ;;
            minor) ((MINOR++)); PATCH=0 ;;
            patch) ((PATCH++)) ;;
        esac
        VERSION="v$MAJOR.$MINOR.$PATCH"
    else
        # The command failed, which means no tags were found.
        echo "🔍 No existing tags found. Creating initial release."
        VERSION="$INITIAL_VERSION"
    fi
fi

# --- Confirmation Step ---
echo "✅ New version will be: $VERSION"
read -p "   Are you sure you want to proceed with tagging? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "🛑 Release cancelled."
    exit 1
fi

# Get release notes from the user
if [[ -z "$NOTES" ]]; then
    echo "✏️ Please enter the release notes. End with Ctrl+D."
    NOTES=$(</dev/stdin)
fi
if [[ -z "$NOTES" ]]; then
    echo "❌ Release notes cannot be empty."
    exit 1
fi

# --- Execution Step ---
echo "1. Tagging version $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

echo "2. Pushing tag to GitHub..."
if ! GIT_TERMINAL_PROMPT=0 git push origin "$VERSION"; then
    echo "❌ Git Error: Failed to push the new tag. Please check your permissions and credentials."
    git tag -d "$VERSION"
    exit 1
fi

echo "3. Building release artifacts using 'make'..."
make release

echo "4. Creating GitHub Release..."
gh release create "$VERSION" dist/* \
    --title "$VERSION" \
    --notes "$NOTES"

echo "5. Notifying Go proxy..."
(
  go list -m "$REPO@$VERSION" &>/dev/null
  curl -sSf "https://proxy.golang.org/$REPO/@v/$VERSION.info" > /dev/null
) &

echo ""
echo "✅ Release $VERSION completed successfully!"