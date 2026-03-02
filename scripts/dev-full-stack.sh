#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

cleanup() {
	local exit_code=$?
	trap - EXIT INT TERM

	if [[ -n "${api_pid:-}" ]] && kill -0 "$api_pid" 2>/dev/null; then
		kill "$api_pid" 2>/dev/null || true
	fi
	if [[ -n "${worker_pid:-}" ]] && kill -0 "$worker_pid" 2>/dev/null; then
		kill "$worker_pid" 2>/dev/null || true
	fi

	if [[ -n "${api_pid:-}" ]]; then
		wait "$api_pid" 2>/dev/null || true
	fi
	if [[ -n "${worker_pid:-}" ]]; then
		wait "$worker_pid" 2>/dev/null || true
	fi

	exit "$exit_code"
}
trap cleanup EXIT INT TERM

echo "[dev] starting infrastructure (mongodb, redis, keycloak)..."
docker-compose up -d mongodb redis keycloak mongo-init

echo "[dev] starting worker..."
FLOWRA_DEV_MODE=fullstack go run ./cmd/worker &
worker_pid=$!

echo "[dev] starting API..."
FLOWRA_DEV_MODE=fullstack go run ./cmd/api &
api_pid=$!

wait -n "$api_pid" "$worker_pid"
status=$?
if [[ $status -ne 0 ]]; then
	echo "[dev] one of the processes exited with status $status"
fi

exit "$status"
