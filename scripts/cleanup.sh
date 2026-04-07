#!/bin/bash
# scripts/cleanup.sh - Remove unused files for go-only refactor
# Run after Phase 4

set -e

echo "Removing unused files..."

# Admin VM management
rm -f admin.go admin_test.go

# Client and server (API server removed)
rm -f client.go server.go

# Crypto and vault (not needed)
rm -f crypto.go crypto_test.go vault.go

# Docker (not needed)
rm -f docker.go

# VM images (not needed)
rm -f image.go image_test.go

# Installer (not needed - TUI remains in cmd/)
rm -f installer.go installer_gui.go installer_test.go

# RAG library (not needed)
rm -f library.go library_test.go

# GitLab runner (not needed)
rm -f runner_test.go

# Tasks (not needed)
rm -f tasks.go

# OpenAI-specific (keeping ollama/llamacpp)
# Note: llm.go will be replaced by llm/ package

# Unused OpenClaw providers
rm -f openclaw/discord.go
rm -f openclaw/signal.go
rm -f openclaw/whatsapp.go
rm -f openclaw/browser.go

echo "Cleanup complete!"
echo "Files removed: 22"
echo ""
echo "Remaining work:"
echo "  - Refactor main.go to use cmd/ package"
echo "  - Update go.mod dependencies"
echo "  - Run go mod tidy"