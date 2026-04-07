.PPHONY: all build test clean install run help

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -X github.com/crab-meat-repos/cicerone-goclaw/cmd.Version=$(VERSION) \
           -X github.com/crab-meat-repos/cicerone-goclaw/cmd.Commit=$(COMMIT) \
           -X github.com/crab-meat-repos/cicerone-goclaw/cmd.Date=$(DATE)

# Binary name
BINARY := cicerone

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Directories
DOCS_DIR := docs
SCRIPTS_DIR := scripts

all: clean build

## build: Build the binary
build:
	@echo "Building $(BINARY)..."
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "Built: $(BINARY) $(VERSION) ($(COMMIT))"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## coverage: Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin)
	golangci-lint run ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

## install: Install binary to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin..."
	sudo cp $(BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY)
	@echo "Installed: $(BINARY)"

## uninstall: Remove binary from /usr/local/bin
uninstall:
	@echo "Uninstalling..."
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled: $(BINARY)"

## run: Run the binary
run: build
	./$(BINARY)

## doctor: Run health check
doctor: build
	./$(BINARY) doctor

## config: Create default config
config:
	@mkdir -p ~/.cicerone
	@cp $(DOCS_DIR)/config.example.yaml ~/.cicerone/config.yaml 2>/dev/null || echo "Config already exists"

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## build-linux: Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 .

## build-arm: Build for ARM
build-arm:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64 .

## build-all: Build for all platforms
build-all: build-linux build-arm

## release: Create release binaries
release: clean build-all
	@echo "Creating release..."
	tar -czf $(BINARY)-$(VERSION)-linux-amd64.tar.gz $(BINARY)-linux-amd64
	tar -czf $(BINARY)-$(VERSION)-linux-arm64.tar.gz $(BINARY)-linux-arm64
	@echo "Release packages created"

## version: Show version info
version: build
	./$(BINARY) version

## help: Show this help
help:
	@echo "Makefile for cicerone"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'

.PHONY: all build test coverage lint clean install uninstall run doctor config deps build-linux build-arm build-all release version help