#!/usr/bin/env bash
# Security scanning: govulncheck, npm audit.
# Usage: ./scripts/testing/security.sh [--verbose]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

log_header "Security Scan"

# Step 1: Go vulnerability check
log_section "Step 1: Go Vulnerability Check"
if command -v govulncheck &> /dev/null; then
    if govulncheck ./... 2>&1; then
        log_success "govulncheck: no vulnerabilities found"
    else
        log_fail "govulncheck: vulnerabilities detected"
    fi
else
    log_warning "govulncheck not installed (go install golang.org/x/vuln/cmd/govulncheck@latest)"
fi

# Step 2: npm audit
log_section "Step 2: npm Audit"
if [ -d "ui/node_modules" ]; then
    if (cd ui && npm audit --audit-level=high 2>&1); then
        log_success "npm audit: no high/critical vulnerabilities"
    else
        log_fail "npm audit: vulnerabilities detected"
    fi
else
    log_warning "UI dependencies not installed"
fi

# Step 3: Go race detection
log_section "Step 3: Race Condition Detection"
if go test -race -short -count=1 ./... 2>&1; then
    log_success "Race detection: clean"
else
    log_fail "Race detection: data races found"
fi

print_summary
