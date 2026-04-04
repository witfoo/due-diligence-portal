# Claude AI Development Guide for Due Diligence Portal

**Last Updated**: April 4, 2026
**Branch**: main
**Go Version**: 1.26.1
**Status**: Phase 1 (Foundation) complete. Building toward v1.0.

## Architecture

Single Docker image containing Go API (Echo v4) + Svelte 5 (SvelteKit) frontend.
SQLite (modernc.org/sqlite, pure Go) for all data including document BLOBs.
No fault tolerance needed — single instance deployment.

### Key Patterns (WitFoo Way)

- **Go service layout**: `cmd/`, `internal/{handler,service,repository,domain,middleware}`, `pkg/`
- **Interface-driven design**: Interfaces defined by consumers, constructor injection
- **Error handling**: `%w` wrapping, sentinel errors, context-rich messages, log at handling point
- **Security**: All input is hostile. CWE-117 log injection prevention via `pkg/sanitize`
- **Testing**: Table-driven with `t.Run()`, testify assert/require, 75% coverage target
- **Air-gapped**: No CDN fonts, no external APIs. All assets self-hosted.

### API Flow

Browser → Go Echo server (8080/8443) → SQLite

The Go server serves both `/api/v1/*` REST endpoints and the pre-built SvelteKit
static assets (via filesystem serving from `ui/build/`).

### Database

SQLite with WAL mode. All documents stored as BLOBs in `document_versions` table.
Schema: `internal/repository/migrations/*.sql` (embedded via `embed.FS`).
Migrations are idempotent (IF NOT EXISTS / INSERT OR IGNORE).

### TLS Modes

| Mode | Env Var | Description |
| --- | --- | --- |
| `self-signed` | `DD_TLS_MODE=self-signed` | Auto-generate ECDSA cert at startup |
| `custom` | `DD_TLS_MODE=custom` | Read cert from `/certs/` volume mount |
| `none` | `DD_TLS_MODE=none` | HTTP only (behind load balancer) |

### Frontend

Svelte 5 + SvelteKit with `adapter-static`. Built to `ui/build/`, served by Go.
Carbon Design System for components. WitFoo branding engine for white-label.
Rune-based stores (`.svelte.ts`). API client at `ui/src/lib/api/client.ts`.

## Development Tools

### Internet Research

```bash
cd utils/internet-research && go build -o internet-research .
./internet-research --query "topic" --deep --json --max-results 5
```

### Quick Start

```bash
# Backend
go build -o portal ./cmd && DD_TLS_MODE=none DD_DB_PATH=./dev.db ./portal

# Frontend (separate terminal)
cd ui && npm run dev
```

### Testing

```bash
go test ./...              # Go unit tests
cd ui && npm test          # Vitest
cd ui && npm run check     # svelte-check
```

## Environment Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DD_JWT_SECRET` | Yes | — | HS256 signing key (min 32 chars) |
| `DD_DB_PATH` | No | `/data/portal.db` | SQLite database path |
| `DD_TLS_MODE` | No | `self-signed` | TLS mode |
| `DD_HTTP_PORT` | No | `8080` | HTTP port |
| `DD_HTTPS_PORT` | No | `8443` | HTTPS port |
| `DD_UI_PATH` | No | `ui/build` | UI static files path |
| `DD_MAX_UPLOAD_SIZE` | No | `104857600` | Max upload size (bytes) |
| `DD_ADMIN_EMAIL` | No | `admin@localhost` | Initial admin email |
| `DD_ADMIN_PASSWORD` | No | random | Initial admin password |
| `DD_SMTP_ENABLED` | No | `false` | Enable email notifications |
| `DD_SMTP_HOST` | No | — | SMTP server hostname |
| `DD_SMTP_PORT` | No | `587` | SMTP port |
| `DD_SMTP_FROM` | No | `noreply@example.com` | Sender address |

## Implementation Phases

- [x] Phase 1: Foundation (skeleton, health checks, TLS, Docker)
- [x] Phase 2: Auth + Users (JWT, RBAC, invites, email integration)
- [x] Phase 3: Documents + Categories (upload, FTS5, versioning, upload size limits)
- [x] Phase 4: Permissions + NDA (access grants, NDA gates, signing)
- [x] Phase 5: Q&A + Audit + Analytics (threads, immutable audit, engagement)
- [x] Phase 6: Branding + Watermark (white-label, watermark config)
- [x] Phase 7: Testing + CI/CD (185 Go tests, GitHub Actions, linter configs)
