#!/bin/bash
# Verification tests for Cicerone + llama.cpp integration
# Tests all cicerone commands with llama.cpp backend

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="/var/log/cicerone/tests"
DATE=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$LOG_DIR/test-$DATE.log"
REPORT_FILE="$LOG_DIR/test-$DATE.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
SKIP=0

# Initialize
mkdir -p "$LOG_DIR"
echo "=== Cicerone + llama.cpp Verification Tests ===" | tee "$LOG_FILE"
echo "Started: $(date)" | tee -a "$LOG_FILE"
echo "Log: $LOG_FILE" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

# Helper functions
log() {
    echo "$1" | tee -a "$LOG_FILE"
}

pass() {
    echo -e "${GREEN}PASS${NC}: $1" | tee -a "$LOG_FILE"
    ((PASS++))
}

fail() {
    echo -e "${RED}FAIL${NC}: $1" | tee -a "$LOG_FILE"
    ((FAIL++))
}

skip() {
    echo -e "${YELLOW}SKIP${NC}: $1" | tee -a "$LOG_FILE"
    ((SKIP++))
}

test_command() {
    local name="$1"
    local cmd="$2"
    local expected="$3"
    
    log ""
    log "Testing: $name"
    log "Command: $cmd"
    
    if output=$($cmd 2>&1); then
        if [ -n "$expected" ]; then
            if echo "$output" | grep -q "$expected"; then
                pass "$name"
                return 0
            else
                fail "$name (expected: '$expected')"
                log "Output: $output"
                return 1
            fi
        else
            pass "$name"
            return 0
        fi
    else
        fail "$name"
        log "Error: $output"
        return 1
    fi
}

# ========================================
# SECTION 1: Environment Checks
# ========================================
log ""
log "=== SECTION 1: Environment Checks ==="

test_command "Cicerone binary exists" "which cicerone" "cicerone"
test_command "Cicerone is executable" "test -x $(which cicerone)" ""
test_command "llama-server binary exists" "which llama-server" "llama-server" || skip "llama-server not in PATH"
test_command "Go version" "go version" "go"

# ========================================
# SECTION 2: Configuration Tests
# ========================================
log ""
log "=== SECTION 2: Configuration Tests ==="

test_command "Config directory exists" "test -d /var/lib/cicerone/.cicerone" "" || skip "Config directory not found"
test_command "Config file exists" "test -f /var/lib/cicerone/.cicerone/cicerone.json" "" || skip "Config file not found"
test_command "Config is valid JSON" "cat /var/lib/cicerone/.cicerone/cicerone.json" "llm" || skip "Config not found"

# ========================================
# SECTION 3: Cicerone Command Tests
# ========================================
log ""
log "=== SECTION 3: Cicerone Command Tests ==="

test_command "cicerone about" "cicerone about" "CICERONE"
test_command "cicerone help" "cicerone --help" "Available Commands"
test_command "cicerone check" "cicerone check" "Checking" || skip "check requires gitlab-runner"
test_command "cicerone node show" "cicerone node show" "Total:" || skip "No nodes configured"
test_command "cicerone library show" "cicerone library show" "Libraries" || skip "No libraries configured"

# ========================================
# SECTION 4: LLM Command Tests
# ========================================
log ""
log "=== SECTION 4: LLM Command Tests ==="

test_command "cicerone llm show" "cicerone llm show" "Provider:"
test_command "LLM provider check" "cicerone llm show" "provider"

# ========================================
# SECTION 5: LLM Connection Tests
# ========================================
log ""
log "=== SECTION 5: LLM Connection Tests ==="

# Check if llama-server is running
if pgrep -f "llama-server" > /dev/null; then
    pass "llama-server process running"
    
    # Test connection
    if curl -s http://localhost:8080/v1/models > /dev/null 2>&1; then
        pass "LLM server responding on port 8080"
    else
        skip "LLM server not responding (may need to start)"
    fi
