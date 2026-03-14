#!/usr/bin/env bash
set -euo pipefail

# Full test suite for Obeya CLI
# Runs all Go tests across all packages, with TUI golden file validation.
#
# Usage:
#   ./scripts/test.sh              # run all tests
#   ./scripts/test.sh --update     # run all tests + regenerate golden files
#   ./scripts/test.sh --tui        # run only TUI tests (fast feedback)
#   ./scripts/test.sh --golden     # run only golden file snapshot tests

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

UPDATE_FLAG=""
TEST_FILTER=""

while [[ "${1:-}" == -* ]]; do
    case "$1" in
        --update) UPDATE_FLAG="-update"; shift ;;
        --tui) TEST_FILTER="./internal/tui/"; shift ;;
        --golden) TEST_FILTER="./internal/tui/ -run TestGolden"; shift ;;
        *) echo "Unknown flag: $1"; exit 1 ;;
    esac
done

echo -e "${BOLD}=== Obeya Test Suite ===${NC}"
echo ""

# Step 1: Build check
echo -e "${BOLD}[1/4] Build check${NC}"
if go build ./... 2>&1; then
    echo -e "  ${GREEN}[ok]${NC} Build passes"
else
    echo -e "  ${RED}[FAIL]${NC} Build failed"
    exit 1
fi

# Step 2: Vet
echo -e "${BOLD}[2/4] Go vet${NC}"
if go vet ./... 2>&1; then
    echo -e "  ${GREEN}[ok]${NC} Vet passes"
else
    echo -e "  ${RED}[FAIL]${NC} Vet found issues"
    exit 1
fi

# Step 3: Unit + integration tests
echo -e "${BOLD}[3/4] Tests${NC}"
if [ -n "$TEST_FILTER" ]; then
    echo "  Running: go test $TEST_FILTER $UPDATE_FLAG -timeout 120s -v"
    # shellcheck disable=SC2086
    if go test $TEST_FILTER $UPDATE_FLAG -timeout 120s -v 2>&1 | tee /tmp/obeya-test-output.txt; then
        PASS_COUNT=$(grep -c "PASS:" /tmp/obeya-test-output.txt || true)
        FAIL_COUNT=$(grep -c "FAIL:" /tmp/obeya-test-output.txt || true)
        echo -e "  ${GREEN}[ok]${NC} ${PASS_COUNT} passed, ${FAIL_COUNT} failed"
    else
        echo -e "  ${RED}[FAIL]${NC} Tests failed"
        exit 1
    fi
else
    echo "  Running: go test ./... $UPDATE_FLAG -timeout 120s"
    # shellcheck disable=SC2086
    if go test ./... $UPDATE_FLAG -timeout 120s 2>&1 | tee /tmp/obeya-test-output.txt; then
        PKG_COUNT=$(grep -c "^ok" /tmp/obeya-test-output.txt || true)
        echo -e "  ${GREEN}[ok]${NC} ${PKG_COUNT} packages pass"
    else
        echo -e "  ${RED}[FAIL]${NC} Tests failed"
        grep "^FAIL" /tmp/obeya-test-output.txt || true
        exit 1
    fi
fi

# Step 4: Golden file check (only when running full suite)
if [ -z "$TEST_FILTER" ]; then
    echo -e "${BOLD}[4/4] Golden file snapshots${NC}"
    if go test ./internal/tui/ -run TestGolden -timeout 60s $UPDATE_FLAG 2>&1 | tee /tmp/obeya-golden-output.txt; then
        GOLDEN_COUNT=$(ls internal/tui/testdata/*.golden 2>/dev/null | wc -l | tr -d ' ')
        echo -e "  ${GREEN}[ok]${NC} ${GOLDEN_COUNT} golden files verified"
    else
        echo -e "  ${RED}[FAIL]${NC} Golden file mismatch — run './scripts/test.sh --update' to regenerate"
        exit 1
    fi
else
    echo -e "${BOLD}[4/4] Golden file snapshots${NC}"
    echo -e "  ${YELLOW}[skip]${NC} filtered run"
fi

echo ""
echo -e "${GREEN}${BOLD}All checks passed.${NC}"
