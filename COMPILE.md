# Compiling Cicerone

This document describes how to compile cicerone for various platforms and architectures.

## Prerequisites

- Go 1.22 or later
- Git (for version embedding)

## Quick Start

```bash
# Build for current platform
go build -o cicerone .

# Or use make
make build
```

## Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build for current platform |
| `make test` | Run all tests |
| `make coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make clean` | Clean build artifacts |
| `make install` | Install to /usr/local/bin |
| `make build-all` | Build for all platforms |
| `make release` | Create release tarballs |

## Platform-Specific Builds

### Linux AMD64

```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o cicerone-linux-amd64 .
```

### Linux ARM64 (aarch64)

```bash
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o cicerone-linux-arm64 .
```

### macOS AMD64 (Intel Macs)

```bash
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o cicerone-darwin-amd64 .
```

### macOS ARM64 (Apple Silicon)

```bash
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o cicerone-darwin-arm64 .
```

### Windows AMD64

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o cicerone-windows-amd64.exe .
```

### Windows ARM64

```bash
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o cicerone-windows-arm64.exe .
```

## Build Optimizations

### Reduced Binary Size

```bash
# Strip debug symbols and DWARF
go build -ldflags="-s -w" -o cicerone .

# Even smaller with UPX (optional, requires UPX installed)
go build -ldflags="-s -w" -o cicerone .
upx --best cicerone
```

### With Version Information

```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags="-X github.com/crab-meat-repos/cicerone-goclaw/cmd.Version=$VERSION \
                   -X github.com/crab-meat-repos/cicerone-goclaw/cmd.Commit=$COMMIT \
                   -X github.com/crab-meat-repos/cicerone-goclaw/cmd.Date=$DATE" \
         -o cicerone .
```

## Build All Platforms

```bash
# Using make
make build-all

# Manual build for all platforms
make build-linux build-arm

# Or create release tarballs
make release
```

## Cross-Compilation Matrix

| OS | Architecture | Output |
|----|--------------|--------|
| linux | amd64 | cicerone-linux-amd64 |
| linux | arm64 | cicerone-linux-arm64 |
| darwin | amd64 | cicerone-darwin-amd64 |
| darwin | arm64 | cicerone-darwin-arm64 |
| windows | amd64 | cicerone-windows-amd64.exe |
| windows | arm64 | cicerone-windows-arm64.exe |

## Docker Build (Optional)

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cicerone .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/cicerone .
ENTRYPOINT ["./cicerone"]
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific package tests
go test -v ./agent/...
go test -v ./llm/...

# Run E2E tests (requires Ollama)
go test -v -tags=e2e ./tests/...

# Run integration tests
go test -v -tags=integration ./...
```

## Installation

```bash
# Install to /usr/local/bin (requires sudo)
make install

# Or install to custom location
go build -o /path/to/cicerone .
sudo ln -s /path/to/cicerone /usr/local/bin/cicerone

# Verify installation
cicerone version
```

## Development Build

For development with debugging symbols:

```bash
go build -gcflags="all=-N -l" -o cicerone .
```

## Release Build

```bash
# Clean and build release
make clean
VERSION=v1.0.0 make release
```

## Troubleshooting

### "command not found: go"

Install Go from https://go.dev/dl/ or via package manager:

```bash
# Ubuntu/Debian
sudo apt install golang-go

# macOS
brew install go
```

### "module not found"

```bash
go mod download
go mod tidy
```

### Binary too large

```bash
# Strip symbols
go build -ldflags="-s -w" -o cicerone .

# Use UPX for additional compression
upx --best cicerone
```