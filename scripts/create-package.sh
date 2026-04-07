#!/bin/bash
# Create install package with cicerone and llama.cpp binaries
# Usage: ./create-package.sh [output-dir]

set -e

OUTPUT_DIR="${1:-/opt/packages}"
PACKAGE_NAME="cicerone-llama-bundle"
VERSION=$(date +%Y%m%d)
PACKAGE_DIR="$OUTPUT_DIR/$PACKAGE_NAME-$VERSION"

echo "=== Creating Install Package ==="
echo "Output: $PACKAGE_DIR"
echo ""

# Create package directory
mkdir -p "$PACKAGE_DIR"/{bin,etc,lib/systemd,share/doc}

# Copy binaries
echo "Copying binaries..."
if [ -f "/usr/local/bin/cicerone" ]; then
    cp /usr/local/bin/cicerone "$PACKAGE_DIR/bin/"
    echo "  - cicerone"
else
    echo "ERROR: cicerone binary not found"
    exit 1
fi

if [ -f "/opt/llama.cpp/build/bin/llama-server" ]; then
    cp /opt/llama.cpp/build/bin/llama-server "$PACKAGE_DIR/bin/"
    echo "  - llama-server"
elif [ -f "/usr/local/bin/llama-server" ]; then
    cp /usr/local/bin/llama-server "$PACKAGE_DIR/bin/"
    echo "  - llama-server"
else
    echo "WARNING: llama-server binary not found"
fi

