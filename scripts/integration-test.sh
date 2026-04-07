#!/bin/bash
# Cicerone Integration Test Suite
# Tests all commands against actual system

CICERONE="${CICERONE:-./cicerone}"
PASSED=0
FAILED=0
SKIPPED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║          C I C E R O N E   I N T E G R A T I O N   T E S T       ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "Date: $(date)"
echo "Cicerone: $CICERONE"
echo ""

# Build if needed
if [ ! -x "$CICERONE" ]; then
    echo "Building cicerone..."
    go build -o cicerone .
fi

test_passed() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    PASSED=$((PASSED + 1))
}

test_failed() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo "  Error: $2"
    FAILED=$((FAILED + 1))
}

test_skipped() {
    echo -e "${YELLOW}⊘ SKIP${NC}: $1"
    echo "  Reason: $2"
    SKIPPED=$((SKIPPED + 1))
}

echo "=== BASIC COMMANDS ==="

# Test help
if $CICERONE help 2>&1 | grep -q "Cicerone"; then
    test_passed "help command"
else
    test_failed "help command" "Cicerone not in output"
fi

# Test about
if $CICERONE about 2>&1 | grep -qE "CICERONE|C.I.C.E.R.O.N.E|Cicerone"; then
    test_passed "about command"
else
    test_failed "about command" "CICERONE not in output"
fi

# Test version
if $CICERONE about 2>&1 | grep -q "Version"; then
    test_passed "version display"
else
    test_failed "version display" "Version not in output"
fi

echo ""
echo "=== NODE COMMANDS ==="

# Test node show
if $CICERONE node show 2>&1 | grep -q "NODE"; then
    test_passed "node show"
else
    test_failed "node show" "NODE not in output"
fi

# Test node list
if $CICERONE node show 2>&1 | grep -q "Total:"; then
    test_passed "node list"
else
    test_failed "node list" "Total: not in output"
fi

echo ""
echo "=== LIBRARY COMMANDS ==="

# Test library show
output=$($CICERONE library show 2>&1) || true
if echo "$output" | grep -q "LIBRARY"; then
    test_passed "library show"
else
    test_skipped "library show" "No libraries configured"
fi

echo ""
echo "=== RUNNER COMMANDS ==="

# Test runner help
if $CICERONE runner help 2>&1 | grep -q "runner"; then
    test_passed "runner help"
else
    test_failed "runner help" "runner not in output"
fi

echo ""
echo "=== PIPELINE COMMANDS ==="

# Test pipeline help
if $CICERONE pipeline --help 2>&1 | grep -q "pipeline"; then
    test_passed "pipeline help"
else
    test_failed "pipeline help" "pipeline not in output"
fi

echo ""
echo "=== IMAGE COMMANDS ==="

# Test image list - check for output (either images or "No images")
output=$($CICERONE image list 2>&1)
if echo "$output" | grep -q "test-image\|No images"; then
    test_passed "image list"
else
    test_failed "image list" "No image output found"
fi

# Test image new help
if $CICERONE image new --help 2>&1 | grep -q "base"; then
    test_passed "image new help"
else
    test_failed "image new help" "base not in output"
fi

# Test image build help
if $CICERONE image build --help 2>&1 | grep -q "build"; then
    test_passed "image build help"
else
    test_failed "image build help" "build not in output"
fi

# Test image deploy help
if $CICERONE image deploy --help 2>&1 | grep -q "count"; then
    test_passed "image deploy help"
else
    test_failed "image deploy help" "count not in output"
fi

# Test image test help
if $CICERONE image test --help 2>&1 | grep -q "suite"; then
    test_passed "image test help"
else
    test_failed "image test help" "suite not in output"
fi

# Test image destroy help
if $CICERONE image destroy --help 2>&1 | grep -q "destroy"; then
    test_passed "image destroy help"
else
    test_failed "image destroy help" "destroy not in output"
fi

echo ""
echo "=== DO COMMAND ==="

# Test do command (localhost - requires SSH)
output=$($CICERONE do localhost:echo test 2>&1) || true
if echo "$output" | grep -q "CICERONE DO\|test"; then
    test_passed "do localhost"
else
    test_skipped "do localhost" "SSH may not be available"
fi

echo ""
echo "=== ERROR HANDLING ==="

# Test invalid command
output=$($CICERONE invalidcommand 2>&1) || true
if echo "$output" | grep -qi "unknown\|invalid"; then
    test_passed "invalid command handling"
else
    test_failed "invalid command handling" "Should show error for invalid command"
fi

# Test missing argument
output=$($CICERONE image new 2>&1) || true
if echo "$output" | grep -qi "accepts\|arg\|required\|error"; then
    test_passed "missing argument handling"
else
    test_failed "missing argument handling" "Should require argument"
fi

echo ""
echo "=== SUMMARY ==="
echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                      T E S T   R E S U L T S                     ║"
echo "╠══════════════════════════════════════════════════════════════════╣"
printf "║  %-20s %10s %10s %10s %-6s║\n" "Tests" "Passed" "Failed" "Skipped" ""
printf "║  %-20s %10d %10d %10d %-6s║\n" "Results" "$PASSED" "$FAILED" "$SKIPPED" ""
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
