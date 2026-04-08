# Cicerone-GoClaw Test Suite

This directory contains automated and manual tests for cicerone-goclaw.

## Prerequisites

- Go 1.25+
- Ollama running locally (default: `http://localhost:11434`)
- A model pulled in Ollama (e.g., `ollama pull gemma3:12b`)

## Running Tests

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
# Requires Ollama running
go test -tags=integration ./tests/...
```

### Manual Verification
See `MANUAL_TESTS.md` for step-by-step verification procedures.

## Test Categories

1. **Unit Tests** (`*_test.go`) - Fast, isolated tests
2. **Integration Tests** (`integration_*.go`) - Tests requiring external services
3. **E2E Tests** (`e2e_*.go`) - End-to-end workflow tests
4. **Manual Tests** (`MANUAL_TESTS.md`) - Human-verified procedures