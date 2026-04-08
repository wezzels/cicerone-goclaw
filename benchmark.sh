#!/bin/bash
# Benchmark script for cicerone-goclaw

echo "========================================"
echo "CICERONE-GOCLAW BENCHMARK"
echo "========================================"
echo ""
echo "Machine: $(hostname)"
echo "CPU: $(cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2 | xargs)"
echo "RAM: $(free -h | grep Mem | awk '{print $2}')"
echo "OS: $(uname -s) $(uname -r)"
echo "Go: $(go version)"
echo ""
echo "========================================"
echo "TEST 1: Go Build (Clean)"
echo "========================================"

cd ~/cicerone-goclaw
rm -f cicerone
sync
echo "Building..."
time go build -o cicerone . 2>&1
echo ""

echo "========================================"
echo "TEST 2: Go Build (Cached)"
echo "========================================"
rm -f cicerone
sync
echo "Building (cached)..."
time go build -o cicerone . 2>&1
echo ""

echo "========================================"
echo "TEST 3: Go Test"
echo "========================================"
echo "Running tests..."
time go test ./... -short 2>&1 | tail -15
echo ""

echo "========================================"
echo "TEST 4: Binary Size"
echo "========================================"
ls -lh cicerone
echo ""

echo "========================================"
echo "BENCHMARK COMPLETE"
echo "========================================"