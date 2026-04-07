#!/bin/bash
# scripts/build.sh - Build cicerone with version info
# Usage: ./scripts/build.sh [version]

set -e

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building cicerone..."
echo "  Version: ${VERSION}"
echo "  Commit:  ${COMMIT}"
echo "  Date:    ${DATE}"
echo ""

# Build with ldflags
go build -ldflags "\
    -X github.com/crab-meat-repos/cicerone/cmd.Version=${VERSION} \
    -X github.com/crab-meat-repos/cicerone/cmd.Commit=${COMMIT} \
    -X github.com/crab-meat-repos/cicerone/cmd.Date=${DATE}" \
    -o cicerone .

# Check binary
SIZE=$(ls -lh cicerone | awk '{print $5}')
echo ""
echo "Build complete!"
echo "  Binary: cicerone"
echo "  Size:   ${SIZE}"
echo ""

# Run tests
echo "Running tests..."
go test ./... -v 2>&1 | tail -20

echo ""
echo "Build successful!"