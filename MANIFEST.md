# Cicerone Go-Only Manifest

**Generated:** 2026-04-06
**Branch:** go-only-refactor
**Total Files:** 39 Go files
**Total Lines:** 14,377

---

## File Classification

### DELETE (22 files, 7,080 lines)

| File | Lines | Reason |
|------|-------|--------|
| admin.go | 1057 | Admin VM management - not needed |
| admin_test.go | 385 | Tests for deleted code |
| client.go | 449 | API client for server - not needed |
| crypto.go | 312 | Vault encryption - not needed |
| crypto_test.go | 273 | Tests for deleted code |
| docker.go | 146 | Docker operations - not needed |
| image.go | 1099 | VM image management - not needed |
| image_test.go | 340 | Tests for deleted code |
| installer.go | 842 | System installer - not needed |
| installer_gui.go | 278 | GUI installer - not needed |
| installer_test.go | 380 | Tests for deleted code |
| library.go | 906 | RAG library - not needed |
| library_test.go | 303 | Tests for deleted code |
| runner_test.go | 209 | GitLab runner tests - not needed |
| server.go | 529 | API server - not needed |
| tasks.go | 336 | Task management - not needed |
| vault.go | 438 | Secret vault - not needed |
| openclaw/discord.go | 277 | Discord provider - not needed |
| openclaw/signal.go | 93 | Signal provider - not needed |
| openclaw/whatsapp.go | 94 | WhatsApp provider - not needed |
| openclaw/browser.go | 282 | Chrome automation - not needed |

### KEEP (17 files, 7,297 lines)

| File | Lines | Action |
|------|-------|--------|
| main.go | 3363 | Refactor - remove 70% of commands |
| llm.go | 544 | Keep - simplify to ollama+llamacpp |
| llm_test.go | 547 | Keep - update tests |
| chat.go | 744 | Keep - llama.cpp chat |
| main_test.go | 181 | Keep - update tests |
| openclaw_cmd.go | 141 | Keep - gateway command |
| installer_tui.go | 575 | Rename to tui.go, simplify |

#### openclaw/ package (KEEP)

| File | Lines | Action |
|------|-------|--------|
| telegram.go | 331 | Keep - primary provider |
| api.go | 585 | Keep - WebSocket API |
| engine.go | 348 | Keep - core engine |
| config.go | 113 | Keep - config loading |
| messaging.go | 195 | Keep - message handling |
| providers.go | 105 | Keep - provider interface |
| plugin.go | 229 | Keep - plugin system |
| plugin_tts.go | 257 | Keep - TTS plugin |
| plugin_websearch.go | 207 | Keep - web search |
| scheduler.go | 370 | Keep - cron jobs |
| scheduler_store.go | 260 | Keep - scheduler persistence |
| nodes.go | 325 | Keep - node pairing |
| tts.go | 76 | Keep - TTS integration |
| cmd/openclaw/main.go | 93 | Keep - entry point |

---

## Dependency Mapping

### Dependencies to REMOVE

| Dependency | Used In | Reason |
|------------|---------|--------|
| github.com/bwmarrin/discordgo | openclaw/discord.go | Discord deleted |
| github.com/chromedp/chromedp | openclaw/browser.go | Browser deleted |
| github.com/chromedp/cdproto | openclaw/browser.go | Browser deleted |
| github.com/chromedp/sysutil | openclaw/browser.go | Browser deleted |
| github.com/mattn/go-sqlite3 | openclaw/scheduler_store.go | May keep for scheduler |

### Dependencies to KEEP

| Dependency | Version | Used In |
|------------|---------|---------|
| github.com/go-telegram-bot-api/telegram-bot-api/v5 | v5.5.1 | openclaw/telegram.go, main.go |
| github.com/gorilla/websocket | v1.5.0 | openclaw/api.go, llm.go |
| github.com/spf13/cobra | v1.8.0 | All commands |
| github.com/spf13/viper | v1.17.0 | Config management |
| github.com/mitchellh/go-homedir | v1.1.0 | Config paths |
| github.com/robfig/cron/v3 | v3.0.1 | openclaw/scheduler.go |
| golang.org/x/crypto | v0.14.0 | SSH in do command |
| gopkg.in/yaml.v3 | v3.0.1 | Config files |

---

## Import Analysis by File

### DELETE Files - Imports

```
admin.go:
  - github.com/mitchellh/go-homedir
  - github.com/spf13/cobra

crypto.go:
  - golang.org/x/crypto/argon2
  - golang.org/x/crypto/scrypt

docker.go:
  - (stdlib only)

image.go:
  - github.com/mitchellh/go-homedir
  - github.com/spf13/cobra

library.go:
  - github.com/mitchellh/go-homedir

server.go:
  - (stdlib only)

vault.go:
  - (stdlib only)

tasks.go:
  - (stdlib only)

openclaw/discord.go:
  - github.com/bwmarrin/discordgo

openclaw/browser.go:
  - github.com/chromedp/chromedp
```

### KEEP Files - Imports

