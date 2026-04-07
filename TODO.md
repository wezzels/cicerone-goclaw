# TODO.md - cicerone-goclaw Development Tasks

## Active Development

### Code Execution & SSH Capabilities

**Status:** In Progress
**Priority:** High
**Started:** 2026-04-07

#### Phase 1: Workspace Management ✅ COMPLETE
- [x] Create `internal/workspace/workspace.go`
- [x] Create `internal/workspace/executor.go`
- [x] Create `internal/workspace/sandbox.go`
- [x] Add `cmd/workspace.go` command
- [x] Add `cmd/exec.go` command
- [x] Tests for workspace package (19 tests PASS)

#### Phase 2: Command Executor ✅ COMPLETE
- [x] Implement `Executor` struct with timeout support
- [x] Add `Run()` for simple commands
- [x] Add `RunInteractive()` for interactive commands
- [x] Add `RunBackground()` for background processes
- [x] Process management (list, kill)
- [x] Tests for executor

#### Phase 3: SSH Client ✅ COMPLETE
- [x] Create `internal/ssh/client.go`
- [x] Create `internal/ssh/config.go`
- [x] Implement `NewClient()` with key auth
- [x] Implement `Exec()` for remote commands
- [x] Implement `Shell()` for interactive shell
- [x] Add tunnel support (`internal/ssh/tunnel.go`)
- [x] Add `cmd/ssh.go` commands (add, list, test, exec, shell, push, pull, remove)
- [x] Tests for SSH client (12 tests PASS)

#### Phase 4: File Transfer ✅ COMPLETE
- [x] Basic file transfer via `CopyFile` in client.go
- [x] Create `internal/ssh/transfer.go` with SFTP support
- [x] Implement `Push()` (local → remote)
- [x] Implement `Pull()` (remote → local)
- [x] Add SFTP dependency (github.com/pkg/sftp)
- [x] Tests for transfer (unit tests + integration structure)

#### Phase 5: Commands Integration ✅ COMPLETE
- [x] `cicerone workspace init [path]`
- [x] `cicerone workspace status`
- [x] `cicerone workspace clean`
- [x] `cicerone exec <command>`
- [x] `cicerone ssh add <name> <host> <user>`
- [x] `cicerone ssh list`
- [x] `cicerone ssh test <name>`
- [x] `cicerone ssh exec <name> <command>`
- [x] `cicerone ssh push/pull`
- [x] `cicerone ssh shell <name>`
- [x] `cicerone ssh remove <name>`
- [x] `cicerone test [--remote <host>]`
- [x] Build and test all commands (44 tests PASS)

#### Phase 6: Testing & Documentation ✅ COMPLETE
- [x] Unit tests passing (44 tests)
- [x] README.md updated with new commands
- [x] TODO.md updated with completion status
- [x] All phases complete

---

## Completed

### v2.1.0 - Code Execution & SSH (2026-04-07)
- [x] Workspace management (workspace package)
- [x] Command executor with timeout support
- [x] Sandbox isolation
- [x] SSH client with key authentication
- [x] SFTP file transfer (push/pull)
- [x] SSH tunnel support
- [x] Remote test execution
- [x] All 44 unit tests passing

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
- [x] CODE_EXEC_ROADMAP.md
- [x] README.md (updated)
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