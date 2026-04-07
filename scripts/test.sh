#!/bin/bash
# /opt/cicerone/scripts/test.sh
# Cicerone Test Suite
set -e

echo "=== Cicerone Test Suite ==="
echo "Date: $(date)"
echo "Host: $(hostname)"
echo "OS: $(cat /etc/os-release | head -3)"
echo ""

# Build
echo "Building cicerone..."
cd /opt/cicerone
go build -o cicerone .

# Basic tests
echo ""
echo "=== Test 1: cicerone about ==="
./cicerone about

echo ""
echo "=== Test 2: cicerone --help ==="
./cicerone --help

echo ""
echo "=== Test 3: cicerone check ==="
./cicerone check || echo "OpenClaw not installed"

echo ""
echo "=== Test 4: cicerone node show ==="
./cicerone node show || echo "No node configuration"

echo ""
echo "=== Test 5: ollama list ==="
ollama list

echo ""
echo "=== Test 6: cicerone do 'what is 2+2?' ==="
./cicerone do "what is 2+2?" || echo "Ollama query failed"

echo ""
echo "=== Test 7: localhost commands ==="
./cicerone do "localhost:hostname" || echo "localhost:hostname failed"
./cicerone do "localhost:whoami" || echo "localhost:whoami failed"
./cicerone do "localhost:df -h /" || echo "localhost:df -h / failed"

echo ""
echo "=== All tests complete ==="