```
main.go:
  - github.com/mitchellh/go-homedir
  - github.com/spf13/cobra
  - github.com/spf13/viper

llm.go:
  - github.com/mitchellh/go-homedir

chat.go:
  - github.com/mitchellh/go-homedir
  - github.com/spf13/cobra

openclaw_cmd.go:
  - github.com/crab-meat-repos/cicerone/openclaw
  - github.com/spf13/cobra
  - github.com/spf13/viper

installer_tui.go:
  - github.com/spf13/cobra

openclaw/telegram.go:
  - github.com/go-telegram-bot-api/telegram-bot-api/v5

openclaw/api.go:
  - github.com/gorilla/websocket

openclaw/scheduler.go:
  - github.com/robfig/cron/v3

openclaw/config.go:
  - gopkg.in/yaml.v3
```

---

## Commands in main.go to REMOVE

| Command | Lines | Reason |
|---------|-------|--------|
| aboutCmd | ~30 | Trivial, can be version |
| checkCmd | ~20 | GitLab runner specific |
| tellCmd | ~20 | GitLab runner specific |
| configCmd | ~60 | GitLab runner specific |
| configNewCmd | ~50 | GitLab runner specific |
| nodeCmd | ~40 | GitLab runner specific |
| nodeShowCmd | ~15 | GitLab runner specific |
| nodeTestCmd | ~20 | GitLab runner specific |
| runnerCmd | ~150 | GitLab runner management |
| runnerNewCmd | ~30 | GitLab runner management |
| runnerConfigCmd | ~30 | GitLab runner management |
| runnerDeployCmd | ~30 | GitLab runner management |
| runnerCancelCmd | ~30 | GitLab runner management |
| runnerHelpCmd | ~30 | GitLab runner management |
| pipelineCmd | ~50 | GitLab pipeline tests |
| pipelineRunCmd | ~20 | GitLab pipeline tests |
| pipelineResultsCmd | ~20 | GitLab pipeline tests |
| pipelineCleanCmd | ~20 | GitLab pipeline tests |
| serveCmd | ~40 | API server |
| adminCmd | ~80 | Admin VM management |
| adminNewCmd | ~60 | Admin VM management |
| adminShowCmd | ~30 | Admin VM management |
| adminListCmd | ~30 | Admin VM management |
| adminRemoveCmd | ~30 | Admin VM management |
| libraryCmd | ~50 | RAG library |
| libraryNewCmd | ~30 | RAG library |
| libraryShowCmd | ~20 | RAG library |
| libraryRefreshCmd | ~30 | RAG library |
| libraryRemoveCmd | ~20 | RAG library |
| libraryUpdateCmd | ~20 | RAG library |
| libraryQueryCmd | ~30 | RAG library |
| imageCmd | ~50 | VM image management |
| installerCmd | ~50 | System installer |

**Total commands to remove:** ~35 commands

---

## Commands to KEEP

| Command | Reason |
|---------|--------|
| rootCmd | Root command |
| openclawCmd | Gateway start |
| openclawVersionCmd | Version info |
| llmCmd | LLM management |
| llmShowCmd | Show LLM config |
| doCmd | Execute via LLM |
| chatCmd | Interactive chat |

---

## New Commands to ADD

| Command | Description |
|---------|-------------|
| telegramCmd | Start Telegram bot (rename openclaw) |
| tuiCmd | Interactive TUI (simplified) |
| gatewayCmd | Gateway management |
| gatewayRestartCmd | Restart gateway |
| gatewayStatusCmd | Check gateway status |
| doctorCmd | Health diagnostics |
| securityCmd | Security audit |
| llmTestCmd | Test LLM connection |
| llmModelsCmd | List available models |
| configShowCmd | Show config |
| configSetCmd | Set config value |

---

## Estimated Refactoring

| Category | Lines Removed | Lines Added |
|----------|---------------|-------------|
| main.go commands | ~2,400 | ~300 |
| Deleted files | ~7,000 | 0 |
| New commands | 0 | ~400 |
| LLM simplification | ~100 | 0 |
| Telegram refactor | ~200 | ~100 |
| **Total** | **~9,700** | **~800** |

**Final codebase estimate:** ~5,500 lines (from 14,377)

---

## Cleanup Script

```bash
#!/bin/bash
# scripts/cleanup.sh - Run after Phase 2

set -e

echo "Removing unused files..."

# Admin VM management
rm -f admin.go admin_test.go

# Client and server
rm -f client.go server.go

# Crypto and vault
rm -f crypto.go crypto_test.go vault.go

# Docker
rm -f docker.go

# VM images
rm -f image.go image_test.go

# Installer
rm -f installer.go installer_gui.go installer_test.go

# RAG library
rm -f library.go library_test.go

# GitLab runner
rm -f runner_test.go

# Tasks
rm -f tasks.go

# Unused providers
rm -f openclaw/discord.go
rm -f openclaw/signal.go
rm -f openclaw/whatsapp.go
rm -f openclaw/browser.go

echo "Cleanup complete!"
echo "Files removed: 22"
echo "Lines removed: ~7,000"
```

---

## Phase 1 Checklist

- [x] Inventory all Go files and line counts
- [x] Map dependency graph
- [x] Create deletion checklist
- [x] Analyze imports per file
- [x] List commands to remove/keep/add
- [ ] Commit MANIFEST.md

---

*Generated by Phase 1 analysis*