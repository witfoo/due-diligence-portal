# Environment Variables

All configuration is via environment variables. Every variable has a sensible default —
the portal starts with zero required env vars in development.

## Required for Production

| Variable | Default | Description |
| --- | --- | --- |
| `DD_JWT_SECRET` | `dev-secret-...` | HS256 JWT signing key. Generate with `openssl rand -hex 32`. **Must be set in production.** |

## Server

| Variable | Default | Description |
| --- | --- | --- |
| `DD_HTTP_PORT` | `8080` | HTTP port (health checks, redirect in TLS modes) |
| `DD_HTTPS_PORT` | `8443` | HTTPS port (used in `self-signed` and `custom` TLS modes) |
| `DD_TLS_MODE` | `self-signed` | TLS mode: `self-signed`, `custom`, or `none` |
| `DD_TLS_CERT_PATH` | `/certs/tls.crt` | Custom TLS certificate path (only used when `DD_TLS_MODE=custom`) |
| `DD_TLS_KEY_PATH` | `/certs/tls.key` | Custom TLS key path (only used when `DD_TLS_MODE=custom`) |
| `DD_CERT_DIR` | `/data/certs` | Directory for auto-generated self-signed certs |
| `DD_RATE_LIMIT` | `200` | Max API requests per minute per IP (static assets excluded) |

## Database

| Variable | Default | Description |
| --- | --- | --- |
| `DD_DB_PATH` | `/data/portal.db` | SQLite database file path. Use `:memory:` for testing. |

## Application

| Variable | Default | Description |
| --- | --- | --- |
| `DD_UI_PATH` | `ui/build` | Path to pre-built SvelteKit static assets |
| `DD_MAX_UPLOAD_SIZE` | `104857600` | Maximum file upload size in bytes (default 100MB) |
| `DD_ADMIN_EMAIL` | `admin@localhost` | Initial admin account email (created on first boot) |
| `DD_ADMIN_PASSWORD` | (random) | Initial admin password. If unset, a random password is printed to logs. |

## Email (SMTP)

| Variable | Default | Description |
| --- | --- | --- |
| `DD_SMTP_ENABLED` | `false` | Enable email notifications |
| `DD_SMTP_HOST` | (none) | SMTP server hostname (required if enabled) |
| `DD_SMTP_PORT` | `587` | SMTP port |
| `DD_SMTP_USERNAME` | (none) | SMTP authentication username |
| `DD_SMTP_PASSWORD` | (none) | SMTP authentication password |
| `DD_SMTP_FROM` | `noreply@example.com` | Sender email address |
| `DD_SMTP_TLS` | `true` | Use STARTTLS for SMTP connection |
