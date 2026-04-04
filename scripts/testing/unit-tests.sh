#!/usr/bin/env bash
# Runs Go unit tests and UI unit tests.
# Usage: ./scripts/testing/unit-tests.sh [--verbose]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

VERBOSE=""
if [[ "${1:-}" == "--verbose" ]]; then
    VERBOSE="-v"
fi

log_header "Unit Test Suite"

# Step 1: Go build
log_section "Step 1: Go Build"
if go build ./... 2>&1; then
    log_success "Go build: clean"
else
    log_fail "Go build: failed"
    print_summary
    exit 1
fi

# Step 2: Go unit tests
log_section "Step 2: Go Unit Tests"
if go test -short -count=1 $VERBOSE -coverprofile=coverage.out ./... 2>&1; then
    log_success "Go unit tests: passed"
    echo ""
    go tool cover -func=coverage.out | tail -1
else
    log_fail "Go unit tests: failed"
fi

# Step 3: UI unit tests
log_section "Step 3: UI Unit Tests"
if [ -d "ui/node_modules" ]; then
    if (cd ui && npm test 2>&1); then
        log_success "UI unit tests: passed"
    else
        log_fail "UI unit tests: failed"
    fi
else
    log_warning "UI dependencies not installed"
fi

print_summary
