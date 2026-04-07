#!/bin/bash
# Install script for Cicerone + llama.cpp bundle
# This script is included in the package tarball

set -e

INSTALL_DIR="${1:-/usr/local}"
MODEL_DIR="${2:-/opt/models}"
DEFAULT_MODEL="gemma-2-2b-it.Q4_K_M.gguf"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         Cicerone + llama.cpp Installation                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check for root
if [ "$EUID" -ne 0 ]; then
    echo "This script requires root privileges."
    echo "Run with: sudo $0"
    exit 1
fi

# Detect system
echo "Detecting system..."
DISTRO=$(cat /etc/os-release | grep "^ID=" | cut -d= -f2 | tr -d '"')
ARCH=$(uname -m)
echo "  Distribution: $DISTRO"
echo "  Architecture: $ARCH"
echo ""

# Install dependencies
echo "Installing dependencies..."
case "$DISTRO" in
    ubuntu|debian)
        apt-get update
        apt-get install -y curl wget git
        ;;
    centos|rhel|fedora)
        yum install -y curl wget git
        ;;
    *)
        echo "Warning: Unsupported distribution. Installing minimal dependencies..."
        ;;
esac

# Create directories
echo "Creating directories..."
mkdir -p "$INSTALL_DIR/bin"
mkdir -p "$MODEL_DIR"
mkdir -p /var/log/cicerone
mkdir -p /var/lib/cicerone

# Create cicerone user
if ! id -u cicerone >/dev/null 2>&1; then
    echo "Creating cicerone user..."
    useradd -r -s /bin/bash -d /var/lib/cicerone cicerone
fi

chown -R cicerone:cicerone /var/log/cicerone
chown -R cicerone:cicerone /var/lib/cicerone

# Install binaries
echo "Installing binaries..."
if [ -f "$(dirname "$0")/bin/cicerone" ]; then
    # Running from package directory
    cp "$(dirname "$0")/bin/cicerone" "$INSTALL_DIR/bin/"
    cp "$(dirname "$0")/bin/llama-server" "$INSTALL_DIR/bin/" 2>/dev/null || true
elif [ -f "./bin/cicerone" ]; then
    # Running from extracted package
    cp ./bin/cicerone "$INSTALL_DIR/bin/"
    cp ./bin/llama-server "$INSTALL_DIR/bin/" 2>/dev/null || true
else
    echo "ERROR: Binaries not found"
    exit 1
fi

chmod +x "$INSTALL_DIR/bin/cicerone"
chmod +x "$INSTALL_DIR/bin/llama-server" 2>/dev/null || true

# Install configuration
echo "Installing configuration..."
mkdir -p /var/lib/cicerone/.cicerone

if [ -f "$(dirname "$0")/etc/cicerone.json" ]; then
    cp "$(dirname "$0")/etc/cicerone.json" /var/lib/cicerone/.cicerone/
elif [ -f "./etc/cicerone.json" ]; then
    cp ./etc/cicerone.json /var/lib/cicerone/.cicerone/
else
    # Create default config
    cat > /var/lib/cicerone/.cicerone/cicerone.json << 'EOF'
{
  "llm": {
    "provider": "openai",
    "apiUrl": "http://localhost:8080/v1",
    "model": "gemma-2-2b"
  }
}
EOF
fi

chown -R cicerone:cicerone /var/lib/cicerone/.cicerone

# Install systemd service
echo "Installing systemd service..."
cat > /etc/systemd/system/llama-server.service << 'EOF'
[Unit]
Description=llama.cpp Server
After=network.target

[Service]
Type=simple
User=cicerone
Group=cicerone
ExecStart=/usr/local/bin/llama-server --model /opt/models/gemma-2-2b-it.Q4_K_M.gguf --host 0.0.0.0 --port 8080 --ctx-size 4096 --n-gpu-layers 35
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Installation Complete                         ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Installed:"
echo "  - $INSTALL_DIR/bin/cicerone"
echo "  - $INSTALL_DIR/bin/llama-server"
echo ""
echo "Configuration:"
echo "  - /var/lib/cicerone/.cicerone/cicerone.json"
echo ""
echo "Next Steps:"
echo ""
echo "1. Download a model to $MODEL_DIR:"
echo "   wget -O $MODEL_DIR/gemma-2-2b-it.Q4_K_M.gguf \\"
echo "     https://huggingface.co/bartowski/gemma-2-2b-it-GGUF/resolve/main/gemma-2-2b-it-Q4_K_M.gguf"
echo ""
echo "2. Start llama.cpp server:"
echo "   /usr/local/bin/llama-server --model $MODEL_DIR/gemma-2-2b-it.Q4_K_M.gguf --port 8080 &"
echo ""
echo "   Or with systemd:"
echo "   systemctl enable --now llama-server"
echo ""
echo "3. Test cicerone:"
echo "   cicerone llm show"
echo "   cicerone do \"what is the hostname\""
echo ""
echo "4. Run verification tests:"
echo "   cicerone about"
echo "   cicerone check"
echo "   cicerone node show"
echo ""

exit 0