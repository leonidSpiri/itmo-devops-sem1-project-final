#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[prepare] downloading Go dependencies"
go mod download

if ! command -v psql >/dev/null 2>&1; then
  echo "[prepare] psql not found, installing postgresql-client"
  sudo apt-get update -y
  sudo apt-get install -y postgresql-client
fi

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-project-sem-1}"
POSTGRES_USER="${POSTGRES_USER:-validator}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-val1dat0r}"

export PGPASSWORD="$POSTGRES_PASSWORD"

echo "[prepare] waiting for postgres at ${POSTGRES_HOST}:${POSTGRES_PORT}"
for i in {1..30}; do
  if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" >/dev/null 2>&1; then
    break
  fi
  sleep 1
  if [[ $i -eq 30 ]]; then
    echo "[prepare] postgres is not ready" >&2
    exit 1
  fi
done

# Create DB if needed
if ! psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c '\q' >/dev/null 2>&1; then
  echo "[prepare] database ${POSTGRES_DB} not found or not accessible, trying to create"
  psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${POSTGRES_DB}\";" || true
fi

echo "[prepare] creating table prices (idempotent)"
psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS prices (
  id           BIGINT  NOT NULL,
  create_date  DATE    NOT NULL,
  name         TEXT    NOT NULL,
  category     TEXT    NOT NULL,
  price        NUMERIC NOT NULL
);
SQL

echo "[prepare] done"
