#!/bin/bash
# Entry point for the Railway deployment image: runs DB migrations once, then
# starts the API server and the background worker as sibling processes in the
# same container. Railway only lets you run one process per service, but this
# app is split into "api" and "worker" binaries (see main.go) — so we launch
# both here and treat the container as unhealthy the moment either one exits,
# instead of silently keeping a half-dead container alive.
set -euo pipefail

# Railway injects PORT; the app itself reads APP_PORT. Map one to the other
# without requiring an app code change.
export APP_PORT="${PORT:-${APP_PORT:-8080}}"
export APP_HOST="${APP_HOST:-0.0.0.0}"

echo "[entrypoint] running migrations..."
/app/crm migrate up

echo "[entrypoint] starting worker..."
/app/crm worker &
worker_pid=$!

echo "[entrypoint] starting api on ${APP_HOST}:${APP_PORT}..."
/app/crm api &
api_pid=$!

shutdown() {
  echo "[entrypoint] shutting down..."
  kill -TERM "$worker_pid" "$api_pid" 2>/dev/null || true
  wait "$worker_pid" "$api_pid" 2>/dev/null || true
}
trap shutdown TERM INT

# If either process exits (crash or normal), stop the other and exit with
# that process's status so Railway sees the container as failed and restarts it.
wait -n "$worker_pid" "$api_pid"
exit_code=$?

kill -TERM "$worker_pid" "$api_pid" 2>/dev/null || true
wait "$worker_pid" "$api_pid" 2>/dev/null || true

exit "$exit_code"
