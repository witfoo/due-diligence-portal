#!/usr/bin/env bash
# Comprehensive system tests for the Due Diligence Portal.
# Starts a fresh Docker container, validates all API endpoints, then tears down.
# Usage: ./scripts/testing/system-tests.sh [--keep] [--verbose]
#
# Each run starts with fresh data (no persistent volumes).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

CONTAINER_NAME="dd-portal-system-test"
HOST_PORT=9191
BASE_URL="http://localhost:${HOST_PORT}"
JWT_SECRET="system-test-secret-32-chars-min!!"
ADMIN_EMAIL="admin@test.com"
ADMIN_PASSWORD="SystemTest123!"
KEEP_CONTAINER=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --keep) KEEP_CONTAINER=true; shift ;;
        --verbose) VERBOSE=true; shift ;;
        *) shift ;;
    esac
done

cleanup() {
    if [ "$KEEP_CONTAINER" = false ]; then
        log_info "Cleaning up container..."
        docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    else
        log_info "Container kept: $CONTAINER_NAME (port $HOST_PORT)"
    fi
}
trap cleanup EXIT

# ============================================================
# STEP 0: Build and start fresh container
# ============================================================
log_header "System Tests"

log_section "Step 0: Build and start fresh container"

# Remove any existing container.
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# Build image.
log_info "Building Docker image..."
docker build -t dd-portal:system-test "$PROJECT_DIR" -q 2>&1 | tail -1
log_success "Docker image built"

# Start container with NO volumes (fresh data every run).
log_info "Starting container on port $HOST_PORT..."
docker run -d \
    --name "$CONTAINER_NAME" \
    -p "${HOST_PORT}:8080" \
    -e DD_TLS_MODE=none \
    -e DD_JWT_SECRET="$JWT_SECRET" \
    -e DD_ADMIN_EMAIL="$ADMIN_EMAIL" \
    -e DD_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
    dd-portal:system-test >/dev/null

# Wait for health check.
log_info "Waiting for service to be ready..."
for i in $(seq 1 30); do
    if curl -sf "${BASE_URL}/health" >/dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 30 ]; then
        log_fail "Service did not become healthy in 30 seconds"
        docker logs "$CONTAINER_NAME"
        exit 1
    fi
    sleep 1
done
log_success "Service is healthy"

# Helper functions.
api_get() {
    curl -sf -H "Authorization: Bearer $TOKEN" "${BASE_URL}/api/v1$1" 2>/dev/null
}
api_post() {
    curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${BASE_URL}/api/v1$1" 2>/dev/null
}
api_put() {
    curl -sf -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${BASE_URL}/api/v1$1" 2>/dev/null
}
api_patch() {
    curl -sf -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${BASE_URL}/api/v1$1" 2>/dev/null
}
api_delete() {
    curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" "${BASE_URL}/api/v1$1" 2>/dev/null
}
json_field() {
    python3 -c "import sys,json; print(json.load(sys.stdin)$1)" 2>/dev/null
}

# ============================================================
# STEP 1: Health endpoints
# ============================================================
log_section "Step 1: Health Endpoints"

HEALTH=$(curl -sf "${BASE_URL}/health")
if echo "$HEALTH" | json_field "['status']" | grep -q "healthy"; then
    log_success "GET /health returns healthy"
else
    log_fail "GET /health did not return healthy"
fi

READY=$(curl -sf "${BASE_URL}/ready")
if echo "$READY" | json_field "['checks']['sqlite']" | grep -q "ok"; then
    log_success "GET /ready returns sqlite ok"
else
    log_fail "GET /ready sqlite check failed"
fi

VERSION=$(curl -sf "${BASE_URL}/version")
if echo "$VERSION" | json_field "['version']" | grep -q "dev"; then
    log_success "GET /version returns version"
else
    log_fail "GET /version failed"
fi

# ============================================================
# STEP 2: Authentication
# ============================================================
log_section "Step 2: Authentication"

# Login.
LOGIN_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
    "${BASE_URL}/api/v1/auth/login")
