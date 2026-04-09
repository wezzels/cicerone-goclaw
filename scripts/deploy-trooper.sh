#!/bin/bash
# Deploy cicerone-goclaw to replace Python cicerone on trooper VMs
#
# Usage: ./deploy-trooper.sh [host]
# Example: ./deploy-trooper.sh 192.168.122.248
#
# This script:
# 1. Builds cicerone for the target architecture
# 2. Copies binary and systemd unit to remote host
# 3. Installs and starts the service

set -e

HOST="${1:-localhost}"
ARCH="${2:-amd64}"
BINARY="cicerone-linux-${ARCH}"
REMOTE_USER="${3:-user}"
REMOTE_DIR="/opt/cicerone"

echo "=== Deploying cicerone-goclaw to $HOST ==="
echo "Architecture: $ARCH"
echo "Remote user: $REMOTE_USER"
echo "Remote dir: $REMOTE_DIR"

# Build if needed
if [ ! -f "$BINARY" ]; then
    echo "Building for $ARCH..."
    GOOS=linux GOARCH=$ARCH go build -o "$BINARY" .
fi

# Create remote directory
echo "Creating remote directory..."
ssh "$REMOTE_USER@$HOST" "sudo mkdir -p $REMOTE_DIR && sudo chown $REMOTE_USER:$REMOTE_USER $REMOTE_DIR"

# Copy binary
echo "Copying binary..."
scp "$BINARY" "$REMOTE_USER@$HOST:$REMOTE_DIR/cicerone"
ssh "$REMOTE_USER@$HOST" "chmod +x $REMOTE_DIR/cicerone"

# Copy systemd unit
echo "Installing systemd service..."
cat scripts/cicerone.service | ssh "$REMOTE_USER@$HOST" "sudo tee /etc/systemd/system/cicerone.service > /dev/null"

# Create config directory
ssh "$REMOTE_USER@$HOST" "mkdir -p ~/.cicerone"

# Create default config if not exists
ssh "$REMOTE_USER@$HOST" "cat > ~/.cicerone/config.yaml << 'EOF'
llm:
  provider: ollama
  base_url: http://127.0.0.1:11434
  model: llama3.1:8b
  timeout: 300
EOF"

# Reload and enable
echo "Enabling service..."
ssh "$REMOTE_USER@$HOST" "sudo systemctl daemon-reload && sudo systemctl enable cicerone"

# Check if ollama is running
echo "Checking ollama..."
if ! ssh "$REMOTE_USER@$HOST" "systemctl is-active ollama" 2>/dev/null; then
    echo "WARNING: ollama service not active. Install with:"
    echo "  curl -fsSL https://ollama.com/install.sh | sh"
    echo "  ollama pull llama3.1:8b"
fi

echo ""
echo "=== Deployment complete ==="
echo ""
echo "To start the service:"
echo "  ssh $REMOTE_USER@$HOST 'sudo systemctl start cicerone'"
echo ""
echo "To check status:"
echo "  ssh $REMOTE_USER@$HOST 'sudo systemctl status cicerone'"
echo ""
echo "To test:"
echo "  curl http://$HOST:18789/health"
echo "  curl http://$HOST:18789/v1/models"