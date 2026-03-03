# Plan: PR-13 — Single-Image Self-Hosted Docker Deployment

## Context

Currently the Go application (api + worker) runs outside Docker — only infrastructure
(MongoDB, Redis, Keycloak) is containerized. There is no Dockerfile, no `.dockerignore`,
and no image build pipeline. The goal is a single Docker image that bundles both `api`
and `worker` binaries, with a production-ready `docker-compose.prod.yml` for one-command
self-hosted deployment.

## Validation Commands

```bash
# Build image locally
docker build -t flowra:local .

# Run full stack
docker compose -f docker-compose.prod.yml up -d

# Smoke test
curl -sf http://localhost:8080/health

# Run existing tests (must still pass)
make test-unit
make lint
```

---

### Task 1: Merge worker into api binary (optional unified mode)

Add a `--with-worker` flag (or `FLOWRA_WORKER=true` env) to `cmd/api/main.go` that
starts worker goroutines (UserSync, Outbox, Repair) inside the api process.
This eliminates the need for a process supervisor in Docker.

- [x] Extract worker setup from `cmd/worker/main.go` into a reusable `internal/worker/runner.go` function `Run(ctx, cfg, mongoDB, redisCli)`
- [x] In `cmd/api/main.go`, check `FLOWRA_WORKER` env / `--with-worker` flag; if set, call `worker.Run()` in a goroutine before starting Echo
- [x] Ensure graceful shutdown tears down both api and worker goroutines
- [x] Keep `cmd/worker/main.go` working standalone (it calls the same `worker.Run()`)
- [x] Add unit test for the flag parsing / env check

### Task 2: Create Dockerfile

Multi-stage build: builder compiles Go binaries, runtime is a minimal image.

- [ ] Create `Dockerfile` at repo root
  - Stage 1 (`builder`): `golang:1.26-alpine`, copy `go.mod`/`go.sum`, `go mod download`, copy source, build `bin/api` with CGO_ENABLED=0
  - Stage 2 (`runtime`): `alpine:3.21`, install `ca-certificates tzdata`, copy `bin/api` from builder, copy `configs/config.yaml` to `/etc/flowra/config.yaml`
  - `EXPOSE 8080`, `ENV FLOWRA_WORKER=true`
  - `ENTRYPOINT ["/app/api"]`
- [ ] Create `.dockerignore` (exclude: `.git`, `bin/`, `tmp/`, `uploads/`, `tests/`, `*.md`, `.github/`, `docs/`)
- [ ] Verify `docker build -t flowra:local .` succeeds

### Task 3: Create docker-compose.prod.yml

A single-file compose for self-hosted deployment: MongoDB (replica set), Redis, Keycloak, and the Flowra app.

- [ ] Create `docker-compose.prod.yml` with services:
  - `mongodb` — `mongo:6.0`, replica set `rs0`, named volume, healthcheck
  - `mongo-init` — one-shot `rs.initiate()` (same as current dev compose)
  - `redis` — `redis:7-alpine`, named volume, healthcheck
  - `keycloak` — `quay.io/keycloak/keycloak:23.0`, imports realm, healthcheck
  - `app` — `build: .` (or `image: flowra:latest`), depends on all three, env vars pointing to container hostnames, volume for `uploads/`, healthcheck on `/health`
- [ ] Environment variables for `app` service: `MONGODB_URI`, `MONGODB_DATABASE`, `REDIS_ADDR`, `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_CLIENT_SECRET`, `KEYCLOAK_ADMIN_USERNAME`, `KEYCLOAK_ADMIN_PASSWORD`, `AUTH_JWT_SECRET`
- [ ] Add `.env.example` with placeholder values for all secrets
- [ ] Verify `docker compose -f docker-compose.prod.yml up -d` brings up the full stack

### Task 4: Keycloak realm auto-import on first start

Ensure Keycloak realm is configured automatically without manual setup scripts.

- [ ] Mount `configs/keycloak/realm-export.json` into Keycloak container with `--import-realm` (already done in dev compose — verify it works in prod compose)
- [ ] Add healthcheck for Keycloak readiness before app starts (depends_on + condition: service_healthy)
- [ ] Document that on subsequent starts the realm is not re-imported (Keycloak skips existing realms)

### Task 5: MongoDB replica set init reliability

The current `mongo-init` sidecar uses `sleep 5` then `rs.initiate()`. Make it more robust.

- [ ] Replace `sleep` with a retry loop that waits for `mongosh --eval "db.adminCommand('ping')"` to succeed
- [ ] Add healthcheck to `mongodb` service (`mongosh --eval "rs.status().ok"`)
- [ ] Ensure `app` service `depends_on` mongodb with `condition: service_healthy`

### Task 6: Makefile targets

- [ ] Add `make docker-build` — `docker build -t flowra:latest .`
- [ ] Add `make docker-prod-up` — `docker compose -f docker-compose.prod.yml up -d --build`
- [ ] Add `make docker-prod-down` — `docker compose -f docker-compose.prod.yml down`
- [ ] Add `make docker-prod-logs` — `docker compose -f docker-compose.prod.yml logs -f`

### Task 7: Update docs/DEPLOYMENT.md

- [ ] Add "Docker (self-hosted)" section with `docker compose -f docker-compose.prod.yml up -d` instructions
- [ ] Document environment variables (actual names without `FLOWRA_` prefix — fix the incorrect docs)
- [ ] Document volume mounts: `uploads/` for file persistence, MongoDB and Redis data volumes
- [ ] Document the `FLOWRA_WORKER=true` env for unified mode vs separate worker
