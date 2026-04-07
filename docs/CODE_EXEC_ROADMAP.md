# Code Execution & SSH Roadmap

**Goal:** Add workspace code execution and remote SSH capabilities to cicerone-goclaw.

## Overview

cicerone-goclaw will gain the ability to:
1. Execute code locally in a workspace
2. Run tests and build commands
3. SSH into remote machines and execute commands
4. Transfer files between local and remote systems

---

## Architecture

### New Packages

```
cmd/
├── exec.go           # Local execution command
├── ssh.go            # SSH management commands
├── workspace.go      # Workspace management
└── test.go           # Test runner command

internal/
├── workspace/
│   ├── workspace.go   # Workspace management
│   ├── executor.go    # Command executor
│   └── sandbox.go     # Sandbox isolation
├── ssh/
│   ├── client.go      # SSH client wrapper
│   ├── config.go      # SSH configuration
│   ├── tunnel.go      # Tunnel management
│   └── transfer.go    # File transfer (scp/sftp)
└── runner/
    ├── runner.go      # Code runner interface
    ├── go.go          # Go runner
    ├── python.go      # Python runner (optional)
    └── shell.go       # Shell runner
```

---

## Commands

### Workspace Commands

```bash
# Create/initialize workspace
cicerone workspace init [path]

# Show workspace info
cicerone workspace status

# List workspaces
cicerone workspace list

# Clean workspace
cicerone workspace clean
```

### Execution Commands

```bash
# Execute command in workspace
cicerone exec <command>

# Execute with timeout
cicerone exec --timeout 30s "go test ./..."

# Execute in background
cicerone exec --bg "go build"

# List running processes
cicerone exec ps

# Kill process
cicerone exec kill <pid>
```

### SSH Commands

```bash
# Add SSH host
cicerone ssh add <name> <host> <user> [--key ~/.ssh/id_rsa]

# List SSH hosts
cicerone ssh list

# Test connection
cicerone ssh test <name>

# Execute remote command
cicerone ssh exec <name> <command>

# Execute with sudo
cicerone ssh exec <name> --sudo "<command>"

# Copy file to remote
cicerone ssh push <name> <local> <remote>

# Copy file from remote
cicerone ssh pull <name> <remote> <local>

# Interactive shell
cicerone ssh shell <name>

# Tunnel local port
cicerone ssh tunnel <name> -L 8080:localhost:80

# Remove host
cicerone ssh remove <name>
```

### Test Commands

```bash
# Run tests
cicerone test [path]

# Run with coverage
cicerone test --cover

# Run specific test
cicerone test --run TestName

# Run tests on remote
cicerone test --remote <host> [path]
```

---

## Configuration

### SSH Hosts in config.yaml

```yaml
ssh:
  hosts:
    - name: darth
      host: 10.0.0.117
      user: wez
      key: ~/.ssh/id_rsa
      port: 22
      
    - name: miner
      host: 207.244.226.151
      user: wez
      key: ~/.ssh/id_ed25519
      port: 22
      tunnel:
        local: 8080
        remote: localhost:8080

workspace:
  default: ~/workspace
  sandbox: true
  timeout: 60s
```

---

## Implementation Plan

### Phase 1: Workspace Management (2-3 hours)

**File:** `internal/workspace/workspace.go`

```go
package workspace

import (
    "os"
    "path/filepath"
)

type Workspace struct {
    Path    string
    Root    string
    Sandbox bool
}

func New(path string) (*Workspace, error) {
    abs, err := filepath.Abs(path)
    if err != nil {
        return nil, err
    }
    
    return &Workspace{
        Path:    abs,
        Root:    abs,
        Sandbox: true,
    }, nil
}

func (w *Workspace) Init() error {
    dirs := []string{
        w.Path,
        filepath.Join(w.Path, "src"),
        filepath.Join(w.Path, "build"),
        filepath.Join(w.Path, "logs"),
    }
    
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
    }
    
    return nil
}

func (w *Workspace) WriteFile(name string, content []byte) error {
    path := filepath.Join(w.Path, name)
    dir := filepath.Dir(path)
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    return os.WriteFile(path, content, 0644)
}

func (w *Workspace) ReadFile(name string) ([]byte, error) {
    return os.ReadFile(filepath.Join(w.Path, name))
}
```

### Phase 2: Command Executor (2-3 hours)

**File:** `internal/workspace/executor.go`

```go
package workspace

import (
    "bytes"
    "context"
    "os/exec"
    "time"
)

type Executor struct {
    workspace *Workspace
    timeout   time.Duration
    env       []string
}

func NewExecutor(w *Workspace) *Executor {
    return &Executor{
        workspace: w,
        timeout:   60 * time.Second,
        env:       os.Environ(),
    }
}

func (e *Executor) SetTimeout(d time.Duration) {
    e.timeout = d
}

func (e *Executor) SetEnv(env []string) {
    e.env = env
}

func (e *Executor) Run(ctx context.Context, command string, args ...string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, command, args...)
    cmd.Dir = e.workspace.Path
    cmd.Env = e.env
    
    return cmd.CombinedOutput()
}

func (e *Executor) RunInteractive(command string, args ...string) error {
    cmd := exec.Command(command, args...)
    cmd.Dir = e.workspace.Path
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}

func (e *Executor) RunBackground(command string, args ...string) (int, error) {
    cmd := exec.Command(command, args...)
    cmd.Dir = e.workspace.Path
    cmd.Env = e.env
    
    if err := cmd.Start(); err != nil {
        return 0, err
    }
    
    return cmd.Process.Pid, nil
}
```

