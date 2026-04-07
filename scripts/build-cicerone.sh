#!/bin/bash
# Build Cicerone from source
# Usage: ./build-cicerone.sh

set -e

CICERONE_DIR="/opt/cicerone"
BUILD_DIR="/opt/cicerone/build"

echo "=== Building Cicerone ==="
echo ""

# Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo "Installing Go..."
    apt-get update
    apt-get install -y golang-go
fi

echo "Go version: $(go version)"

# Clone if not exists
if [ ! -d "$CICERONE_DIR" ]; then
    echo "Cloning cicerone..."
    git clone https://idm.wezzel.com/crab-meat-repos/cicerone.git "$CICERONE_DIR"
fi

cd "$CICERONE_DIR"

# Pull latest
echo "Updating repository..."
git pull origin main || git pull origin master

# Download dependencies
echo "Downloading dependencies..."
go mod tidy
go mod download

# Build with optimizations
echo "Building..."
go build -ldflags="-s -w" -o cicerone .

# Verify build
echo ""
echo "Build output:"
ls -la cicerone
du -h cicerone

# Test binary
echo ""
echo "Testing binary..."
./cicerone about
./cicerone --help
./cicerone llm show

# Create build directory
mkdir -p "$BUILD_DIR"
cp cicerone "$BUILD_DIR/"

# Install to system
echo "Installing to /usr/local/bin..."
cp cicerone /usr/local/bin/ 2>/dev/null || sudo cp cicerone /usr/local/bin/

echo ""
echo "=== Cicerone Build Complete ==="
echo "Binary: $BUILD_DIR/cicerone"
echo "Installed: /usr/local/bin/cicerone"

exit 0