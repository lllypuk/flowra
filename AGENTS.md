# Repository Guidelines

## Project Structure & Module Organization
`flowra` is a Go monorepo. Entry points live in `cmd/` (`cmd/api`, `cmd/worker`, and tooling in `cmd/tools`). Core code is under `internal/` with layered packages: `application/`, `domain/`, `infrastructure/`, `handler/`, `middleware/`, `service/`, and `worker/`. Frontend templates and static assets are in `web/` (`web/templates`, `web/static/css`, `web/static/js`). Tests are split by scope in `tests/` (`integration/`, `e2e/`, `e2e/frontend/`, `load/`, `mocks/`, `testutil/`). Runtime/config files are in `configs/` and `docker-compose.yml`.

## Context Requirements
When starting work on this repository, include `CLAUDE.md` in the working context along with the relevant code/files for the task.

## Build, Test, and Development Commands
Use `make` targets (see `Makefile`) for common workflows:

- `make dev` - run the API locally (`go run ./cmd/api`)
- `make build` - build `bin/api` and `bin/worker`
- `make test` - full test suite with race detector and coverage profile
- `make test-unit` - fast unit tests in `internal/...`
- `make test-integration` - integration tests (`-tags=integration`, testcontainers)
- `make test-e2e` / `make test-e2e-frontend` - end-to-end tests (`-tags=e2e`)
- `make test-load-tags` - manual k6 load test for tag-heavy message flows (requires `k6` and `AUTH_TOKEN`)
- `make playwright-install` - install Chromium for frontend E2E
- `make lint` - format + lint (`go fmt`, `golangci-lint --fix`)
- `make docker-up` / `make docker-down` - local services (MongoDB, Redis, Keycloak)

## Coding Style & Naming Conventions
Target Go `1.26` (`go.mod`). Follow standard Go formatting (`gofmt`) and imports (`goimports` via `golangci-lint`); keep lines under the configured `golines` limit (120). Use package-focused names, exported identifiers in `CamelCase`, unexported in `camelCase`, and files in `snake_case.go`. Keep tests in `*_test.go`. Prefer small, cohesive packages under the existing layer structure rather than cross-layer shortcuts.

## Testing Guidelines
Unit tests sit beside code in `internal/.../*_test.go`; broader scenarios live in `tests/integration` and `tests/e2e`; manual load tests live in `tests/load` (k6). Use build tags (`integration`, `e2e`) consistently. Test utilities belong in `tests/testutil`; reusable fakes/mocks in `tests/mocks`. Integration tests rely on Docker/testcontainers (MongoDB is started as a single-node replica set for transaction support), so ensure Docker is running. Load tests are not part of `make test`/CI by default and require explicit auth setup (`AUTH_TOKEN`). After development, you must run `make lint` and `make test` before opening a PR.

## Commit & Pull Request Guidelines
Recent history mixes Conventional Commit style (`feat:`, `docs:`) with direct fixes. Prefer: `<type>: short imperative summary` (for example, `fix: handle websocket reconnect timeout`). Keep commits focused. PRs should include a clear description, linked issue/task, test evidence (commands run), and screenshots/GIFs for UI changes in `web/`.
