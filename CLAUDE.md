# Claude AI Development Guide

**Go Version**: 1.26.1 | **Status**: All phases complete, CI green.

## Quick Start

```bash
# Backend (HTTP mode, local SQLite)
go build -o portal ./cmd && DD_TLS_MODE=none DD_DB_PATH=./dev.db ./portal

# Frontend (separate terminal, proxies API to :8080)
cd ui && npm install && npm run dev

# Run all Go tests
go test ./... -count=1

# Run UI tests
cd ui && npm test

# System tests (Docker-based, fresh data)
./scripts/testing/system-tests.sh
```

## Architecture

Single Docker image: Go API (Echo v4) serves REST endpoints at `/api/v1/*` and
pre-built SvelteKit static assets. SQLite (`modernc.org/sqlite`, pure Go) for
all data including document BLOBs. Browser -> Go Echo -> SQLite.

## Project Layout

```text
cmd/main.go                    Entry point, DI wiring, TLS, graceful shutdown
internal/
  handler/                     12 HTTP handlers (one per resource)
  service/                     Auth (JWT/bcrypt) + Email (SMTP)
  repository/                  10 SQLite repositories + migrate.go
  domain/                      Domain models + sentinel errors
  middleware/                   JWT auth, RBAC, audit, security headers, rate limit
pkg/                           envconfig, sanitize (CWE-117), response envelope
migrations/                    3 SQL files (schema, FTS5, seed categories)
ui/src/                        Svelte 5 + SvelteKit (adapter-static)
  lib/api/client.ts            Typed API client with JWT
  lib/stores/                  Svelte 5 rune-based stores (.svelte.ts)
  lib/theme/branding-engine.ts CSS custom property injection (white-label)
  routes/                      8 pages: login, documents, qa, analytics, admin/*
```

## Key Patterns (WitFoo Way)

- **Interfaces defined by consumers**, constructor injection, narrow (1-5 methods)
- **Error handling**: `%w` wrapping, sentinel errors, context-rich, log at handling point
- **Security**: All input hostile. `pkg/sanitize` for logs (CWE-117), CSS, SVG, filenames
- **Testing**: Table-driven `t.Run()`, testify, in-memory SQLite for repo tests
- **Air-gapped**: No CDN fonts, no external APIs. Self-hosted assets.
- **Disconnected-network**: CSP allows `'self'` + `'unsafe-inline'` (SvelteKit bootstrap)

## Documentation

| Doc | Contents |
| --- | --- |
| [docs/API.md](docs/API.md) | All REST endpoints by resource |
| [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) | All 20 environment variables |
| [docs/TESTING.md](docs/TESTING.md) | Test pyramid, scripts, patterns |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Docker, TLS modes, backup |

## Database

SQLite WAL mode. Schema in `internal/repository/migrations/`:

- `001_initial_schema.sql` — 16 tables (users, documents, document_versions, categories, access_grants, audit_log, qa_threads, qa_messages, nda_templates, nda_signatures, branding_config, branding_assets, view_events, watermark_config, notification_preferences, invite_tokens)
- `002_fts_indexes.sql` — FTS5 full-text search with sync triggers
- `003_seed_categories.sql` — 10 due diligence categories + singleton configs

Migrations are idempotent (`IF NOT EXISTS` / `INSERT OR IGNORE`), run on every boot.
