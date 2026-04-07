# Config Wizard Roadmap

**Goal:** Add interactive setup wizard (`cicerone config wizard`) with text-based menu system for first-time configuration.

## Overview

The wizard will guide users through:
1. LLM Provider Setup (Ollama/llama.cpp)
2. Telegram Bot Configuration
3. Gateway Settings
4. Optional Features (TTS, plugins)
5. Security Hardening

## Feature Sections

### Section 1: LLM Configuration (--section:llm)

```
╔═══════════════════════════════════════════════════════════════╗
║                    LLM Configuration                           ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Select your LLM provider:                                     ║
║                                                                ║
║    1. Ollama (recommended)                                     ║
║    2. llama.cpp server                                         ║
║    3. OpenAI-compatible API                                    ║
║    4. Skip (configure later)                                   ║
║                                                                ║
║  Choice [1-4]: _                                              ║
╚═══════════════════════════════════════════════════════════════╝
```

**Ollama Setup Flow:**
```
╔═══════════════════════════════════════════════════════════════╗
║                    Ollama Setup                                ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Checking for Ollama...                                        ║
║  ✓ Ollama found at http://localhost:11434                      ║
║  ✓ Ollama version: 0.19.0                                      ║
║                                                                ║
║  Available models:                                             ║
║    1. gemma3:12b (7.0 GB) ✓                                   ║
║    2. mistral:latest (4.1 GB)                                  ║
║    3. qwen3:0.6b (0.5 GB)                                      ║
║    4. Download new model...                                    ║
║    5. Use custom model name                                    ║
║                                                                ║
║  Select model [1-5]: 1                                         ║
║                                                                ║
║  ✓ Selected: gemma3:12b                                        ║
║  ✓ Model is pulled and ready                                   ║
║                                                                ║
║  Test generation? [Y/n]: Y                                     ║
║  Testing... "Say 'hello'" → "Hello! How can I help..."        ║
║  ✓ Generation test passed                                      ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

**llama.cpp Setup Flow:**
```
╔═══════════════════════════════════════════════════════════════╗
║                  llama.cpp Setup                              ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Enter llama.cpp server URL [http://localhost:8080]: _        ║
║                                                                ║
║  Checking connection...                                        ║
║  ✓ Connected to llama.cpp server                               ║
║                                                                ║
║  Available models:                                              ║
║    1. local-model                                              ║
║                                                                ║
║  Model name [local-model]: _                                   ║
║                                                                ║
║  Test generation? [Y/n]: Y                                     ║
║  ✓ Generation test passed                                       ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### Section 2: Telegram Configuration

```
╔═══════════════════════════════════════════════════════════════╗
║                 Telegram Bot Setup                             ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  To use Telegram, you need a bot token from @BotFather.        ║
║                                                                ║
║  Steps to get a token:                                         ║
║    1. Open Telegram and search for @BotFather                  ║
║    2. Send /newbot command                                     ║
║    3. Follow the prompts                                       ║
║    4. Copy the token below                                     ║
║                                                                ║
║  Bot token: _                                                  ║
║                                                                ║
║  Validating token...                                           ║
║  ✓ Bot token valid                                             ║
║  ✓ Bot: @YourBotName (ID: 123456789)                           ║
║                                                                ║
║  Restrict to specific users? [y/N]: y                          ║
║  Enter allowed user IDs (comma-separated): 123456789,987654321 ║
║                                                                ║
║  ✓ Telegram configured                                         ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### Section 3: Gateway Settings

```
╔═══════════════════════════════════════════════════════════════╗
║                   Gateway Settings                              ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Listen address [127.0.0.1:8080]: _                            ║
║                                                                ║
║  Enable WebSocket API? [y/N]: n                                ║
║  Enable session persistence? [Y/n]: y                          ║
║  Session database [./data/sessions.db]: _                       ║
║                                                                ║
║  ✓ Gateway configured                                          ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### Section 4: Optional Features

```
╔═══════════════════════════════════════════════════════════════╗
║                   Optional Features                             ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Select features to enable:                                     ║
║                                                                ║
║    [ ] TTS (Text-to-Speech)                                     ║
║    [ ] Web Search Plugin                                        ║
║    [ ] Scheduler (Cron jobs)                                   ║
║    [ ] Node Pairing (Distributed)                              ║
║                                                                ║
║  Press SPACE to toggle, ENTER to continue                       ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### Section 5: Security Hardening

```
╔═══════════════════════════════════════════════════════════════╗
║                   Security Settings                             ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Enable rate limiting? [Y/n]: y                                ║
║  Max requests per minute [60]: _                               ║
║  Max failed attempts [5]: _                                    ║
║  Block duration (minutes) [30]: _                              ║
║                                                                ║
║  Run security audit now? [Y/n]: y                              ║
║                                                                ║
║  🔒 Security Audit                                              ║
║  =================                                              ║
║  [!] SSH: Password auth enabled                                 ║
║  [?] Firewall: UFW not available                               ║
║  [✓] File Permissions: Protected                               ║
║                                                                ║
║  Fix issues automatically? [y/N]: n                            ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### Section 6: Review & Save

```
╔═══════════════════════════════════════════════════════════════╗
║                   Configuration Review                          ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  LLM:                                                          ║
║    Provider: ollama                                            ║
║    URL: http://localhost:11434                                 ║
║    Model: gemma3:12b                                           ║
║                                                                ║
║  Telegram:                                                     ║
║    Bot Token: 123456789:ABC... (configured)                    ║
║    Allowed Users: 123456789, 987654321                        ║
║                                                                ║
║  Gateway:                                                      ║
║    Listen: 127.0.0.1:8080                                      ║
║    Sessions: ./data/sessions.db                                ║
║                                                                ║
║  Features:                                                     ║
║    TTS: disabled                                               ║
║    Scheduler: enabled                                          ║
║                                                                ║
║  Security:                                                     ║
║    Rate Limit: 60/min                                         ║
║                                                                ║
║  Save to ~/.cicerone/config.yaml? [Y/n]: y                     ║
║  ✓ Configuration saved                                          ║
║                                                                ║
║  Run 'cicerone doctor' to verify setup? [Y/n]: y               ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

---

## Implementation Plan

### Phase 1: Wizard Framework (2-3 hours)

**File:** `cmd/wizard.go`

```go
package cmd

// Wizard manages the interactive configuration
type Wizard struct {
    config   *Config
    sections  []Section
    current   int
    term      *terminal.Terminal
}

type Section interface {
    Name() string
    Run(w *Wizard) error
    Validate() error
}

// Sections in order
var sections = []Section{
    &LLMSection{},
    &TelegramSection{},
    &GatewaySection{},
    &FeaturesSection{},
    &SecuritySection{},
    &ReviewSection{},
}
```

### Phase 2: LLM Section (2 hours)

**File:** `cmd/wizard_llm.go`

```go
package cmd

type LLMSection struct{}

func (s *LLMSection) Name() string { return "LLM Configuration" }

func (s *LLMSection) Run(w *Wizard) error {
    // 1. Detect available providers
    // 2. Show provider menu
    // 3. Test connection
    // 4. List/download models
    // 5. Test generation
    return nil
}

func (s *LLMSection) Validate() error {
    // Ensure provider and model are set
    return nil
}
```

### Phase 3: Telegram Section (1 hour)

**File:** `cmd/wizard_telegram.go`

```go
package cmd

type TelegramSection struct{}

func (s *TelegramSection) Name() string { return "Telegram Bot Setup" }

func (s *TelegramSection) Run(w *Wizard) error {
    // 1. Prompt for bot token
    // 2. Validate with Telegram API
    // 3. Show bot info
    // 4. Optionally restrict users
    return nil
}
```

### Phase 4: Other Sections (2-3 hours)

- `cmd/wizard_gateway.go` - Gateway settings
- `cmd/wizard_features.go` - Optional features
- `cmd/wizard_security.go` - Security hardening
- `cmd/wizard_review.go` - Review and save

### Phase 5: TUI Enhancements (1-2 hours)

- Add color support
- Add progress indicators
- Add keyboard navigation
- Add help screens
- Add back/forward navigation

---

## Command Structure

```bash
# Run full wizard
cicerone config wizard

# Run specific section
cicerone config wizard --section llm
cicerone config wizard --section telegram
cicerone config wizard --section security

# Non-interactive mode (for scripts)
cicerone config wizard --non-interactive \
    --llm-provider ollama \
    --llm-model gemma3:12b \
    --telegram-token "YOUR_TOKEN"

# Reset configuration
cicerone config reset

# Validate configuration
cicerone config validate
```

---

## File Structure

```
cmd/
├── wizard.go           # Main wizard framework
├── wizard_llm.go       # LLM configuration section
├── wizard_telegram.go  # Telegram bot section
├── wizard_gateway.go   # Gateway settings section
├── wizard_features.go  # Optional features section
├── wizard_security.go  # Security hardening section
├── wizard_review.go    # Review and save section
├── config.go           # Config file operations
└── config_cmd.go       # config show/set commands

internal/
├── tui/
│   ├── terminal.go     # Terminal handling
│   ├── menu.go         # Menu rendering
│   ├── input.go        # Input handling
│   └── colors.go       # Color definitions
└── validate/
    ├── llm.go          # LLM validation
    ├── telegram.go     # Telegram validation
    └── config.go       # Config validation
```

---

## Dependencies

Add to `go.mod`:
```go
require (
    golang.org/x/term v0.16.0  // Terminal handling
)
```

---

## Success Criteria

- [ ] `cicerone config wizard` runs interactive setup
- [ ] LLM section detects Ollama/llama.cpp automatically
- [ ] Telegram section validates token with API
- [ ] All sections can run independently with `--section`
- [ ] Non-interactive mode works for scripting
- [ ] Configuration is saved to `~/.cicerone/config.yaml`
- [ ] `cicerone doctor` passes after wizard completion
- [ ] Works on Linux and macOS (Windows optional)

---

## Timeline

| Phase | Description | Hours |
|-------|-------------|-------|
| 1 | Wizard framework | 2-3 |
| 2 | LLM section | 2 |
| 3 | Telegram section | 1 |
| 4 | Other sections | 2-3 |
| 5 | TUI enhancements | 1-2 |
| **Total** | | **8-11 hours** |

---

## Next Steps

1. Create `cmd/wizard.go` with framework
2. Implement LLM section with auto-detection
3. Implement Telegram section with API validation
4. Add remaining sections
5. Test full wizard flow
6. Add to CI/CD pipeline