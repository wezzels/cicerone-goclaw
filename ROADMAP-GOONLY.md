# Cicerone Go-Only Roadmap

**Goal:** Simplify cicerone to run 100% Go with minimal dependencies and focused feature set.

## Target Feature Set

| Feature | Status | Notes |
|---------|--------|-------|
| Telegram | KEEP | Primary messaging interface |
| TUI | KEEP | Terminal UI for interactive use |
| Gateway restart | KEEP | `cicerone gateway restart` |
| Doctor | KEEP | `cicerone doctor` - health checks |
| Security | KEEP | `cicerone security` - security audit |
| LLM | KEEP | Ollama + llama.cpp only |
| Discord | REMOVE | |
| Signal | REMOVE | |
| WhatsApp | REMOVE | |
| Browser | REMOVE | |
| GitLab runner | REMOVE | runner, pipeline, node commands |
| RAG library | REMOVE | library command |
| VM images | REMOVE | image command |
| Admin VMs | REMOVE | admin command |
| API server | REMOVE | serve command |
| Vault | REMOVE | vault.go, crypto.go |
| Tasks | REMOVE | tasks.go |
| Docker | REMOVE | docker.go |
| Installer | REMOVE | installer*.go |

---

## Phase 1: Analyze and Document (1-2 hours)

### 1.1 Current Codebase Analysis

