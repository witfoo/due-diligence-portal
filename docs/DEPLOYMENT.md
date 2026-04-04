# Deployment

## Quick Start

```bash
# Generate a JWT secret
export DD_JWT_SECRET=$(openssl rand -hex 32)

# Start with self-signed TLS (default)
docker compose -f docker/docker-compose.yml up -d

# Or HTTP-only behind a load balancer
docker compose -f docker/docker-compose.http.yml up -d

# Or with your own TLS certificate
mkdir -p certs && cp your-cert.crt certs/tls.crt && cp your-key.key certs/tls.key
docker compose -f docker/docker-compose.tls.yml up -d
```

The admin account is created on first boot. Check logs for the generated password:

```bash
docker logs dd-portal 2>&1 | grep "admin password"
```

Or set it explicitly: `DD_ADMIN_PASSWORD=your-password`

## TLS Modes

| Mode | Env Var | Ports | Use Case |
| --- | --- | --- | --- |
| `self-signed` (default) | `DD_TLS_MODE=self-signed` | 443 (HTTPS) + 80 (redirect) | Development, internal |
| `custom` | `DD_TLS_MODE=custom` | 443 (HTTPS) + 80 (redirect) | Production with your cert |
| `none` | `DD_TLS_MODE=none` | 80 (HTTP) | Behind load balancer |

## Data Persistence

All data is stored in a single SQLite file at `/data/portal.db` inside the container.
Mount a Docker volume to persist across container restarts:

```yaml
volumes:
  - portal-data:/data
```

The volume contains:

```
/data/
  portal.db       # SQLite database (all data including document BLOBs)
  portal.db-wal   # Write-ahead log
  portal.db-shm   # Shared memory
  certs/           # Self-signed TLS certs (only in self-signed mode)
```

## Docker Image

Published to GitHub Container Registry on every push to `main`:

```bash
docker pull ghcr.io/witfoo/due-diligence-portal:latest
```

Tags:
- `latest` -- latest main branch build
- `main` -- same as latest
- `v1.0.0` -- release versions (created via `git tag v1.0.0 && git push --tags`)

## Health Checks

```bash
curl http://localhost:8080/health    # {"status":"healthy"}
curl http://localhost:8080/ready     # {"status":"ready","checks":{"sqlite":"ok"}}
curl http://localhost:8080/version   # {"version":"...","commit":"..."}
```

## Email Configuration

To enable email notifications (invites, Q&A alerts):

```yaml
environment:
  DD_SMTP_ENABLED: "true"
  DD_SMTP_HOST: smtp.example.com
  DD_SMTP_PORT: "587"
  DD_SMTP_USERNAME: user@example.com
  DD_SMTP_PASSWORD: your-password
  DD_SMTP_FROM: portal@example.com
```

## Backup

SQLite backup while the portal is running:

```bash
docker exec dd-portal sqlite3 /data/portal.db ".backup /data/backup.db"
docker cp dd-portal:/data/backup.db ./backup-$(date +%Y%m%d).db
```
