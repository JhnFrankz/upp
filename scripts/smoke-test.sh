#!/usr/bin/env bash
# smoke-test.sh — Quick smoke test for the upp binary.
# Verifies basic functionality: bare dashboard, --help, --version, list, update --dry-run, init --ci, flag shorthands.
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

run_test_without_output() {
    local name="$1"
    local unexpected="$2"
    shift 2
    local output
    local exit_code=0

    output=$("$@" 2>&1) || exit_code=$?

    if [ "$exit_code" -ne 0 ]; then
        fail "$name" "exit code: $exit_code"
        return
    fi

    if echo "$output" | grep -q "$unexpected"; then
        fail "$name" "did not expect '$unexpected' in output"
    else
        pass "$name"
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

# Test 1: --help and --version
echo "1. Basic flags & 2-group help"
run_test_with_output "upp --help (Commands group)" "Commands" "$BINARY" --help
run_test_with_output "upp --help (Maintenance group)" "Maintenance" "$BINARY" --help
run_test_without_output "upp --help (no legacy Tool Commands)" "Tool Commands" "$BINARY" --help
run_test_without_output "upp --help (no legacy Config Commands)" "Config Commands" "$BINARY" --help
run_test_without_output "upp --help (no export)" "export" "$BINARY" --help
run_test_without_output "upp --help (no import)" "import" "$BINARY" --help
run_test_with_output "upp --version" "upp version" "$BINARY" --version

# Test 2: Subcommand help
echo ""
echo "2. Subcommand help"
run_test_with_output "upp init --help" "Detect installed tools" "$BINARY" init --help
run_test_with_output "upp update --help" "Process each enabled tool" "$BINARY" update --help
run_test_with_output "upp list --help" "Show all tools available" "$BINARY" list --help
run_test_with_output "upp self-update --help" "Check for a newer upp release" "$BINARY" self-update --help

# Test 3: Bare dashboard
echo ""
echo "3. Bare invocation dashboard"
TMPDIR_BARE=$(mktemp -d)
HOME_ORIG="${HOME:-}"
export HOME="$TMPDIR_BARE"
run_test_with_output "bare upp (no config -> guidance)" "No configuration found" "$BINARY"
run_test_with_output "bare upp (no config -> prompt init)" "upp init" "$BINARY"
"$BINARY" init --ci >/dev/null 2>&1 || true
run_test_with_output "bare upp (with config -> dashboard banner)" "upp" "$BINARY"
run_test_with_output "bare upp (with config -> commands guide)" "Commands:" "$BINARY"
export HOME="${HOME_ORIG:-$HOME}"
rm -rf "$TMPDIR_BARE"

# Test 4: list command
echo ""
echo "4. List command"
run_test "upp list" "$BINARY" list

# Test 5: read-only query surface (update --dry-run); check is pruned
echo ""
echo "5. Read-only query surface"
run_test_with_output "upp update --dry-run (query header)" "Dry run" "$BINARY" update --dry-run
run_test_exit_code "upp check (pruned, exit 1)" 1 "$BINARY" check

# Test 6: init --ci (creates config)
echo ""
echo "6. Init --ci"
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

# Test 7: Quiet flag and shorthand (-q) on the dry-run query surface
echo ""
echo "7. Quiet mode"
run_test "upp update -n --quiet" "$BINARY" update -n --quiet
run_test "upp update --dry-run -q" "$BINARY" update --dry-run -q

# Test 8: Verbose flag and shorthand (-v) on the dry-run query surface
echo ""
echo "8. Verbose mode"
run_test "upp update -n --verbose" "$BINARY" update -n --verbose
run_test "upp update --dry-run -v" "$BINARY" update --dry-run -v

# Test 9: Filter flags on the dry-run query surface
echo ""
echo "9. Filter flags"
run_test "upp update -n --only npm" "$BINARY" update -n --only npm
run_test "upp update -n --skip npm" "$BINARY" update -n --skip npm
run_test "upp update -n --only brew --skip npm (--only wins)" "$BINARY" update -n --only brew --skip npm

# Test 10: Dry-run flag and shorthand (-n)
echo ""
echo "10. Dry-run mode"
run_test "upp update --dry-run" "$BINARY" update --dry-run
run_test "upp update -n" "$BINARY" update -n

# Test 11: Pruned commands rejected with exit 1
echo ""
echo "11. Pruned commands error handling"
run_test_exit_code "upp export (pruned, exit 1)" 1 "$BINARY" export
run_test_exit_code "upp import (pruned, exit 1)" 1 "$BINARY" import "/tmp/nonexistent.toml"

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
