#!/usr/bin/env bash
#set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-8080}"

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-project-sem-1}"
POSTGRES_USER="${POSTGRES_USER:-validator}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-val1dat0r}"

mkdir -p bin
go build -o bin/server .

# stop previous if exists
if [[ -f server.pid ]]; then
  old_pid="$(cat server.pid || true)"
  if [[ -n "${old_pid}" ]] && kill -0 "${old_pid}" >/dev/null 2>&1; then
    echo "[run] stopping previous server pid=${old_pid}"
    kill "${old_pid}" || true
    sleep 1
  fi
fi

export POSTGRES_HOST POSTGRES_PORT POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD PORT

echo "[run] starting server on :${PORT}"
nohup ./bin/server > server.log 2>&1 &
echo $! > server.pid

# wait for health
for i in {1..30}; do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    echo "[run] ready"
    echo "127.0.0.1"
    exit 0
  fi
  sleep 1
done

echo "[run] ERROR: server did not become ready; last logs:" >&2
tail -n 200 server.log || true
exit 1