| File | Lines | Action |
|------|-------|--------|
| main.go | 3363 | Refactor - remove 70% of commands |
| llm.go | 544 | Keep, simplify to ollama+llamacpp only |
| chat.go | 744 | Keep, already uses llama.cpp |
| openclaw_cmd.go | 141 | Keep |
| openclaw/*.go | ~15 files | Refactor - remove non-telegram providers |
| admin.go | 1057 | DELETE |
| server.go | 529 | DELETE |
| library.go | 906 | DELETE |
| image.go | 1099 | DELETE |
| installer.go | 842 | DELETE |
| installer_gui.go | 278 | DELETE |
| installer_tui.go | 575 | KEEP (rename to tui.go, simplify) |
| docker.go | 146 | DELETE |
| crypto.go | 312 | DELETE |
| vault.go | 438 | DELETE |
| tasks.go | 336 | DELETE |
| client.go | 449 | Review - may keep for API client |

**Total to remove:** ~6,000 lines
**Total to keep/refactor:** ~5,000 lines

### 1.2 Dependency Analysis

Current go.mod dependencies to REMOVE:
```go
github.com/bwmarrin/discordgo        // Discord - REMOVE
github.com/chromedp/chromedp          // Browser - REMOVE
```

Dependencies to KEEP:
```go
github.com/go-telegram-bot-api/telegram-bot-api/v5  // Telegram
github.com/spf13/cobra              // CLI
github.com/spf13/viper              // Config
github.com/gorilla/websocket         // WebSocket (for ollama)
golang.org/x/crypto                 // SSH (for do command)
```

---

## Phase 2: Create New Command Structure (2-3 hours)

### 2.1 New Command Tree

```
cicerone
├── telegram          # Start Telegram bot (was openclaw)
├── tui               # Interactive TUI mode
├── gateway
│   ├── restart       # Restart OpenClaw gateway
│   └── status        # Check gateway status
├── doctor            # Health check diagnostics
├── security          # Security audit
├── llm
│   ├── show          # Show LLM config
│   ├── test          # Test LLM connection
│   └── models        # List available models
├── do                # Execute command via LLM
├── chat              # Interactive LLM chat
├── config
│   ├── show          # Show config
│   └── set           # Set config value
└── version           # Show version
```

### 2.2 Refactored main.go Structure

```go
package main

// Root command
var rootCmd = &cobra.Command{
    Use:   "cicerone",
    Short: "Cicerone - Go-only messaging gateway",
}

// Subcommands
var telegramCmd = &cobra.Command{
    Use:   "telegram",
    Short: "Start Telegram bot",
    RunE:  runTelegram,
}

var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Launch interactive TUI",
    RunE:  runTUI,
}

var gatewayCmd = &cobra.Command{
    Use:   "gateway",
    Short: "Gateway management",
}

var doctorCmd = &cobra.Command{
    Use:   "doctor",
    Short: "Run health diagnostics",
    RunE:  runDoctor,
}

var securityCmd = &cobra.Command{
    Use:   "security",
    Short: "Security audit",
    RunE:  runSecurity,
}

var llmCmd = &cobra.Command{
    Use:   "llm",
    Short: "LLM management",
}

var doCmd = &cobra.Command{
    Use:   "do [instructions]",
    Short: "Execute via LLM",
    RunE:  runDo,
}

var chatCmd = &cobra.Command{
    Use:   "chat",
    Short: "Interactive LLM chat",
    RunE:  runChat,
}
```

---

## Phase 3: Refactor LLM Package (2-3 hours)

### 3.1 LLM Provider Interface

```go
// llm.go - Simplified LLM package
package main

type LLMProvider interface {
    Generate(prompt string) (string, error)
    Chat(messages []ChatMessage) (string, error)
    Models() ([]string, error)
    IsRunning() bool
}

// OllamaProvider - Direct Ollama API
type OllamaProvider struct {
    BaseURL string
    Model   string
    Timeout time.Duration
}

// LlamaCPPProvider - Direct llama.cpp server
type LlamaCPPProvider struct {
    BaseURL string
    Model   string
    Port    int
}
```

### 3.2 Remove OpenAI Support

Delete all OpenAI-specific code:
- Remove `apiKey` field from LLMConfig
- Remove OpenAI API endpoint handling
- Remove bearer token auth
- Keep only Ollama `/api/generate` and `/api/chat`

### 3.3 llama.cpp Integration

Keep existing chat.go implementation:
- Start llama.cpp server if not running
- Use OpenAI-compatible API on localhost
- Support GGUF model loading

---

## Phase 4: Refactor Telegram Package (2-3 hours)

### 4.1 Telegram-Only Gateway

Refactor `openclaw/` directory:

```go
// openclaw/telegram.go
package openclaw

type TelegramBot struct {
    Token     string
    Debug     bool
    AllowedUsers []int64
    llm       LLMProvider
}

func (b *TelegramBot) Start(ctx context.Context) error {
    // Connect to Telegram
    // Process messages
    // Call LLM for responses
}
```

### 4.2 Files to Delete

| File | Reason |
|------|--------|
| discord.go | Discord support |
| signal.go | Signal support |
| whatsapp.go | WhatsApp support |
| browser.go | Chrome automation |
| plugin_tts.go | TTS plugin (optional keep) |
| plugin_websearch.go | Web search (optional keep) |

### 4.3 Simplified Config

```yaml
# config.yaml - Simplified
telegram:
  enabled: true
  bot_token: "${TELEGRAM_BOT_TOKEN}"
  allowed_users:
    - 8318706992

llm:
  provider: ollama  # or llamacpp
  base_url: "http://localhost:11434"
  model: "gemma3:12b"
  
gateway:
  listen: "127.0.0.1:8080"
  
logging:
  level: info
```

---

## Phase 5: Implement Doctor Command (1-2 hours)

### 5.1 Health Checks

```go
// doctor.go
package main

func runDoctor(cmd *cobra.Command, args []string) error {
    fmt.Println("🏥 Cicerone Health Check")
    fmt.Println("========================")
    
    checks := []DoctorCheck{
        {"Config", checkConfig},
        {"Telegram Token", checkTelegramToken},
        {"LLM Connection", checkLLMConnection},
        {"Ollama Status", checkOllamaStatus},
        {"Model Available", checkModelAvailable},
        {"Network", checkNetwork},
        {"Disk Space", checkDiskSpace},
        {"Memory", checkMemory},
    }
    
    for _, check := range checks {
        status, detail := check.Run()
        printStatus(check.Name, status, detail)
    }
    
    return nil
}

type DoctorCheck struct {
    Name string
    Run  func() (Status, string)
}

type Status int
const (
    StatusOK Status = iota
    StatusWarn
    StatusFail
)
```

### 5.2 Doctor Checks to Implement

1. **Config** - Check `~/.cicerone/config.yaml` exists and valid
2. **Telegram Token** - Validate token format, check bot info
3. **LLM Connection** - Ping Ollama/llama.cpp server
4. **Ollama Status** - Check if ollama serve is running
5. **Model Available** - Verify configured model exists
6. **Network** - Check internet connectivity
7. **Disk Space** - Check available disk space
8. **Memory** - Check available memory

---

## Phase 6: Implement Security Command (1-2 hours)

### 6.1 Security Audit

```go
// security.go
package main

func runSecurity(cmd *cobra.Command, args []string) error {
    fmt.Println("🔒 Security Audit")
    fmt.Println("=================")
    
    audits := []SecurityCheck{
        {"SSH Config", auditSSH},
        {"Firewall (UFW)", auditFirewall},
        {"Open Ports", auditPorts},
        {"User Permissions", auditUsers},
        {"File Permissions", auditFiles},
        {"Package Updates", auditUpdates},
        {"Failed Logins", auditLogins},
    }
    
    for _, audit := range audits {
        result := audit.Run()
        printResult(audit.Name, result)
    }
    
    return nil
}
```

### 6.2 Security Checks to Implement

1. **SSH Config** - Password auth, root login, key-only
2. **Firewall** - UFW status, allowed ports
3. **Open Ports** - Listening services
4. **User Permissions** - Sudo users, wheel group
5. **File Permissions** - Sensitive files
6. **Package Updates** - Security updates pending
7. **Failed Logins** - Recent auth failures

---

## Phase 7: TUI Refactoring (1-2 hours)

### 7.1 Simplified TUI

Keep installer_tui.go but rename to tui.go:
- Remove installer-specific code
- Add main menu with options
- Add LLM status display
- Add gateway controls
- Add security quick check

### 7.2 TUI Menu Structure

```
╔═══════════════════════════════════════╗
║           CICERONE TUI                ║
╠═══════════════════════════════════════╣
║  1. Start Telegram Bot                ║
║  2. Start LLM Chat                    ║
║  3. Run Doctor                        ║
║  4. Run Security Audit                ║
║  5. Restart Gateway                   ║
║  6. View Status                       ║
║  7. Settings                          ║
║  Q. Quit                              ║
╚═══════════════════════════════════════╝
```

---

## Phase 8: Gateway Management (1 hour)

### 8.1 Gateway Commands

```go
// gateway.go
package main

var gatewayRestartCmd = &cobra.Command{
    Use:   "restart",
    Short: "Restart OpenClaw gateway",
    RunE:  runGatewayRestart,
}

var gatewayStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Check gateway status",
    RunE:  runGatewayStatus,
}

func runGatewayRestart(cmd *cobra.Command, args []string) error {
    // 1. Find gateway process
    // 2. Send SIGTERM
    // 3. Wait for shutdown
    // 4. Start new gateway process
    // 5. Verify it's running
    return nil
}

func runGatewayStatus(cmd *cobra.Command, args []string) error {
    // 1. Check if process running
    // 2. Check API endpoint
    // 3. Check Telegram connection
    // 4. Check LLM connection
    return nil
}
```

---

## Phase 9: Build and Test (2-3 hours)

### 9.1 Build Script

```bash
#!/bin/bash
# build.sh

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.date=${DATE}" \
    -o cicerone .

echo "Built: cicerone ${VERSION} (${COMMIT})"
```

### 9.2 Test Plan

| Test | Command | Expected |
|------|---------|----------|
| Version | `./cicerone version` | Shows version |
| Doctor | `./cicerone doctor` | Health checks pass |
| Security | `./cicerone security` | Security audit runs |
| Telegram | `./cicerone telegram` | Bot starts |
| Chat | `./cicerone chat` | LLM chat starts |
| Gateway | `./cicerone gateway status` | Status shown |
| TUI | `./cicerone tui` | TUI launches |

### 9.3 Integration Tests

```go
// main_test.go
func TestDoctorCommand(t *testing.T) {
    buf := new(bytes.Buffer)
    rootCmd.SetOut(buf)
    rootCmd.SetArgs([]string{"doctor"})
    rootCmd.Execute()
    
    output := buf.String()
    if !strings.Contains(output, "Health Check") {
        t.Error("Doctor command failed")
    }
}

func TestLLMConnection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    provider := NewOllamaProvider("http://localhost:11434", "gemma3:12b")
    if !provider.IsRunning() {
        t.Error("Ollama not running")
    }
}
```

---

## Phase 10: Documentation (1 hour)

### 10.1 README.md

```markdown
# Cicerone

Go-only messaging gateway with LLM integration.

## Features

- **Telegram Bot** - Full bot API support
- **LLM Chat** - Ollama and llama.cpp integration
- **Doctor** - Health diagnostics
- **Security** - Security audit
- **TUI** - Interactive terminal interface

## Quick Start

```bash
# Build
go build -o cicerone .

# Run doctor
./cicerone doctor

# Start Telegram bot
./cicerone telegram

# Interactive chat
./cicerone chat

# Security audit
./cicerone security
```

## Configuration

Edit `~/.cicerone/config.yaml`:

```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN"
  
llm:
  provider: ollama
  base_url: "http://localhost:11434"
  model: "gemma3:12b"
```

## Requirements

- Go 1.22+
- Ollama (for LLM) or llama.cpp server
- Telegram bot token
```

---

## Estimated Timeline

| Phase | Description | Time |
|-------|-------------|------|
| 1 | Analyze and Document | 1-2 hours |
| 2 | New Command Structure | 2-3 hours |
| 3 | Refactor LLM Package | 2-3 hours |
| 4 | Refactor Telegram | 2-3 hours |
| 5 | Doctor Command | 1-2 hours |
| 6 | Security Command | 1-2 hours |
| 7 | TUI Refactoring | 1-2 hours |
| 8 | Gateway Management | 1 hour |
| 9 | Build and Test | 2-3 hours |
| 10 | Documentation | 1 hour |
| **Total** | | **14-22 hours** |

---

## Files to Delete

```bash
# Remove unused files
rm -f admin.go admin_test.go
rm -f server.go
rm -f library.go library_test.go
rm -f image.go image_test.go
rm -f installer.go installer_gui.go
rm -f docker.go
rm -f crypto.go crypto_test.go
rm -f vault.go
rm -f tasks.go
rm -f client.go  # May keep for API client

# Remove unused OpenClaw providers
rm -f openclaw/discord.go
rm -f openclaw/signal.go
rm -f openclaw/whatsapp.go
rm -f openclaw/browser.go
```

---

## New go.mod

```go
module github.com/crab-meat-repos/cicerone

go 1.22

require (
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/gorilla/websocket v1.5.0
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.17.0
    golang.org/x/crypto v0.14.0
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## Success Criteria

- [ ] Single binary with no external dependencies (except libc)
- [ ] Telegram bot functional
- [ ] LLM chat works with Ollama and llama.cpp
- [ ] Doctor command passes all checks
- [ ] Security audit completes successfully
- [ ] TUI launches and navigates
- [ ] Gateway restart works
- [ ] Build size < 20MB
- [ ] Memory usage < 100MB idle
- [ ] All tests pass