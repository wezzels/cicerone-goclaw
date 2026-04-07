# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] - 2026-04-07

### Breaking Changes

- **Removed**: Discord, Signal, WhatsApp, Browser support
- **Removed**: GitLab runner management
- **Removed**: RAG library, VM images, Admin VMs
- **Removed**: API server (`serve` command)
- **Removed**: Vault encryption, Tasks, Docker support

### Added

- `telegram` command - Start Telegram bot
- `tui` command - Interactive terminal UI
- `gateway restart/status` - Gateway management
- `doctor` command - Health diagnostics (8 checks)
- `security` command - Security audit (7 checks)
- `llm show/test/models` - LLM management
- `do` command - Execute instructions via LLM
- `chat` command - Interactive LLM chat
- `config show/set` - Configuration management

### Changed

- **Architecture**: Simplified from multi-provider to Telegram-only
- **LLM**: Removed OpenAI support, Ollama + llama.cpp only
- **Package structure**: 
  - New `cmd/` package for all commands
  - New `llm/` package for LLM providers
  - New `telegram/` package for Telegram bot
- **Binary size**: Reduced from ~17MB to ~13MB
- **Dependencies**: Reduced from 39 to ~28

### Removed

- `admin` command and related code
- `serve` command and API server
- `library` command and RAG support
- `image` command and VM image management
- `runner` command and GitLab runner support
- `pipeline` command and CI/CD tests
- `node` command and SSH node management
- `check` command (replaced by `doctor`)
- `tell` command (replaced by `do`)
- `installer` commands (GUI and TUI)
- `openclaw` package (replaced by `telegram/`)

### Technical Changes

- **Lines removed**: ~18,800
- **Lines added**: ~750
- **Files deleted**: 42
- **Files created**: 15
- **Test coverage**: 18 tests (llm: 11, telegram: 7)

## [1.5.0] - 2026-04-03

### Added

- Initial OpenClaw gateway support
- Multi-provider messaging (Telegram, Discord, Signal, WhatsApp)
- Browser automation support
- Plugin system (TTS, web search)
- Scheduler for cron jobs
- Node pairing for distributed deployments

## [1.0.0] - 2026-01-15

### Added

- Initial release
- GitLab runner management
- VM image management
- System installer
- LLM support (Ollama, OpenAI)