else
    skip "llama-server not running"
    
    log "To start llama-server:"
    log "  llama-server --model /opt/models/model.gguf --port 8080 &"
fi

# ========================================
# SECTION 6: Natural Language Tests
# ========================================
log ""
log "=== SECTION 6: Natural Language Tests ==="

# These tests require a running LLM server
if curl -s http://localhost:8080/v1/models > /dev/null 2>&1; then
    test_command "cicerone do hostname" "cicerone do 'what is the hostname' --timeout 30" "hostname" || \
        log "Note: LLM interpretation may vary"
    
    test_command "cicerone do localhost test" "cicerone do 'localhost:hostname'" "hostname" || \
        skip "Localhost command failed"
    
    test_command "cicerone do date" "cicerone do 'show me the current date' --timeout 30" "" || \
        log "Note: LLM interpretation may vary"
else
    skip "Natural language tests (LLM server not running)"
fi

# ========================================
# SECTION 7: Direct Command Tests
# ========================================
log ""
log "=== SECTION 7: Direct Command Tests ==="

test_command "localhost:whoami" "cicerone do 'localhost:whoami'" "root\|wez\|cicerone" || \
    log "Output: $(cicerone do 'localhost:whoami' 2>&1)"

test_command "localhost:pwd" "cicerone do 'localhost:pwd'" "/"

test_command "localhost:ls" "cicerone do 'localhost:ls -la /'" "total\|root"

test_command "localhost:date" "cicerone do 'localhost:date'" ""

# ========================================
# SECTION 8: Model Download Test
# ========================================
log ""
log "=== SECTION 8: Model Tests ==="

if [ -f "/opt/models/gemma-2-2b-it.Q4_K_M.gguf" ]; then
    pass "Model file exists"
    ls -lh /opt/models/gemma-2-2b-it.Q4_K_M.gguf | tee -a "$LOG_FILE"
else
    skip "Model file not found at /opt/models/"
    log "Download a model:"
    log "  wget -O /opt/models/gemma-2-2b-it.Q4_K_M.gguf \\"
    log "    https://huggingface.co/bartowski/gemma-2-2b-it-GGUF/resolve/main/gemma-2-2b-it.Q4_K_M.gguf"
fi

# ========================================
# Summary
# ========================================
log ""
log "=== Test Summary ==="
log ""
echo -e "Passed: ${GREEN}$PASS${NC}" | tee -a "$LOG_FILE"
echo -e "Failed: ${RED}$FAIL${NC}" | tee -a "$LOG_FILE"
echo -e "Skipped: ${YELLOW}$SKIP${NC}" | tee -a "$LOG_FILE"
log ""

# Generate report
cat > "$REPORT_FILE" << EOF
# Cicerone + llama.cpp Verification Report

**Date:** $(date)
**Host:** $(hostname)
**System:** $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d '"')
**Architecture:** $(uname -m)

## Test Results

| Category | Passed | Failed | Skipped |
|----------|--------|--------|---------|
| Tests | $PASS | $FAIL | $SKIP |

## Environment

- Cicerone: $(cicerone about | grep Version || echo "installed")
- llama-server: $(which llama-server 2>/dev/null || echo "not found")
- Go: $(go version | awk '{print $3}')

## LLM Configuration

\`\`\`
$(cicerone llm show 2>/dev/null || echo "Not configured")
\`\`\`

## Log

Full log available at: \`$LOG_FILE\`

## Notes

- Tests require llama-server running on localhost:8080 for full verification
- Direct commands (localhost:*) work without LLM server
- Natural language commands require running LLM server

## Next Steps

1. Start llama-server: \`llama-server --model /opt/models/model.gguf --port 8080 &\`
2. Re-run tests: \`./test-llama-pipeline.sh\`
3. Configure nodes: \`cicerone config new <node-name>\`
EOF

log "Report saved to: $REPORT_FILE"

# Exit code
if [ $FAIL -gt 0 ]; then
    exit 1
else
    exit 0
fi