#!/usr/bin/env bash
# Shared shell functions for test scripts.
# Ported from WitFoo Analytics.

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

CHECKS_PASSED=0
CHECKS_FAILED=0

log_header() { echo -e "\n${BLUE}════════════════════════════════════════${NC}"; echo -e "${BLUE}  $1${NC}"; echo -e "${BLUE}════════════════════════════════════════${NC}\n"; }
log_section() { echo -e "\n${CYAN}── $1 ──${NC}\n"; }
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; CHECKS_PASSED=$((CHECKS_PASSED + 1)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; CHECKS_FAILED=$((CHECKS_FAILED + 1)); }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        log_fail "Missing dependency: $1"
        return 1
    fi
    log_info "Found: $1 ($(command -v "$1"))"
}

print_summary() {
    echo ""
    log_header "Test Summary"
    echo -e "${GREEN}  Passed: ${CHECKS_PASSED}${NC}"
    echo -e "${RED}  Failed: ${CHECKS_FAILED}${NC}"
    echo ""
    if [ "$CHECKS_FAILED" -gt 0 ]; then
        echo -e "${RED}  RESULT: FAILED${NC}"
        return 1
    else
        echo -e "${GREEN}  RESULT: PASSED${NC}"
        return 0
    fi
}
