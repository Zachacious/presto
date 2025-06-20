#!/usr/bin/env bash

set -e

# Help text
function usage() {
    echo "Usage: ./build.sh [command]"
    echo ""
    echo "Commands:"
    echo "  build      Build the binary for the current platform"
    echo "  release    Cross-compile for all platforms and zip outputs"
    echo "  clean      Remove build artifacts"
    echo "  version    Show version metadata"
    echo "  help       Show this help message"
    echo ""
}

# Run commands from the root Makefile
case "$1" in
    build)
        echo "📦 Building presto..."
        make build
        ;;
    release)
        echo "🚀 Building release artifacts for all platforms..."
        make release
        ;;
    clean)
        echo "🧹 Cleaning..."
        make clean
        ;;
    version)
        echo "🔖 Version Info:"
        make version
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