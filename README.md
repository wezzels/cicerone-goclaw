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

# Or with version info
./scripts/build.sh v2.0.0

# Run doctor
./cicerone doctor

# Start Telegram bot
./cicerone telegram

# Interactive chat
./cicerone chat

# Security audit
./cicerone security

# Show version
./cicerone version
```

## Commands

| Command | Description |
|---------|-------------|
| `telegram` | Start Telegram bot |
| `tui` | Launch interactive TUI |
| `gateway restart` | Restart gateway |
| `gateway status` | Check gateway status |
| `doctor` | Run health diagnostics |
| `security` | Run security audit |
| `llm show` | Show LLM config |
| `llm test` | Test LLM connection |
| `llm models` | List available models |
| `do` | Execute via LLM |
| `chat` | Interactive LLM chat |
| `config show` | Show configuration |
| `config set <key> <value>` | Set configuration |
| `version` | Show version |

## Configuration

Edit `~/.cicerone/config.yaml`:

```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN"
  allowed_users:
    - 123456789

llm:
  provider: ollama
  base_url: "http://localhost:11434"
  model: "gemma3:12b"

gateway:
  listen: "127.0.0.1:8080"

logging:
  level: info
```

## Requirements

- Go 1.22+
- Ollama (for LLM) or llama.cpp server
- Telegram bot token (for Telegram)

## Installation

```bash
# Clone
git clone https://idm.wezzel.com/crab-meat-repos/cicerone.git
cd cicerone

# Build
go build -o cicerone .

# Install
sudo cp cicerone /usr/local/bin/
```

## Architecture

```
cicerone/
├── cmd/                 # Command implementations
│   ├── root.go         # Root command
│   ├── telegram.go     # Telegram bot command
│   ├── tui.go          # TUI command
│   ├── gateway.go      # Gateway management
│   ├── doctor.go       # Health checks
│   ├── security.go     # Security audit
│   ├── llm_cmd.go      # LLM commands
│   ├── do.go           # Execute via LLM
│   ├── chat_cmd.go     # Interactive chat
│   ├── config_cmd.go   # Configuration
│   └── version.go      # Version info
├── llm/                 # LLM provider implementations
│   ├── provider.go     # Provider interface
│   ├── ollama.go       # Ollama client
│   ├── llamacpp.go     # llama.cpp client
│   └── llm_test.go     # Tests
├── telegram/            # Telegram bot implementation
│   ├── bot.go          # Bot client
│   ├── llm_handler.go  # LLM message handler
│   ├── conversation.go # Conversation history
│   └── bot_test.go     # Tests
├── main.go              # Entry point
└── scripts/
    ├── build.sh        # Build script with version
    └── cleanup.sh       # Cleanup script
```

## License

MIT