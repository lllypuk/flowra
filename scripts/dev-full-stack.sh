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

if [[ "${FLOWRA_LOCAL:-}" == "1" ]]; then
	if [[ -z "${AUTH_OWNER_TOKEN_SHA256:-}" ]]; then
		echo "[dev] AUTH_OWNER_TOKEN_SHA256 is not set." >&2
		echo "[dev] generate a token:  token=\$(openssl rand -hex 32); echo \"\$token\"" >&2
		echo "[dev] and export its digest:  export AUTH_OWNER_TOKEN_SHA256=\$(printf %s \"\$token\" | sha256sum | cut -d' ' -f1)" >&2
		exit 1
	fi
	export CONFIG_PATH="${CONFIG_PATH:-configs/config.token.yaml}"
	echo "[dev] starting infrastructure (mongodb, redis), no keycloak..."
	docker-compose up -d mongodb redis mongo-init
else
	echo "[dev] starting infrastructure (mongodb, redis, keycloak)..."
	docker-compose --profile keycloak up -d mongodb redis keycloak mongo-init
fi

if [[ "${FLOWRA_LOCAL:-}" == "1" ]]; then
	echo "[dev] starting API with the worker inside..."
	FLOWRA_DEV_MODE=fullstack FLOWRA_WORKER=true go run ./cmd/api &
	api_pid=$!
	wait "$api_pid"
	exit $?
fi

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
