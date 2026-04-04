#!/usr/bin/env bash
# Start the Due Diligence Portal with self-signed TLS.
# Reads JWT secret from secret.txt (generate with: openssl rand -hex 32 > secret.txt)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRET_FILE="${SCRIPT_DIR}/secret.txt"

if [ ! -f "$SECRET_FILE" ]; then
    echo "Generating JWT secret..."
    openssl rand -hex 32 > "$SECRET_FILE"
    echo "Secret written to $SECRET_FILE"
fi

export DD_JWT_SECRET=$(cat "$SECRET_FILE" | tr -d '[:space:]')

docker compose -f "${SCRIPT_DIR}/docker/docker-compose.yml" up -d

echo ""
echo "Portal starting at https://localhost"
echo "Admin password (first run only):"
sleep 3
docker logs dd-portal 2>&1 | grep -i "admin password" || echo "  (already created — check earlier logs)"