### Phase 3: SSH Client (3-4 hours)

**File:** `internal/ssh/client.go`

```go
package ssh

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "net"
    "os"
    "time"
    
    "golang.org/x/crypto/ssh"
)

type Client struct {
    client *ssh.Client
    config *Config
}

type Config struct {
    Host    string
    Port    int
    User    string
    KeyPath string
    Timeout time.Duration
}

func NewClient(cfg *Config) (*Client, error) {
    key, err := os.ReadFile(expandHome(cfg.KeyPath))
    if err != nil {
        return nil, fmt.Errorf("failed to read key: %w", err)
    }
    
    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, fmt.Errorf("failed to parse key: %w", err)
    }
    
    config := &ssh.ClientConfig{
        User: cfg.User,
        Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key
        Timeout: cfg.Timeout,
    }
    
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    client, err := ssh.Dial("tcp", addr, config)
    if err != nil {
        return nil, fmt.Errorf("failed to dial: %w", err)
    }
    
    return &Client{
        client: client,
        config: cfg,
    }, nil
}

func (c *Client) Close() error {
    return c.client.Close()
}

func (c *Client) Exec(ctx context.Context, command string) ([]byte, error) {
    session, err := c.client.NewSession()
    if err != nil {
        return nil, err
    }
    defer session.Close()
    
    var stdout, stderr bytes.Buffer
    session.Stdout = &stdout
    session.Stderr = &stderr
    
    done := make(chan error, 1)
    go func() {
        done <- session.Run(command)
    }()
    
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case err := <-done:
        if err != nil {
            return stderr.Bytes(), err
        }
        return stdout.Bytes(), nil
    }
}

func (c *Client) Shell() error {
    session, err := c.client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()
    
    session.Stdin = os.Stdin
    session.Stdout = os.Stdout
    session.Stderr = os.Stderr
    
    modes := ssh.TerminalModes{
        ssh.ECHO:          0,
        ssh.TTY_OP_ISPEED: 14400,
        ssh.TTY_OP_OSPEED: 14400,
    }
    
    fd := int(os.Stdin.Fd())
    term := os.Getenv("TERM")
    
    if err := session.RequestPty(term, 80, 40, modes); err != nil {
        return err
    }
    
    return session.Shell()
}
```

### Phase 4: File Transfer (2 hours)

**File:** `internal/ssh/transfer.go`

```go
package ssh

import (
    "io"
    "os"
    "path/filepath"
    
    "github.com/pkg/sftp"
)

func (c *Client) Push(localPath, remotePath string) error {
    client, err := sftp.NewClient(c.client)
    if err != nil {
        return err
    }
    defer client.Close()
    
    local, err := os.Open(localPath)
    if err != nil {
        return err
    }
    defer local.Close()
    
    remote, err := client.Create(remotePath)
    if err != nil {
        return err
    }
    defer remote.Close()
    
    _, err = io.Copy(remote, local)
    return err
}

func (c *Client) Pull(remotePath, localPath string) error {
    client, err := sftp.NewClient(c.client)
    if err != nil {
        return err
    }
    defer client.Close()
    
    remote, err := client.Open(remotePath)
    if err != nil {
        return err
    }
    defer remote.Close()
    
    if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
        return err
    }
    
    local, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer local.Close()
    
    _, err = io.Copy(local, remote)
    return err
}
```

---

## Security Considerations

1. **Sandboxing** - Workspace operations should be sandboxed
2. **Command Whitelisting** - Optional whitelist for allowed commands
3. **Timeout Enforcement** - All commands must have timeouts
4. **SSH Key Protection** - Keys should be loaded securely
5. **Audit Logging** - All executions should be logged

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

## Success Criteria

- [ ] `cicerone exec` runs local commands
- [ ] `cicerone ssh add` stores host configuration
- [ ] `cicerone ssh exec` runs remote commands
- [ ] `cicerone ssh push/pull` transfers files
- [ ] `cicerone test` runs tests locally and remotely
- [ ] All commands have timeout enforcement
- [ ] SSH connections are properly closed

---

## Timeline

| Phase | Description | Hours |
|-------|-------------|-------|
| 1 | Workspace management | 2-3 |
| 2 | Command executor | 2-3 |
| 3 | SSH client | 3-4 |
| 4 | File transfer | 2 |
| 5 | Commands integration | 2-3 |
| 6 | Testing | 2 |
| **Total** | | **13-17 hours** |