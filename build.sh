#!/usr/bin/env bash

set -e

# Define paths
MAKEFILE_PATH="cmd/presto/Makefile"
DIST_DIR="dist"

# Help text
function usage() {
    echo "Usage: ./build.sh [command]"
    echo ""
    echo "Commands:"
    echo "  build        Build the binary for the current platform"
    echo "  release      Cross-compile for all platforms and zip outputs"
    echo "  clean        Remove build artifacts"
    echo "  version      Show version metadata"
    echo "  help         Show this help message"
    echo ""
}

# Ensure makefile exists
if [ ! -f "$MAKEFILE_PATH" ]; then
    echo "❌ Makefile not found at $MAKEFILE_PATH"
    exit 1
fi

# Run commands
case "$1" in
    build)
        echo "📦 Building presto..."
        make -f "$MAKEFILE_PATH" build
        ;;
    release)
        echo "🚀 Building release artifacts for all platforms..."
        make -f "$MAKEFILE_PATH" release
        ;;
    clean)
        echo "🧹 Cleaning..."
        make -f "$MAKEFILE_PATH" clean
        ;;
    version)
        echo "🔖 Version Info:"
        make -f "$MAKEFILE_PATH" version
        ;;
    help|"")
        usage
        ;;
    *)
        echo "❌ Unknown command: $1"
        usage
        exit 1
        ;;
esac