TOKEN=$(echo "$LOGIN_RESP" | json_field "['data']['access_token']")
if [ -n "$TOKEN" ] && [ "$TOKEN" != "None" ]; then
    log_success "POST /auth/login returns JWT token"
else
    log_fail "POST /auth/login failed to return token"
    echo "$LOGIN_RESP"
    exit 1
fi

# Wrong password.
WRONG=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
    -d '{"email":"admin@test.com","password":"wrong"}' \
    "${BASE_URL}/api/v1/auth/login")
if [ "$WRONG" = "401" ]; then
    log_success "POST /auth/login rejects wrong password (401)"
else
    log_fail "POST /auth/login should return 401 for wrong password, got $WRONG"
fi

# Me.
ME=$(api_get "/auth/me")
if echo "$ME" | json_field "['data']['email']" | grep -q "$ADMIN_EMAIL"; then
    log_success "GET /auth/me returns current user"
else
    log_fail "GET /auth/me failed"
fi

# Refresh token.
REFRESH_TOKEN=$(echo "$LOGIN_RESP" | json_field "['data']['refresh_token']")
REFRESH_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
    -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" \
    "${BASE_URL}/api/v1/auth/refresh")
NEW_TOKEN=$(echo "$REFRESH_RESP" | json_field "['data']['access_token']")
if [ -n "$NEW_TOKEN" ] && [ "$NEW_TOKEN" != "None" ]; then
    log_success "POST /auth/refresh returns new access token"
else
    log_fail "POST /auth/refresh failed"
fi

# Unauthenticated access.
UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/api/v1/auth/me")
if [ "$UNAUTH" = "401" ]; then
    log_success "Unauthenticated request returns 401"
else
    log_fail "Unauthenticated should return 401, got $UNAUTH"
fi

# ============================================================
# STEP 3: Categories
# ============================================================
log_section "Step 3: Categories"

# List (should have 10 seeded categories).
CATS=$(api_get "/categories")
CAT_COUNT=$(echo "$CATS" | json_field "['meta']['count']" 2>/dev/null || echo "$CATS" | json_field "[len('data')]" 2>/dev/null || echo "0")
if [ "$CAT_COUNT" -ge 10 ] 2>/dev/null; then
    log_success "GET /categories returns $CAT_COUNT seeded categories"
