# Testing

## Test Pyramid

| Level | Count | Tools | What It Tests |
| --- | --- | --- | --- |
| Unit | 205 | testify (Go), Vitest (UI) | Functions with mocked deps |
| System | 60 | curl + Docker | Full API validation against running container |
| E2E | 20 | Playwright | Browser UI rendering and navigation |

## Running Tests

```bash
# Go unit tests
go test ./... -count=1

# Go unit tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# UI unit tests
cd ui && npm test

# Race detection
go test -race -short ./...

# System tests (builds Docker image, starts fresh container, validates all endpoints)
./scripts/testing/system-tests.sh

# E2E tests (builds Docker image, runs Playwright)
./scripts/testing/e2e-tests.sh

# Full pipeline
./scripts/testing/full-testing.sh
```

## Test Scripts

| Script | What It Does |
| --- | --- |
| `scripts/testing/linter.sh` | gofmt, goimports, go vet, golangci-lint, svelte-check, eslint |
| `scripts/testing/unit-tests.sh` | Go unit tests + Vitest |
| `scripts/testing/system-tests.sh` | Docker-based API validation (15 steps, 60 checks) |
| `scripts/testing/e2e-tests.sh` | Playwright against Docker container |
| `scripts/testing/security.sh` | govulncheck, npm audit, race detection |
| `scripts/testing/full-testing.sh` | Orchestrates all above, supports `--from-step` |

## System Tests

The system test starts a **fresh Docker container with no volumes** on each run.
It validates every API endpoint through 15 steps:

1. Health endpoints
2. Authentication (login, wrong password, refresh, 401)
3. Categories (10 seeded + create)
4. Documents (upload, list, detail, download, search, versioning)
5. Permissions (grant, list)
6. Q&A (create thread, post message, change status)
7. NDA (create template, check status, sign, verify)
8. Analytics (record view, dashboard)
9. Branding (get, update, reset)
10. Watermark (get, update, reset)
11. Audit log (verify entries logged)
12. User management (list, invite, register)
13. RBAC enforcement (investor blocked from admin endpoints)
14. UI pages (all 10 routes return 200)
15. Security headers (CSP, X-Frame-Options, etc.)

## Playwright E2E

Three test projects matching the WitFoo Analytics pattern:

- **chromium** -- Authenticated admin user (storageState)
- **chromium-no-auth** -- Public pages and health endpoints
- **chromium-role-auth** -- RBAC tests with different roles

Configuration: `ui/playwright.config.ts`

## Go Test Patterns

Table-driven tests with `t.Run()`, testify `assert`/`require`:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "hello", "HELLO", false},
        {"empty", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            assert.Equal(t, tt.want, got)
        })
    }
}
```

Repository tests use in-memory SQLite (`":memory:"`) with `setupTestDB(t)` helper.
Handler tests use `httptest.NewServer` with real JWT authentication.
