# PR-13: Self-Hosted Single-Image Docker Deployment

## Goal

Move deployment to a self-hosted model based on one Docker image that contains all app binaries and can run required roles via startup mode.

## Why This PR

Current operational model assumes separate API/worker processes. Target state is simpler self-hosting with one image artifact, easier distribution, and predictable runtime setup.

## In Scope

- Build a single production image containing both `api` and `worker` binaries.
- Add role-based startup contract (for example `FLOWRA_ROLE=api|worker|all`).
- Provide self-hosted deployment docs for running one image in Docker Compose.
- Keep external dependencies (MongoDB/Redis/Keycloak) as separate services.

## Out of Scope

- Converting external infra dependencies into the same container.
- Kubernetes-specific deployment manifests (optional later).

## Planned Changes

- Create/adjust multi-stage Dockerfile to produce one artifact with:
  - `bin/api`
  - `bin/worker`
  - entrypoint script selecting role.
- Add compose profile for self-hosted deployment using the same image for API and worker roles.
- Add health/readiness behavior per role.
- Update deployment docs for single-image workflow.

## Implementation Steps

1. Design container runtime contract (`FLOWRA_ROLE`, env validation, defaults).
2. Implement unified Dockerfile and entrypoint.
3. Add compose examples:
   - one image, two services (`api`, `worker`) with different role env.
   - optional `all-in-one` mode for local/self-hosted non-prod usage.
4. Update docs with build/push/run steps and rollback flow.
5. Add CI check that image builds and both roles start successfully.

## Definition of Done

- Single Docker image can start API and worker roles without rebuild.
- Self-hosted docs allow running full app stack from clean host.
- Existing functionality (event processing, projections, websocket, auth) works under new image model.

## Test Plan

- Build image locally and run API role.
- Run worker role from same image and verify outbox processing.
- Run smoke checklist in self-hosted compose setup.
- Validate graceful shutdown and restart behavior for each role.

## Risks

- Risk: role entrypoint complexity and misconfiguration.
- Mitigation: strict startup validation + clear logs and fail-fast on invalid role.

## Dependencies

- PR-11 (dev/full-stack runtime contract) is recommended first.
