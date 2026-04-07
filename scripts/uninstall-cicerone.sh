#!/bin/bash
# uninstall-cicerone.sh - Remove Cicerone server from the system
#
# Usage:
#   sudo ./uninstall-cicerone.sh [--purge]
#
# Options:
#   --purge    Also remove vault data and logs

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Check root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}ERROR${NC} Please run as root (use sudo)"
    exit 1
fi

# Parse args
PURGE=false
if [ "$1" = "--purge" ]; then
    PURGE=true
fi

echo ""
log_warn "This will remove Cicerone from your system."
if [ "$PURGE" = true ]; then
    log_warn "Including all vault data and logs!"
fi
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Stop service
log_info "Stopping cicerone service..."
systemctl stop cicerone 2>/dev/null || true
systemctl disable cicerone 2>/dev/null || true

# Remove service file
log_info "Removing systemd service..."
rm -f /etc/systemd/system/cicerone.service
systemctl daemon-reload

# Remove binary
log_info "Removing binary..."
rm -f /usr/local/bin/cicerone

# Remove user (optional)
log_info "Removing cicerone user..."
userdel cicerone 2>/dev/null || true

# Purge data if requested
if [ "$PURGE" = true ]; then
    log_info "Purging data..."
    rm -rf /var/lib/cicerone
    rm -rf /var/log/cicerone
    rm -rf /etc/cicerone
else
    log_info "Keeping data in /var/lib/cicerone and /var/log/cicerone"
    log_info "Keeping config in /etc/cicerone"
fi

echo ""
log_info "Cicerone uninstalled successfully."
if [ "$PURGE" = false ]; then
    echo ""
    echo "To fully remove all data, run:"
    echo "  sudo rm -rf /var/lib/cicerone /var/log/cicerone /etc/cicerone"
fi
echo ""