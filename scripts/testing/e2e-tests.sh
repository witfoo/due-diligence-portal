#!/usr/bin/env bash
# Runs Playwright E2E tests against a fresh Docker container.
# Usage: ./scripts/testing/e2e-tests.sh [--headed] [--verbose] [--keep]
#
# Starts fresh container, runs Playwright tests, then tears down.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

CONTAINER_NAME="dd-portal-e2e-test"
HOST_PORT=9192
HEADED=""
KEEP_CONTAINER=false
EXTRA_ARGS=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --headed) HEADED="1"; shift ;;
        --keep) KEEP_CONTAINER=true; shift ;;
        --verbose) EXTRA_ARGS="$EXTRA_ARGS --reporter=list"; shift ;;
        --workers=*) EXTRA_ARGS="$EXTRA_ARGS --workers=${1#*=}"; shift ;;
        *) EXTRA_ARGS="$EXTRA_ARGS $1"; shift ;;
    esac
done

cleanup() {
    if [ "$KEEP_CONTAINER" = false ]; then
        docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    fi
}
trap cleanup EXIT

log_header "E2E Tests (Playwright)"

# Step 1: Start fresh container.
log_section "Step 1: Start fresh container"
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

docker build -t dd-portal:e2e-test "$PROJECT_DIR" -q 2>&1 | tail -1
log_success "Docker image built"

docker run -d \
    --name "$CONTAINER_NAME" \
    -p "${HOST_PORT}:8080" \
    -e DD_TLS_MODE=none \
    -e DD_JWT_SECRET="e2e-test-secret-32-chars-minimum!!" \
    -e DD_ADMIN_EMAIL="admin@localhost" \
    -e DD_ADMIN_PASSWORD="testpass123" \
    dd-portal:e2e-test >/dev/null

# Wait for ready.
for i in $(seq 1 30); do
    if curl -sf "http://localhost:${HOST_PORT}/ready" >/dev/null 2>&1; then break; fi
    if [ "$i" -eq 30 ]; then
        log_fail "Service not ready"
        docker logs "$CONTAINER_NAME"
        exit 1
    fi
    sleep 1
done
log_success "Service ready on port $HOST_PORT"

# Step 2: Install Playwright browsers if needed.
log_section "Step 2: Check Playwright browsers"
cd "$PROJECT_DIR/ui"
if ! npx playwright install --dry-run chromium 2>/dev/null | grep -q "already"; then
    npx playwright install chromium 2>&1 | tail -3
fi
log_success "Playwright chromium ready"

# Step 3: Run tests.
log_section "Step 3: Run Playwright tests"
export BASE_URL="http://localhost:${HOST_PORT}"
export DD_TEST_ADMIN_EMAIL="admin@localhost"
export DD_TEST_ADMIN_PASSWORD="testpass123"
export HEADED="${HEADED:-}"

if npx playwright test \
    --project=chromium --project=chromium-no-auth \
    $EXTRA_ARGS 2>&1; then
    log_success "All E2E tests passed"
else
    log_fail "E2E tests failed"
fi

cd "$PROJECT_DIR"
print_summary
