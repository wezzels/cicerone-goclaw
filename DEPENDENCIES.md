# Cicerone Dependencies Analysis

**Generated:** 2026-04-06
**Go Version:** 1.22.2

---

## Current Dependencies (go.mod)

### Direct Dependencies

| Dependency | Version | Purpose | Status |
|------------|---------|---------|--------|
| github.com/bwmarrin/discordgo | v0.29.0 | Discord bot API | REMOVE |
| github.com/chromedp/chromedp | v0.9.5 | Chrome automation | REMOVE |
| github.com/go-telegram-bot-api/telegram-bot-api/v5 | v5.5.1 | Telegram bot API | KEEP |
| github.com/gorilla/websocket | v1.5.0 | WebSocket client | KEEP |
| github.com/mattn/go-sqlite3 | v1.14.17 | SQLite driver | REVIEW |
| github.com/mitchellh/go-homedir | v1.1.0 | Home directory | KEEP |
| github.com/robfig/cron/v3 | v3.0.1 | Cron scheduler | KEEP |
| github.com/spf13/cobra | v1.8.0 | CLI framework | KEEP |
| github.com/spf13/viper | v1.17.0 | Config management | KEEP |
| golang.org/x/crypto | v0.14.0 | Crypto/SSH | KEEP |
| gopkg.in/yaml.v3 | v3.0.1 | YAML parsing | KEEP |

### Indirect Dependencies (via above)

| Dependency | Version | Required By |
|------------|---------|-------------|
| github.com/chromedp/cdproto | v0.0.0-20240202 | chromedp |
| github.com/chromedp/sysutil | v1.0.0 | chromedp |
| github.com/fsnotify/fsnotify | v1.6.0 | viper |
| github.com/gobwas/httphead | v0.1.0 | gorilla/websocket |
| github.com/gobwas/pool | v0.2.1 | gorilla/websocket |
| github.com/gobwas/ws | v1.3.2 | gorilla/websocket |
| github.com/hashicorp/hcl | v1.0.0 | viper |
| github.com/inconshreveable/mousetrap | v1.1.0 | cobra |
| github.com/josharian/intern | v1.0.0 | chromedp/cdproto |
| github.com/magiconair/properties | v1.8.7 | viper |
| github.com/mailru/easyjson | v0.7.7 | chromedp/cdproto |
| github.com/mitchellh/mapstructure | v1.5.0 | viper |
| github.com/pelletier/go-toml/v2 | v2.1.0 | viper |
| github.com/sagikazarmark/locafero | v0.3.0 | viper |
| github.com/sagikazarmark/slog-shim | v0.1.0 | viper |
| github.com/sourcegraph/conc | v0.3.0 | chromedp |
| github.com/spf13/afero | v1.10.0 | viper |
| github.com/spf13/cast | v1.5.1 | viper |
| github.com/spf13/pflag | v1.0.5 | cobra |
| github.com/subosito/gotenv | v1.6.0 | viper |
| go.uber.org/atomic | v1.9.0 | chromedp |
| go.uber.org/multierr | v1.9.0 | chromedp |
| golang.org/x/exp | v0.0.0-20230905 | sourcegraph/conc |
| golang.org/x/sys | v0.16.0 | multiple |
| golang.org/x/text | v0.13.0 | chromedp |
| gopkg.in/ini.v1 | v1.67.0 | viper |

---

## After Cleanup

### New go.mod

```go
module github.com/crab-meat-repos/cicerone

go 1.22

require (
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/gorilla/websocket v1.5.0
    github.com/mitchellh/go-homedir v1.1.0
    github.com/robfig/cron/v3 v3.0.1
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.17.0
    golang.org/x/crypto v0.14.0
    gopkg.in/yaml.v3 v3.0.1
)
```

### Dependency Count

| Category | Before | After |
|----------|--------|-------|
| Direct | 11 | 8 |
| Indirect | 28 | 20 |
| **Total** | **39** | **28** |

---

## Dependency Usage Details

### KEEP Dependencies

#### github.com/go-telegram-bot-api/telegram-bot-api/v5
- **Used in:** openclaw/telegram.go, main.go
- **Purpose:** Telegram Bot API client
- **Replaceable?** No - core feature

#### github.com/gorilla/websocket
- **Used in:** openclaw/api.go (WebSocket server), llm.go (Ollama streaming)
- **Purpose:** WebSocket client and server
- **Replaceable?** Possible with stdlib, but more work

#### github.com/mitchellh/go-homedir
- **Used in:** main.go, llm.go, chat.go, admin.go, image.go, library.go
- **Purpose:** Cross-platform home directory resolution
- **Replaceable?** Yes - can use os.UserHomeDir() (Go 1.12+)

#### github.com/robfig/cron/v3
- **Used in:** openclaw/scheduler.go
- **Purpose:** Cron job scheduling
- **Replaceable?** Possible, but cron is well-tested

#### github.com/spf13/cobra
- **Used in:** main.go, all command files
- **Purpose:** CLI framework
- **Replaceable?** No - core feature

#### github.com/spf13/viper
- **Used in:** main.go, openclaw_cmd.go
- **Purpose:** Configuration management
- **Replaceable?** Possible with simpler YAML unmarshal

#### golang.org/x/crypto
- **Used in:** crypto.go (argon2, scrypt), SSH in do command
- **Purpose:** Cryptographic functions, SSH
- **Replaceable?** Keep for SSH support

#### gopkg.in/yaml.v3
- **Used in:** openclaw/config.go, viper (indirect)
- **Purpose:** YAML parsing
- **Replaceable?** No - config format

---

### REMOVE Dependencies

#### github.com/bwmarrin/discordgo
- **Used in:** openclaw/discord.go (deleted)
- **Reason:** Discord provider removed

#### github.com/chromedp/chromedp
- **Used in:** openclaw/browser.go (deleted)
- **Reason:** Chrome automation removed

#### github.com/mattn/go-sqlite3
- **Used in:** openclaw/scheduler_store.go
- **Reason:** May keep for scheduler persistence, or replace with JSON file
- **Decision:** KEEP for now, evaluate in Phase 9

---

## Optional Optimizations

### Replace mitchellh/go-homedir with stdlib

```go
// Before
home, _ := homedir.Dir()

// After
home, _ := os.UserHomeDir()
```

**Impact:** Remove 1 dependency
**Effort:** Low (find/replace)

### Replace viper with simpler config

```go
// Current: viper with multiple formats
// Could use: direct yaml.Unmarshal

type Config struct {
    Telegram TelegramConfig `yaml:"telegram"`
    LLM      LLMConfig       `yaml:"llm"`
    Gateway  GatewayConfig   `yaml:"gateway"`
}

config := &Config{}
data, _ := os.ReadFile(configPath)
yaml.Unmarshal(data, config)
```

**Impact:** Remove viper + 15 indirect dependencies
**Effort:** Medium (rewrite config loading)

---

## Phase 1 Dependency Checklist

- [x] Inventory all dependencies
- [x] Map dependencies to features
- [x] Identify unused dependencies
- [x] Document removal plan
- [ ] Execute go mod tidy after cleanup

---

*Generated by Phase 1 analysis*