#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -z "${AUTH_OWNER_TOKEN_SHA256:-}" ]]; then
	echo "[dev] AUTH_OWNER_TOKEN_SHA256 is not set." >&2
	echo "[dev] generate a token:  token=\$(openssl rand -hex 32); echo \"\$token\"" >&2
	echo "[dev] and export its digest:  export AUTH_OWNER_TOKEN_SHA256=\$(printf %s \"\$token\" | sha256sum | cut -d' ' -f1)" >&2
	exit 1
fi
export CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"

echo "[dev] starting infrastructure (mongodb, redis)..."
docker-compose up -d mongodb redis mongo-init

echo "[dev] starting API with the worker inside..."
exec env FLOWRA_DEV_MODE=fullstack FLOWRA_WORKER=true go run ./cmd/api
