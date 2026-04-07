# Installation Guide

## Requirements

- Go 1.22+ (for building from source)
- Ollama (for LLM) or llama.cpp server
- Telegram bot token (for Telegram bot)

## Install from Binary

### Download

Download the latest release from [GitHub Releases](https://github.com/wezzels/cicerone-goclaw/releases):

```bash
# Linux AMD64
curl -LO https://github.com/wezzels/cicerone-goclaw/releases/download/v2.0.0/cicerone-linux-amd64
chmod +x cicerone-linux-amd64
sudo mv cicerone-linux-amd64 /usr/local/bin/cicerone

# Linux ARM64 (Raspberry Pi, etc.)
curl -LO https://github.com/wezzels/cicerone-goclaw/releases/download/v2.0.0/cicerone-linux-arm64
chmod +x cicerone-linux-arm64
sudo mv cicerone-linux-arm64 /usr/local/bin/cicerone
```

### Verify

```bash
cicerone version
```

## Install from Source

### Prerequisites

```bash
# Install Go (Ubuntu/Debian)
sudo apt update
sudo apt install -y golang-go

# Install Go (macOS)
brew install go

# Install Ollama (for LLM)
curl -fsSL https://ollama.com/install.sh | sh
ollama pull gemma3:12b
```

### Build

```bash
# Clone
git clone https://github.com/wezzels/cicerone-goclaw.git
cd cicerone-goclaw

# Build
go build -o cicerone .

# Or with version info
./scripts/build.sh v2.0.0

# Install
sudo cp cicerone /usr/local/bin/
```

## Configuration

### Create Config Directory

```bash
mkdir -p ~/.cicerone
```

### Create Config File

Create `~/.cicerone/config.yaml`:

```yaml
# Telegram configuration
telegram:
  # Get your bot token from @BotFather on Telegram
  bot_token: "YOUR_BOT_TOKEN"
  # List of allowed user IDs (empty = allow all)
  allowed_users: []
    # - 123456789

# LLM configuration
llm:
  # Provider: ollama or llamacpp
  provider: ollama
  # API URL
  base_url: "http://localhost:11434"
  # Model name
  model: "gemma3:12b"
  # Request timeout (seconds)
  timeout: 60

# Gateway configuration (for future use)
gateway:
  listen: "127.0.0.1:8080"

# Logging
logging:
  level: info
```

### Get Telegram Bot Token

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` command
3. Follow the prompts to create your bot
4. Copy the token to your config

### Get Your User ID

1. Open Telegram and search for `@userinfobot`
2. Start a conversation
3. It will reply with your user ID
4. Add to `allowed_users` in config (optional)

## Run

### Health Check

```bash
cicerone doctor
```

Expected output:
```
🏥 Cicerone Health Check
========================

  ✓ Config:              /home/user/.cicerone/config.yaml
  ⚠ Telegram Token:      configured (46 chars)
  ✓ LLM Connection:       http://localhost:11434
  ✓ Ollama Status:        running (PID 12345)
  ✓ Model Available:      gemma3:12b (configured)
  ✓ Network:              can reach Telegram API
  ✓ Disk Space:           100G available
  ✓ Memory:               16G available

Results: 7 passed, 1 warnings, 0 failed
```

### Start Telegram Bot

```bash
cicerone telegram
```

### Interactive TUI

```bash
cicerone tui
```

### LLM Chat

```bash
cicerone chat
```

## Systemd Service

Create `/etc/systemd/system/cicerone.service`:

```ini
[Unit]
Description=Cicerone Telegram Bot
After=network.target ollama.service

[Service]
Type=simple
User=cicerone
Group=cicerone
WorkingDirectory=/opt/cicerone
ExecStart=/usr/local/bin/cicerone telegram
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable cicerone
sudo systemctl start cicerone
sudo systemctl status cicerone
```

## Docker

### Build Image

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o cicerone .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/cicerone .
COPY config.yaml /root/.cicerone/config.yaml
CMD ["./cicerone", "telegram"]
```

### Run Container

```bash
# Build
docker build -t cicerone:latest .

# Run
docker run -d \
  --name cicerone \
  --restart unless-stopped \
  -v ~/.cicerone:/root/.cicerone \
  cicerone:latest
```

## Troubleshooting

### Config not found

```bash
mkdir -p ~/.cicerone
cicerone config show
```

### Ollama not running

```bash
ollama serve
```

### Model not available

```bash
ollama pull gemma3:12b
```

### Telegram token invalid

- Check token in config file
- Ensure no extra spaces or quotes
- Regenerate token from @BotFather if needed

### Permission denied

```bash
chmod +x cicerone
```

## Upgrade

```bash
# Stop service
sudo systemctl stop cicerone

# Download new version
curl -LO https://github.com/wezzels/cicerone-goclaw/releases/download/vNEW_VERSION/cicerone-linux-amd64
chmod +x cicerone-linux-amd64
sudo mv cicerone-linux-amd64 /usr/local/bin/cicerone

# Start service
sudo systemctl start cicerone
```