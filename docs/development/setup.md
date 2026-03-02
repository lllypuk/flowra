# Development Environment Setup Guide

## Overview

This guide describes the current local development setup for Flowra after the Chat=SoT refactor hardening.

## Required Tools

- Go 1.26+
- Docker + Docker Compose
- Make
- Git

Optional but recommended:
- `golangci-lint`
- `mongosh`
- `redis-cli`
- Playwright browser binaries (`make playwright-install`)

## Quick Start

```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
make deps
make dev
```

`make dev` starts a full local runtime:
- MongoDB, Redis, Keycloak (via docker-compose)
- worker
- API server

Health check:

```bash
curl http://localhost:8080/health
```

## Runtime Modes

### Full-stack mode (default)

```bash
make dev
```

Use this mode for normal development, smoke checks, and frontend E2E.

### API-only mode (limited)

```bash
make dev-lite
# or
FLOWRA_DEV_MODE=lite go run ./cmd/api
```

Use this only when worker/outbox processing is not needed.

## Chat=SoT Data Reset

When switching branches around Chat=SoT changes, reset local data to avoid stale read-model shapes:

```bash
make docker-up
make reset-data
make dev
```

`make reset-data` drops and recreates Chat=SoT local collections (`events`, `chats_read_model`, `tasks_read_model`, `outbox`, `repair_queue`) and recreates indexes.

## Common Make Targets

```bash
make help
make build
make lint
make test
make test-unit
make test-integration
make test-e2e
make test-e2e-frontend
make test-e2e-frontend-smoke
make test-load-tags
make test-coverage
make docker-up
make docker-down
make docker-logs
```

## Frontend E2E Setup

Install Playwright Chromium once:

```bash
make playwright-install
```

Run frontend tests (server must be running):

```bash
make test-e2e-frontend
make test-e2e-frontend-smoke
```

## Troubleshooting

### Legacy read-model warning on startup

If logs mention non-empty legacy collections (`chat_read_model`, `task_read_model`), run:

```bash
make reset-data
```

### Frontend E2E tests are skipped

Ensure the app is reachable at `http://localhost:8080` and Keycloak is running.

### Outbox-related stale UI updates in local run

Use `make dev` instead of `make dev-lite` so worker processing is active.
