#!/usr/bin/env bash
# Runs all linters: Go (vet, golangci-lint) + UI (eslint, svelte-check).
# Usage: ./scripts/testing/linter.sh [--verbose]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

log_header "Linter Suite"

# Step 1: Go formatting
log_section "Step 1: Go Formatting"
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor/ | grep -v node_modules/ | grep -v utils/internet-research/ || true)
if [ -z "$UNFORMATTED" ]; then
    log_success "gofmt: all files formatted"
else
    log_fail "gofmt: unformatted files: $UNFORMATTED"
fi

# Step 2: Go vet
log_section "Step 2: Go Vet"
if go vet ./... 2>&1; then
    log_success "go vet: clean"
else
    log_fail "go vet: issues found"
fi

# Step 3: golangci-lint (if available)
log_section "Step 3: golangci-lint"
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --config .golangci.yml --timeout 5m ./... 2>&1; then
        log_success "golangci-lint: clean"
    else
        log_fail "golangci-lint: issues found"
    fi
else
    log_warning "golangci-lint not installed, skipping"
fi

# Step 4: UI linting
log_section "Step 4: UI Linting"
if [ -d "ui/node_modules" ]; then
    if (cd ui && npm run lint 2>&1); then
        log_success "eslint: clean"
    else
        log_fail "eslint: issues found"
    fi
else
    log_warning "UI dependencies not installed (run 'cd ui && npm ci')"
fi

# Step 5: Svelte check
log_section "Step 5: Svelte Check"
if [ -d "ui/node_modules" ]; then
    if (cd ui && npm run check 2>&1); then
        log_success "svelte-check: clean"
    else
        log_fail "svelte-check: issues found"
    fi
else
    log_warning "UI dependencies not installed"
fi

print_summary
