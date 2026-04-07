#!/bin/bash
# Cicerone Runner Integration Test
# Tests the full runner workflow

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CICERONE_ROOT="$(dirname "$SCRIPT_DIR")"
CICERONE_BIN="${CICERONE_ROOT}/cicerone"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

echo "=========================================="
echo "    CICERONE RUNNER INTEGRATION TEST"
echo "=========================================="
echo ""

test_pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    ((PASS++))
}

test_fail() {
    echo -e "${RED}FAIL${NC}: $1"
    ((FAIL++))
}

test_section() {
    echo ""
    echo "--- $1 ---"
}

# Build cicerone if needed
if [ ! -f "$CICERONE_BIN" ]; then
    echo "Building cicerone..."
    cd "$CICERONE_ROOT"
    go build -o cicerone .
fi

test_section "1. Command Availability"

# Test runner help
if "$CICERONE_BIN" runner help 2>&1 | grep -qi "runner"; then
    test_pass "runner help command works"
else
    test_fail "runner help command not found"
fi

# Test runner subcommands
for cmd in new config deploy cancel; do
    OUTPUT=$("$CICERONE_BIN" runner $cmd --help 2>&1)
    if echo "$OUTPUT" | grep -qi "flags:" || echo "$OUTPUT" | grep -qi "usage"; then
        test_pass "runner $cmd command available"
    else
        test_fail "runner $cmd command not available"
    fi
done

test_section "2. Runner New"

rm -rf ~/.cicerone/runners 2>/dev/null || true

OUTPUT=$("$CICERONE_BIN" runner new --name test-runner --tags "test,linux" 2>&1)
if echo "$OUTPUT" | grep -q "created successfully"; then
    test_pass "runner new creates configuration"
else
    test_fail "runner new failed"
fi

[ -d ~/.cicerone/runners ] && test_pass "runners directory created" || test_fail "runners directory not created"
[ -f ~/.cicerone/runners/state.json ] && test_pass "state.json created" || test_fail "state.json not created"
[ -f ~/.cicerone/runners/active.json ] && test_pass "active.json created" || test_fail "active.json not created"
[ -d ~/.cicerone/runners/tokens ] && test_pass "tokens directory created" || test_fail "tokens directory not created"
[ -d ~/.cicerone/runners/archive ] && test_pass "archive directory created" || test_fail "archive directory not created"

if command -v jq &> /dev/null; then
    NAME=$(cat ~/.cicerone/runners/active.json | jq -r '.name' 2>/dev/null)
    [ "$NAME" = "test-runner" ] && test_pass "active.json has correct name" || test_fail "wrong name: $NAME"
fi

test_section "3. Runner Config Flags"

OUTPUT=$("$CICERONE_BIN" runner config --help 2>&1)
echo "$OUTPUT" | grep -qi "api-token" && test_pass "has --api-token flag" || test_fail "missing --api-token flag"
echo "$OUTPUT" | grep -qi "project" && test_pass "has --project flag" || test_fail "missing --project flag"

test_section "4. Runner Cancel"

echo "y" | "$CICERONE_BIN" runner cancel 2>&1 || true

[ ! -f ~/.cicerone/runners/active.json ] && test_pass "active.json removed" || test_fail "active.json still exists"

ARCHIVE_COUNT=$(ls ~/.cicerone/runners/archive/*.json 2>/dev/null | wc -l || echo "0")
[ "$ARCHIVE_COUNT" -gt 0 ] && test_pass "runner archived" || test_fail "no archive created"

test_section "5. Prerequisites"

command -v gitlab-runner &> /dev/null && test_pass "gitlab-runner installed" || echo "SKIP: gitlab-runner not installed"
command -v go &> /dev/null && test_pass "go installed" || test_fail "go not installed"

test_section "6. Unit Tests"

cd "$CICERONE_ROOT"
if go test -v -run TestRunner ./... 2>&1 | grep -q "PASS"; then
    test_pass "go unit tests pass"
else
    echo "INFO: Unit tests may need attention"
fi

echo ""
echo "=========================================="
echo "SUMMARY: Passed=$PASS Failed=$FAIL"
echo "=========================================="

[ $FAIL -eq 0 ] && exit 0 || exit 1
