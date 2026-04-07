# TODO.md - cicerone-goclaw Development Tasks

## Active Development

### Code Execution & SSH Capabilities

**Status:** In Progress
**Priority:** High
**Started:** 2026-04-07

#### Phase 1: Workspace Management (2-3 hours) ✅ Started
- [x] Create `internal/workspace/workspace.go`
- [ ] Create `internal/workspace/executor.go`
- [ ] Create `internal/workspace/sandbox.go`
- [ ] Add `cmd/workspace.go` command
- [ ] Add `cmd/exec.go` command
- [ ] Tests for workspace package

#### Phase 2: Command Executor (2-3 hours)
- [ ] Implement `Executor` struct with timeout support
- [ ] Add `Run()` for simple commands
- [ ] Add `RunInteractive()` for interactive commands
- [ ] Add `RunBackground()` for background processes
- [ ] Process management (list, kill)
- [ ] Tests for executor

#### Phase 3: SSH Client (3-4 hours)
- [ ] Create `internal/ssh/client.go`
- [ ] Create `internal/ssh/config.go`
- [ ] Implement `NewClient()` with key auth
- [ ] Implement `Exec()` for remote commands
- [ ] Implement `Shell()` for interactive shell
- [ ] Add tunnel support (`internal/ssh/tunnel.go`)
- [ ] Tests for SSH client

#### Phase 4: File Transfer (2 hours)
- [ ] Create `internal/ssh/transfer.go`
- [ ] Implement `Push()` (local → remote)
- [ ] Implement `Pull()` (remote → local)
- [ ] Add SFTP dependency
- [ ] Tests for transfer

#### Phase 5: Commands Integration (2-3 hours)
- [ ] `cicerone workspace init [path]`
- [ ] `cicerone workspace status`
- [ ] `cicerone workspace clean`
- [ ] `cicerone exec <command>`
- [ ] `cicerone ssh add <name> <host> <user>`
- [ ] `cicerone ssh list`
- [ ] `cicerone ssh test <name>`
- [ ] `cicerone ssh exec <name> <command>`
- [ ] `cicerone ssh push/pull`
- [ ] `cicerone ssh shell <name>`
- [ ] `cicerone test [--remote <host>]`

#### Phase 6: Testing & Documentation (2 hours)
- [ ] Integration tests
- [ ] Update README.md
- [ ] Update INSTALLATION.md
- [ ] Add examples to docs

---

## Completed

### v2.0.0 - Initial Go-Only Release (2026-04-07)
- [x] Refactor to Go-only codebase
- [x] LLM integration (Ollama, llama.cpp)
- [x] Telegram bot support
- [x] Doctor health checks
- [x] Security audit
- [x] Interactive TUI
- [x] Config wizard

### Documentation (2026-04-07)
- [x] INSTALLATION.md
- [x] TEST_RESULTS.md
- [x] config.example.yaml
- [x] CONFIG_WIZARD_ROADMAP.md
- [x] Makefile

---

## Future Considerations

### Low Priority
- [ ] Python runner (`internal/runner/python.go`)
- [ ] Node.js runner
- [ ] Docker container execution
- [ ] Kubernetes job execution
- [ ] Web-based workspace viewer
- [ ] Real-time log streaming

### Nice to Have
- [ ] Workspace templates
- [ ] Git integration
- [ ] Environment management
- [ ] Secret management (Vault integration)

---

## Dependencies to Add

```go
require (
    golang.org/x/crypto v0.21.0  // SSH
    golang.org/x/term v0.16.0    // Terminal
    github.com/pkg/sftp v1.13.6  // SFTP
)
```

---

## Security Checklist

- [ ] All commands have timeout enforcement
- [ ] SSH connections are properly closed
- [ ] Workspace operations are sandboxed
- [ ] Command whitelisting (optional)
- [ ] Audit logging for all executions
- [ ] SSH key protection
- [ ] Input sanitization

---

## Timeline Estimate

| Phase | Description | Hours | Status |
|-------|-------------|-------|--------|
| 1 | Workspace management | 2-3 | Started |
| 2 | Command executor | 2-3 | Pending |
| 3 | SSH client | 3-4 | Pending |
| 4 | File transfer | 2 | Pending |
| 5 | Commands integration | 2-3 | Pending |
| 6 | Testing | 2 | Pending |
| **Total** | | **13-17** | **~10%** |

---

*Last Updated: 2026-04-07*