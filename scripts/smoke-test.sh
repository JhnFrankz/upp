#!/usr/bin/env bash
# smoke-test.sh — Quick smoke test for the upp binary.
# Verifies basic functionality: --help, --version, list, check, init --ci.
# Exit codes: 0 = all tests passed, 1 = at least one test failed.
#
# Usage:
#   ./scripts/smoke-test.sh          # build and test
#   ./scripts/smoke-test.sh --skip-build  # test existing binary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/upp"
PASS=0
FAIL=0
TOTAL=0

# Colors (if terminal supports it)
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    GREEN=''
    RED=''
    YELLOW=''
    NC=''
fi

# --- Helpers ---

pass() {
    PASS=$((PASS + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${GREEN}✓${NC} $1"
}

fail() {
    FAIL=$((FAIL + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${RED}✗${NC} $1"
    if [ -n "${2:-}" ]; then
        echo -e "    ${YELLOW}Output: $2${NC}"
    fi
}

run_test() {
    local name="$1"
    shift
    local output
    local exit_code=0

    output=$("$@" 2>&1) || exit_code=$?

    if [ "$exit_code" -eq 0 ]; then
        pass "$name"
    else
        fail "$name" "exit code: $exit_code"
    fi
}

run_test_with_output() {
    local name="$1"
    local expected="$2"
    shift 2
    local output
    local exit_code=0

    output=$("$@" 2>&1) || exit_code=$?

    if [ "$exit_code" -ne 0 ]; then
        fail "$name" "exit code: $exit_code"
        return
    fi

    if echo "$output" | grep -q "$expected"; then
        pass "$name"
    else
        fail "$name" "expected '$expected' in output"
    fi
}

run_test_exit_code() {
    local name="$1"
    local expected_exit="$2"
    shift 2
    local exit_code=0

    "$@" >/dev/null 2>&1 || exit_code=$?

    if [ "$exit_code" -eq "$expected_exit" ]; then
        pass "$name"
    else
        fail "$name" "expected exit code $expected_exit, got $exit_code"
    fi
}

# --- Build ---

echo ""
echo "upp smoke test"
echo "=============="
echo ""

if [ "${1:-}" != "--skip-build" ]; then
    echo "Building binary..."
    cd "$PROJECT_DIR"
    go build -o upp ./cmd/upp
    echo "Build complete."
    echo ""
fi

if [ ! -x "$BINARY" ]; then
    echo "Binary not found at $BINARY"
    echo "Run without --skip-build, or build manually first."
    exit 1
fi

# --- Tests ---

echo "Running smoke tests..."
echo ""

# Test 1: --help
echo "1. Basic flags"
run_test_with_output "upp --help" "upp updates your development tools" "$BINARY" --help
run_test_with_output "upp --version" "upp version" "$BINARY" --version

# Test 2: Subcommand help
echo ""
echo "2. Subcommand help"
run_test_with_output "upp init --help" "Detect installed tools" "$BINARY" init --help
run_test_with_output "upp update --help" "Process each enabled tool" "$BINARY" update --help
run_test_with_output "upp check --help" "Query each enabled tool" "$BINARY" check --help
run_test_with_output "upp list --help" "Show all tools available" "$BINARY" list --help
run_test_with_output "upp export --help" "Output the current configuration" "$BINARY" export --help
run_test_with_output "upp import --help" "Replace the current configuration" "$BINARY" import --help

# Test 3: list command
echo ""
echo "3. List command"
run_test "upp list" "$BINARY" list

# Test 4: check command
echo ""
echo "4. Check command"
run_test "upp check" "$BINARY" check

# Test 5: init --ci (creates config)
echo ""
echo "5. Init --ci"
TMPDIR_INIT=$(mktemp -d)
HOME_ORIG="${HOME:-}"
export HOME="$TMPDIR_INIT"
run_test "upp init --ci" "$BINARY" init --ci
export HOME="${HOME_ORIG:-$HOME}"

# Verify config was created
CONFIG_DIR="$TMPDIR_INIT/.config/upp"
if [ -f "$CONFIG_DIR/config.toml" ]; then
    pass "Config file created at $CONFIG_DIR/config.toml"
else
    fail "Config file not created"
fi
rm -rf "$TMPDIR_INIT"

# Test 6: --quiet flag
echo ""
echo "6. Quiet mode"
run_test "upp check --quiet" "$BINARY" check --quiet

# Test 7: --only filter
echo ""
echo "7. Filter flags"
run_test "upp check --only npm" "$BINARY" check --only npm

# Test 8: --skip filter
run_test "upp check --skip npm" "$BINARY" check --skip npm

# Test 8b: --only wins over --skip
run_test "upp check --only brew --skip npm (--only wins)" "$BINARY" check --only brew --skip npm

# Test 9: --dry-run
echo ""
echo "9. Dry-run mode"
run_test "upp update --dry-run" "$BINARY" update --dry-run

# Test 10: import with missing file (should fail)
echo ""
echo "10. Error handling"
run_test_exit_code "upp import missing.toml (expected error)" 1 "$BINARY" import "/nonexistent/file.toml"

# Test 11: export
echo ""
echo "11. Export"
TMPDIR_EXPORT=$(mktemp -d)
HOME_ORIG="${HOME:-}"
export HOME="$TMPDIR_EXPORT"
"$BINARY" init --ci >/dev/null 2>&1 || true
run_test "upp export -o /tmp/test-export.toml" "$BINARY" export -o /tmp/test-export.toml
rm -f /tmp/test-export.toml
export HOME="${HOME_ORIG:-$HOME}"
rm -rf "$TMPDIR_EXPORT"

# --- Summary ---

echo ""
echo "=============="
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}, $TOTAL total"

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
