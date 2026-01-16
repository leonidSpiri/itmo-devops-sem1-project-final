#!/usr/bin/env bash

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[prepare] go mod download"
go mod download

if ! command -v psql >/dev/null 2>&1 || ! command -v pg_isready >/dev/null 2>&1; then
  echo "[prepare] installing postgresql-client"
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
    echo "[prepare] ERROR: postgres not ready" >&2
    exit 1
  fi
done

echo "[prepare] ensuring table prices exists"
psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS prices (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(255) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  create_date TIMESTAMP NOT NULL
);
SQL

echo "[prepare] done"
