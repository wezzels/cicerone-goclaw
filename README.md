# Cicerone

Go-only messaging gateway with LLM integration, code execution, and SSH capabilities.

## Features

- **Telegram Bot** - Full bot API support
- **LLM Chat** - Ollama and llama.cpp integration
- **Doctor** - Health diagnostics
- **Security** - Security audit
- **TUI** - Interactive terminal interface
- **Workspace** - Code execution environment
- **SSH Client** - Remote host management
- **Test Runner** - Local and remote testing

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

### Core Commands

| Command | Description |
|---------|-------------|
| `telegram` | Start Telegram bot |
| `tui` | Launch interactive TUI |
| `doctor` | Run health diagnostics |
| `security` | Run security audit |
| `version` | Show version |

### LLM Commands

| Command | Description |
|---------|-------------|
| `chat` | Interactive LLM chat |
| `llm show` | Show LLM config |
| `llm test` | Test LLM connection |
| `llm models` | List available models |
| `do` | Execute via LLM |

### Workspace Commands

| Command | Description |
|---------|-------------|
| `workspace init [path]` | Initialize workspace |
| `workspace status` | Show workspace info |
| `workspace list` | List workspaces |
| `workspace clean` | Clean workspace |
| `exec <command>` | Execute command in workspace |

### SSH Commands

| Command | Description |
|---------|-------------|
| `ssh add <name> <host> <user>` | Add SSH host |
| `ssh list` | List SSH hosts |
| `ssh test <name>` | Test SSH connection |
| `ssh exec <name> <command>` | Execute remote command |
| `ssh shell <name>` | Start interactive shell |
| `ssh push <name> <local> <remote>` | Push file to remote |
| `ssh pull <name> <remote> <local>` | Pull file from remote |
| `ssh remove <name>` | Remove SSH host |

### Test Commands

| Command | Description |
|---------|-------------|
| `test [path]` | Run tests locally |
| `test --cover` | Run with coverage |
| `test --remote <host>` | Run on remote host |
| `test -run <pattern>` | Run matching tests |

### Configuration Commands

| Command | Description |
|---------|-------------|
| `config show` | Show configuration |
| `config set <key> <value>` | Set configuration |
| `config wizard` | Interactive setup |

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

## SSH Configuration

SSH hosts are stored in `~/.cicerone/ssh_hosts.yaml`:

```yaml
darth:
  name: darth
  host: 10.0.0.117
  port: 22
  user: wez
  key_path: ~/.ssh/id_rsa

miner:
  name: miner
  host: 207.244.226.151
  port: 22
  user: wez
  key_path: ~/.ssh/id_ed25519
```

## Requirements

- Go 1.22+
- Ollama (for LLM) or llama.cpp server
- Telegram bot token (for Telegram)

## Installation

```bash
# Clone
git clone https://github.com/wezzels/cicerone-goclaw.git
cd cicerone-goclaw

# Build
go build -o cicerone .

# Install
sudo cp cicerone /usr/local/bin/

# Or use Makefile
make build
make install
```

## Architecture

```
cicerone-goclaw/
├── cmd/                    # Command implementations
│   ├── root.go            # Root command
│   ├── telegram.go        # Telegram bot command
│   ├── tui.go             # TUI command
│   ├── gateway.go         # Gateway management
│   ├── doctor.go          # Health checks
│   ├── security.go        # Security audit
│   ├── llm_cmd.go         # LLM commands
│   ├── do.go              # Execute via LLM
│   ├── chat_cmd.go        # Interactive chat
│   ├── config_cmd.go      # Configuration
│   ├── wizard.go          # Config wizard
│   ├── workspace.go       # Workspace management
│   ├── exec.go            # Command execution
│   ├── ssh.go             # SSH commands
│   └── test.go            # Test runner
├── internal/               # Internal packages
│   ├── workspace/         # Workspace management
│   │   ├── workspace.go   # Workspace struct
│   │   ├── executor.go    # Command executor
│   │   ├── sandbox.go     # Sandbox isolation
│   │   └── workspace_test.go
│   └── ssh/                # SSH client
│       ├── client.go      # SSH client wrapper
│       ├── config.go      # SSH configuration
│       ├── tunnel.go      # Tunnel management
│       ├── transfer.go    # SFTP file transfer
│       └── ssh_test.go
├── llm/                    # LLM provider implementations
│   ├── provider.go        # Provider interface
│   ├── ollama.go          # Ollama client
│   ├── llamacpp.go        # llama.cpp client
│   └── llm_test.go        # Tests
├── telegram/               # Telegram bot implementation
│   ├── bot.go             # Bot client
│   ├── llm_handler.go     # LLM message handler
│   ├── conversation.go    # Conversation history
│   └── bot_test.go        # Tests
├── main.go                 # Entry point
├── Makefile                # Build automation
└── docs/                   # Documentation
    ├── INSTALLATION.md
    ├── TEST_RESULTS.md
    └── CONFIG_WIZARD_ROADMAP.md
```

## Examples

### Workspace Management

```bash
# Initialize workspace
cicerone workspace init ./myproject

# Check status
cicerone workspace status

# Execute command
cicerone exec --workdir ./myproject "go build ./..."

# Run tests
cicerone test ./myproject/...
```

### SSH Operations

```bash
# Add SSH host
cicerone ssh add darth 10.0.0.117 wez --key ~/.ssh/id_rsa

# Test connection
cicerone ssh test darth

# Run command remotely
cicerone ssh exec darth "ls -la"

# Interactive shell
cicerone ssh shell darth

# Transfer files
cicerone ssh push darth ./local.txt /home/wez/remote.txt
cicerone ssh pull darth /home/wez/remote.txt ./local.txt
```

### Remote Testing

```bash
# Test on remote host
cicerone test --remote darth ./...

# With coverage
cicerone test --remote darth --cover
```

## Development

```bash
# Run tests
make test

# Run with coverage
make coverage

# Build for all platforms
make build-all

# Create release
make release

# Lint code
make lint
```

## License

MIT