# Copy shared libraries if CUDA
if [ -d "/opt/llama.cpp/build/bin" ]; then
    for lib in /opt/llama.cpp/build/bin/*.so 2>/dev/null; do
        if [ -f "$lib" ]; then
            cp "$lib" "$PACKAGE_DIR/lib/"
            echo "  - $(basename $lib)"
        fi
    done
fi

# Create systemd service files
echo "Creating systemd service..."
cat > "$PACKAGE_DIR/lib/systemd/llama-server.service" << 'EOF'
[Unit]
Description=llama.cpp Server
After=network.target

[Service]
Type=simple
User=cicerone
Group=cicerone
ExecStart=/usr/local/bin/llama-server --model /opt/models/gemma-2-2b.Q4_K_M.gguf --host 0.0.0.0 --port 8080 --ctx-size 4096 --n-gpu-layers 35
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create configuration templates
echo "Creating configuration templates..."
cat > "$PACKAGE_DIR/etc/cicerone.json" << 'EOF'
{
  "llm": {
    "provider": "openai",
    "apiUrl": "http://localhost:8080/v1",
    "model": "gemma-2-2b"
  }
}
EOF

cat > "$PACKAGE_DIR/etc/llama-config.sh" << 'EOF'
# llama.cpp configuration
LLAMA_MODEL="/opt/models/gemma-2-2b.Q4_K_M.gguf"
LLAMA_HOST="0.0.0.0"
LLAMA_PORT="8080"
LLAMA_CTX_SIZE="4096"
LLAMA_GPU_LAYERS="35"
EOF

# Create install script
echo "Creating install script..."
cat > "$PACKAGE_DIR/install.sh" << 'INSTALLSCRIPT'
#!/bin/bash
# Install Cicerone + llama.cpp bundle
set -e

echo "=== Installing Cicerone + llama.cpp ==="

# Check for root
if [ "$EUID" -ne 0 ]; then
    echo "Running with sudo..."
    sudo "$0" "$@"
    exit $?
fi

# Install binaries
echo "Installing binaries..."
cp bin/cicerone /usr/local/bin/
chmod +x /usr/local/bin/cicerone

if [ -f bin/llama-server ]; then
    cp bin/llama-server /usr/local/bin/
    chmod +x /usr/local/bin/llama-server
fi

# Create cicerone user
if ! id -u cicerone >/dev/null 2>&1; then
    echo "Creating cicerone user..."
    useradd -r -s /bin/bash cicerone
fi

# Create directories
mkdir -p /opt/models
mkdir -p /var/log/cicerone
mkdir -p /home/cicerone/.cicerone
chown -R cicerone:cicerone /var/log/cicerone
chown -R cicerone:cicerone /home/cicerone/.cicerone

# Install configuration
cp etc/cicerone.json /home/cicerone/.cicerone/
chown cicerone:cicerone /home/cicerone/.cicerone/cicerone.json

# Install systemd service
if [ -f lib/systemd/llama-server.service ]; then
    cp lib/systemd/llama-server.service /etc/systemd/system/
    systemctl daemon-reload
    echo "Systemd service installed (not enabled)"
fi

# Install docs
mkdir -p /usr/share/doc/cicerone
cp share/doc/* /usr/share/doc/cicerone/ 2>/dev/null || true

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Download a model to /opt/models/"
echo "     wget -O /opt/models/gemma-2-2b.Q4_K_M.gguf https://huggingface.co/..."
echo ""
echo "  2. Configure cicerone"
echo "     /usr/local/bin/cicerone llm show"
echo ""
echo "  3. Start llama.cpp server (optional)"
echo "     systemctl enable --now llama-server"
echo ""
INSTALLSCRIPT
chmod +x "$PACKAGE_DIR/install.sh"

# Create README
cat > "$PACKAGE_DIR/share/doc/README.md" << 'EOF'
# Cicerone + llama.cpp Bundle

This package contains pre-built binaries for Cicerone and llama.cpp.

## Contents

- `bin/cicerone` - Cicerone CLI binary
- `bin/llama-server` - llama.cpp HTTP server
- `etc/cicerone.json` - Default configuration
- `lib/systemd/llama-server.service` - Systemd service file

## Installation

```bash
sudo ./install.sh
```

## Quick Start

1. Download a model:
   ```bash
   wget -O /opt/models/gemma-2-2b.Q4_K_M.gguf \
     https://huggingface.co/bartowski/gemma-2-2b-it-GGUF/resolve/main/gemma-2-2b-it-Q4_K_M.gguf
   ```

2. Start llama.cpp server:
   ```bash
   llama-server --model /opt/models/gemma-2-2b.Q4_K_M.gguf --host 0.0.0.0 --port 8080 --ctx-size 4096
   ```

3. Test cicerone:
   ```bash
   cicerone llm show
   cicerone do "what is the hostname"
   ```

## Configuration

Edit `~/.cicerone/cicerone.json` to configure the LLM provider:

```json
{
  "llm": {
    "provider": "openai",
    "apiUrl": "http://localhost:8080/v1",
    "model": "gemma-2-2b"
  }
}
```

## Documentation

- [LLM Providers Guide](../docs/llm-providers.md)
- [README](../README.md)
EOF

# Create package manifest
echo "Creating manifest..."
cat > "$PACKAGE_DIR/MANIFEST" << EOF
Package: $PACKAGE_NAME
Version: $VERSION
Date: $(date -Iseconds)
Architecture: $(uname -m)
Kernel: $(uname -r)

Binaries:
  - bin/cicerone ($(du -h bin/cicerone | cut -f1))
  - bin/llama-server ($(du -h bin/llama-server 2>/dev/null | cut -f1 || echo "N/A"))

Files:
$(find "$PACKAGE_DIR" -type f | sed 's|'"$PACKAGE_DIR"'/||' | sort)
EOF

# Create tarball
echo "Creating tarball..."
cd "$OUTPUT_DIR"
tar -czf "$PACKAGE_NAME-$VERSION.tar.gz" "$PACKAGE_NAME-$VERSION"

echo ""
echo "=== Package Created ==="
echo "Directory: $PACKAGE_DIR"
echo "Tarball: $OUTPUT_DIR/$PACKAGE_NAME-$VERSION.tar.gz"
echo ""
echo "Manifest:"
cat "$PACKAGE_DIR/MANIFEST"

exit 0