else
    # Try alternate parsing.
    CAT_COUNT=$(echo "$CATS" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data',[])) if isinstance(d.get('data'), list) else 0)" 2>/dev/null || echo "0")
    if [ "$CAT_COUNT" -ge 10 ] 2>/dev/null; then
        log_success "GET /categories returns $CAT_COUNT seeded categories"
    else
        log_fail "GET /categories expected >= 10, got $CAT_COUNT"
    fi
fi

# Create category.
NEW_CAT=$(api_post "/categories" '{"name":"Test Category","slug":"test-system-cat","description":"System test category"}')
NEW_CAT_ID=$(echo "$NEW_CAT" | json_field "['data']['id']")
if [ -n "$NEW_CAT_ID" ] && [ "$NEW_CAT_ID" != "None" ]; then
    log_success "POST /categories creates new category (id=$NEW_CAT_ID)"
else
    log_fail "POST /categories failed"
fi

# ============================================================
# STEP 4: Documents
# ============================================================
log_section "Step 4: Documents"

# Upload document (multipart).
UPLOAD_RESP=$(curl -sf -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@${PROJECT_DIR}/LICENSE;type=text/plain" \
    -F "name=Test License" \
    -F "description=System test document" \
    -F "category_id=cat-corporate" \
    -F "tags=test,system" \
    "${BASE_URL}/api/v1/documents")
DOC_ID=$(echo "$UPLOAD_RESP" | json_field "['data']['document']['id']" 2>/dev/null || echo "$UPLOAD_RESP" | json_field "['data']['id']" 2>/dev/null)
if [ -n "$DOC_ID" ] && [ "$DOC_ID" != "None" ]; then
    log_success "POST /documents uploads document (id=$DOC_ID)"
else
    log_fail "POST /documents upload failed"
    if [ "$VERBOSE" = true ]; then echo "$UPLOAD_RESP"; fi
fi

# List documents.
DOCS=$(api_get "/documents")
DOC_LIST_SUCCESS=$(echo "$DOCS" | json_field "['success']")
if [ "$DOC_LIST_SUCCESS" = "True" ]; then
    log_success "GET /documents lists documents"
else
    log_fail "GET /documents failed"
fi

# Get document by ID.
if [ -n "$DOC_ID" ] && [ "$DOC_ID" != "None" ]; then
    DOC_DETAIL=$(api_get "/documents/$DOC_ID")
    DOC_DETAIL_SUCCESS=$(echo "$DOC_DETAIL" | json_field "['success']")
    if [ "$DOC_DETAIL_SUCCESS" = "True" ]; then
        log_success "GET /documents/:id returns document details"
    else
        log_fail "GET /documents/:id failed"
    fi

    # Download document.
    DL_STATUS=$(curl -sf -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" \
        "${BASE_URL}/api/v1/documents/$DOC_ID/download")
    if [ "$DL_STATUS" = "200" ]; then
        log_success "GET /documents/:id/download returns file (200)"
    else
        log_fail "GET /documents/:id/download returned $DL_STATUS"
    fi

    # Search documents.
    SEARCH=$(api_post "/documents/search" '{"query":"License"}')
    SEARCH_SUCCESS=$(echo "$SEARCH" | json_field "['success']")
    if [ "$SEARCH_SUCCESS" = "True" ]; then
        log_success "POST /documents/search finds documents"
    else
        log_fail "POST /documents/search failed"
    fi

    # Upload new version.
    VER_RESP=$(curl -sf -X POST \
        -H "Authorization: Bearer $TOKEN" \
        -F "file=@${PROJECT_DIR}/LICENSE;type=text/plain" \
        -F "change_note=Updated version" \
        "${BASE_URL}/api/v1/documents/$DOC_ID/versions")
    VER_SUCCESS=$(echo "$VER_RESP" | json_field "['success']")
    if [ "$VER_SUCCESS" = "True" ]; then
        log_success "POST /documents/:id/versions uploads new version"
    else
        log_fail "POST /documents/:id/versions failed"
    fi
fi

# ============================================================
# STEP 5: Permissions
# ============================================================
log_section "Step 5: Permissions"

if [ -n "$DOC_ID" ] && [ "$DOC_ID" != "None" ]; then
    # Get admin user ID.
    ADMIN_ID=$(echo "$ME" | json_field "['data']['user_id']")

    # Grant permission.
    GRANT=$(api_post "/permissions" "{\"user_id\":\"$ADMIN_ID\",\"resource_type\":\"document\",\"resource_id\":\"$DOC_ID\",\"access_level\":\"download\"}")
    GRANT_SUCCESS=$(echo "$GRANT" | json_field "['success']")
    if [ "$GRANT_SUCCESS" = "True" ]; then
        log_success "POST /permissions grants access"
    else
        log_fail "POST /permissions failed"
        if [ "$VERBOSE" = true ]; then echo "$GRANT"; fi
    fi

    # List permissions for document.
    PERMS=$(api_get "/permissions/document/$DOC_ID")
    PERMS_SUCCESS=$(echo "$PERMS" | json_field "['success']")
    if [ "$PERMS_SUCCESS" = "True" ]; then
        log_success "GET /permissions/document/:id lists grants"
    else
        log_fail "GET /permissions/document/:id failed"
    fi
fi

# ============================================================
# STEP 6: Q&A
# ============================================================
log_section "Step 6: Q&A"

# Create thread.
QA_RESP=$(api_post "/qa" '{"subject":"Question about financials"}')
THREAD_ID=$(echo "$QA_RESP" | json_field "['data']['id']")
if [ -n "$THREAD_ID" ] && [ "$THREAD_ID" != "None" ]; then
    log_success "POST /qa creates Q&A thread (id=$THREAD_ID)"
else
    log_fail "POST /qa failed"
fi

# List threads.
THREADS=$(api_get "/qa")
THREADS_SUCCESS=$(echo "$THREADS" | json_field "['success']")
if [ "$THREADS_SUCCESS" = "True" ]; then
    log_success "GET /qa lists threads"
else
    log_fail "GET /qa failed"
fi

if [ -n "$THREAD_ID" ] && [ "$THREAD_ID" != "None" ]; then
    # Post message.
    MSG_RESP=$(api_post "/qa/$THREAD_ID/messages" '{"body":"This is a response to the question."}')
    MSG_SUCCESS=$(echo "$MSG_RESP" | json_field "['success']")
    if [ "$MSG_SUCCESS" = "True" ]; then
        log_success "POST /qa/:id/messages posts message"
    else
        log_fail "POST /qa/:id/messages failed"
    fi

    # Get thread with messages.
    THREAD_DETAIL=$(api_get "/qa/$THREAD_ID")
    THREAD_DETAIL_SUCCESS=$(echo "$THREAD_DETAIL" | json_field "['success']")
    if [ "$THREAD_DETAIL_SUCCESS" = "True" ]; then
        log_success "GET /qa/:id returns thread with messages"
    else
        log_fail "GET /qa/:id failed"
    fi

    # Change status.
    STATUS_RESP=$(api_patch "/qa/$THREAD_ID/status" '{"status":"answered"}')
    STATUS_SUCCESS=$(echo "$STATUS_RESP" | json_field "['success']")
    if [ "$STATUS_SUCCESS" = "True" ]; then
        log_success "PATCH /qa/:id/status changes thread status"
    else
        log_fail "PATCH /qa/:id/status failed"
    fi
fi

# ============================================================
# STEP 7: NDA
# ============================================================
log_section "Step 7: NDA"

# Create template.
NDA_RESP=$(api_post "/nda/templates" '{"name":"Standard NDA","content":"This is a non-disclosure agreement..."}')
NDA_ID=$(echo "$NDA_RESP" | json_field "['data']['id']")
if [ -n "$NDA_ID" ] && [ "$NDA_ID" != "None" ]; then
    log_success "POST /nda/templates creates NDA template (id=$NDA_ID)"
else
    log_fail "POST /nda/templates failed"
fi

# Check status (should require signing).
NDA_STATUS=$(api_get "/nda/status")
NDA_SIGNED=$(echo "$NDA_STATUS" | json_field "['data']['signed']")
if [ "$NDA_SIGNED" = "False" ]; then
    log_success "GET /nda/status reports NDA not signed"
else
    log_fail "GET /nda/status should report not signed"
fi

# Sign NDA.
if [ -n "$NDA_ID" ] && [ "$NDA_ID" != "None" ]; then
    SIGN_RESP=$(api_post "/nda/sign/$NDA_ID" '{"signer_name":"Admin User","signer_company":"Test Corp"}')
    SIGN_SUCCESS=$(echo "$SIGN_RESP" | json_field "['success']")
    if [ "$SIGN_SUCCESS" = "True" ]; then
        log_success "POST /nda/sign/:id signs NDA"
    else
        log_fail "POST /nda/sign/:id failed"
    fi

    # Check status again (should be signed now).
    NDA_STATUS2=$(api_get "/nda/status")
    NDA_SIGNED2=$(echo "$NDA_STATUS2" | json_field "['data']['signed']")
    if [ "$NDA_SIGNED2" = "True" ]; then
        log_success "GET /nda/status reports NDA signed after signing"
    else
        log_fail "GET /nda/status should report signed"
    fi

    # List signatures.
    SIGS=$(api_get "/nda/signatures")
    SIGS_SUCCESS=$(echo "$SIGS" | json_field "['success']")
    if [ "$SIGS_SUCCESS" = "True" ]; then
        log_success "GET /nda/signatures lists signatures"
    else
        log_fail "GET /nda/signatures failed"
    fi
fi

# ============================================================
# STEP 8: Analytics
# ============================================================
log_section "Step 8: Analytics"

# Record view event.
if [ -n "$DOC_ID" ] && [ "$DOC_ID" != "None" ]; then
    VIEW_RESP=$(api_post "/analytics/view-event" "{\"document_id\":\"$DOC_ID\",\"duration_ms\":5000,\"page_count\":3}")
    VIEW_SUCCESS=$(echo "$VIEW_RESP" | json_field "['success']")
    if [ "$VIEW_SUCCESS" = "True" ]; then
        log_success "POST /analytics/view-event records view"
    else
        log_fail "POST /analytics/view-event failed"
    fi
fi

# Dashboard.
DASHBOARD=$(api_get "/analytics/dashboard")
DASHBOARD_SUCCESS=$(echo "$DASHBOARD" | json_field "['success']")
if [ "$DASHBOARD_SUCCESS" = "True" ]; then
    log_success "GET /analytics/dashboard returns engagement summary"
else
    log_fail "GET /analytics/dashboard failed"
fi

# ============================================================
# STEP 9: Branding
# ============================================================
log_section "Step 9: Branding"

# Get config.
BRAND=$(api_get "/branding")
BRAND_NAME=$(echo "$BRAND" | json_field "['data']['company_name']")
if [ -n "$BRAND_NAME" ]; then
    log_success "GET /branding returns config (company=$BRAND_NAME)"
else
    log_fail "GET /branding failed"
fi

# Update config.
BRAND_UP=$(api_put "/branding" '{"company_name":"System Test Corp","primary_color":"#ff0000"}')
BRAND_UP_SUCCESS=$(echo "$BRAND_UP" | json_field "['success']")
if [ "$BRAND_UP_SUCCESS" = "True" ]; then
    log_success "PUT /branding updates config"
else
    log_fail "PUT /branding failed"
fi

# Reset.
BRAND_RESET=$(api_delete "/branding")
BRAND_RESET_SUCCESS=$(echo "$BRAND_RESET" | json_field "['success']")
if [ "$BRAND_RESET_SUCCESS" = "True" ]; then
    log_success "DELETE /branding resets to defaults"
else
    log_fail "DELETE /branding failed"
fi

# ============================================================
# STEP 10: Watermark
# ============================================================
log_section "Step 10: Watermark"

# Get config.
WM=$(api_get "/watermark")
WM_ENABLED=$(echo "$WM" | json_field "['data']['enabled']")
if [ "$WM_ENABLED" = "False" ]; then
    log_success "GET /watermark returns config (disabled by default)"
else
    log_fail "GET /watermark failed"
fi

# Update.
WM_UP=$(api_put "/watermark" '{"enabled":true,"position":"bottom","opacity":0.3}')
WM_UP_SUCCESS=$(echo "$WM_UP" | json_field "['success']")
if [ "$WM_UP_SUCCESS" = "True" ]; then
    log_success "PUT /watermark enables watermark"
else
    log_fail "PUT /watermark failed"
fi

# Reset.
WM_RESET=$(api_delete "/watermark")
WM_RESET_ENABLED=$(echo "$WM_RESET" | json_field "['data']['enabled']")
if [ "$WM_RESET_ENABLED" = "False" ]; then
    log_success "DELETE /watermark resets to defaults"
else
    log_fail "DELETE /watermark failed"
fi

# ============================================================
# STEP 11: Audit Log
# ============================================================
log_section "Step 11: Audit Log"

# List (should have entries from all previous operations).
AUDIT=$(api_get "/audit")
AUDIT_SUCCESS=$(echo "$AUDIT" | json_field "['success']")
if [ "$AUDIT_SUCCESS" = "True" ]; then
    AUDIT_COUNT=$(echo "$AUDIT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data',[])))" 2>/dev/null || echo "0")
    if [ "$AUDIT_COUNT" -gt 0 ] 2>/dev/null; then
        log_success "GET /audit returns $AUDIT_COUNT audit entries"
    else
        log_fail "GET /audit returned 0 entries (expected > 0)"
    fi
else
    log_fail "GET /audit failed"
fi

# ============================================================
# STEP 12: User Management
# ============================================================
log_section "Step 12: User Management"

# List users.
USERS=$(api_get "/users")
USERS_SUCCESS=$(echo "$USERS" | json_field "['success']")
if [ "$USERS_SUCCESS" = "True" ]; then
    log_success "GET /users lists users"
else
    log_fail "GET /users failed"
fi

# Create invite.
INVITE=$(api_post "/users/invite" '{"email":"investor@test.com","role":"investor"}')
INVITE_TOKEN=$(echo "$INVITE" | json_field "['data']['token']")
if [ -n "$INVITE_TOKEN" ] && [ "$INVITE_TOKEN" != "None" ]; then
    log_success "POST /users/invite creates invite token"
else
    log_fail "POST /users/invite failed"
fi

# Register with invite.
if [ -n "$INVITE_TOKEN" ] && [ "$INVITE_TOKEN" != "None" ]; then
    REG_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "{\"token\":\"$INVITE_TOKEN\",\"name\":\"Test Investor\",\"password\":\"InvestorPass123\"}" \
        "${BASE_URL}/api/v1/auth/register")
    REG_SUCCESS=$(echo "$REG_RESP" | json_field "['success']")
    if [ "$REG_SUCCESS" = "True" ]; then
        log_success "POST /auth/register creates account via invite"
    else
        log_fail "POST /auth/register failed"
    fi

    # Verify new user can login.
    INV_LOGIN=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"email":"investor@test.com","password":"InvestorPass123"}' \
        "${BASE_URL}/api/v1/auth/login")
    INV_ROLE=$(echo "$INV_LOGIN" | json_field "['data']['user']['role']")
    if [ "$INV_ROLE" = "investor" ]; then
        log_success "New investor can login with correct role"
    else
        log_fail "Investor login returned unexpected role: $INV_ROLE"
    fi
fi

# ============================================================
# STEP 13: RBAC Enforcement
# ============================================================
log_section "Step 13: RBAC Enforcement"

# Login as investor.
if [ -n "$INVITE_TOKEN" ] && [ "$INVITE_TOKEN" != "None" ]; then
    INV_TOKEN=$(echo "$INV_LOGIN" | json_field "['data']['access_token']")

    # Investor should NOT be able to create categories (admin only).
    RBAC_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        -H "Authorization: Bearer $INV_TOKEN" -H "Content-Type: application/json" \
        -d '{"name":"Hacked","slug":"hacked"}' \
        "${BASE_URL}/api/v1/categories")
    if [ "$RBAC_STATUS" = "403" ]; then
        log_success "Investor cannot create categories (403)"
    else
        log_fail "RBAC: investor should get 403 on POST /categories, got $RBAC_STATUS"
    fi

    # Investor should NOT be able to list users (admin only).
    RBAC_USERS=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $INV_TOKEN" \
        "${BASE_URL}/api/v1/users")
    if [ "$RBAC_USERS" = "403" ]; then
        log_success "Investor cannot list users (403)"
    else
        log_fail "RBAC: investor should get 403 on GET /users, got $RBAC_USERS"
    fi

    # Investor should NOT be able to view audit log (admin only).
    RBAC_AUDIT=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $INV_TOKEN" \
        "${BASE_URL}/api/v1/audit")
    if [ "$RBAC_AUDIT" = "403" ]; then
        log_success "Investor cannot access audit log (403)"
    else
        log_fail "RBAC: investor should get 403 on GET /audit, got $RBAC_AUDIT"
    fi
fi

# ============================================================
# STEP 14: UI Pages
# ============================================================
log_section "Step 14: UI Pages"

for page in "/" "/login" "/documents" "/qa" "/analytics" "/admin/users" "/admin/branding" "/admin/watermark" "/admin/audit" "/admin/nda"; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}${page}")
    if [ "$STATUS" = "200" ]; then
        log_success "UI page ${page} returns 200"
    else
        log_fail "UI page ${page} returned $STATUS"
    fi
done

# ============================================================
# STEP 15: Security Headers
# ============================================================
log_section "Step 15: Security Headers"

HEADERS=$(curl -sI "${BASE_URL}/health")

for header in "X-Content-Type-Options" "X-Frame-Options" "Referrer-Policy" "Content-Security-Policy"; do
    if echo "$HEADERS" | grep -qi "$header"; then
        log_success "Security header: $header present"
    else
        log_fail "Security header: $header missing"
    fi
done

# ============================================================
# Summary
# ============================================================
print_summary
