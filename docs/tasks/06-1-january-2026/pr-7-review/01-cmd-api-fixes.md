# PR #7 — Fix plan: `cmd/api` DI container + routing readiness/health

**Scope:** only issues around `cmd/api/*` wiring and readiness/health handling found during review of PR #7 (`lllypuk/flowra`).  
**Goal:** make `cmd/api` safe to merge by ensuring it boots a **real** application by default, and health endpoints are implemented correctly and consistently.

---

## 1) DI Container: remove implicit mocks from production wiring

### Problem statement
`cmd/api/container.go` currently builds HTTP handlers and middleware dependencies using mock implementations unconditionally (auth/workspace/chat services, access checker, etc.). That makes the entrypoint non-production-ready and can mask real integration failures.

### Desired end state
- `cmd/api` builds real dependencies by default.
- Mock-based wiring is only possible **explicitly** (config flag) and cannot activate by accident.
- Startup logs clearly state which wiring mode is active.

### Implementation options (choose one)

#### Option A (recommended): explicit wiring mode in config
Add explicit config field:
- `app.mode: real|mock` (or `server.mode`, but `app` is clearer)

Rules:
- Default: `real`
- `mock` allowed only in development environments (or guarded by explicit warning log)

**Tasks**
1. Extend config schema (the app already has `internal/config/config.go`):
   - Add `AppConfig` with `Mode` field, or add `ServerConfig.Mode`.
   - Add env override, e.g. `APP_MODE`.
   - Validate allowed values.
2. Update `cmd/api/main.go` startup log:
   - Log `app_mode` and refuse to start in `mock` if `cfg.IsProduction()`.
3. Split handler wiring into two functions:
   - `setupHTTPHandlersReal()`
   - `setupHTTPHandlersMock()`
4. Ensure `setupRepositories()` and `setupUseCases()` are actually used by handlers in `real` wiring.

**Acceptance criteria**
- `go run cmd/api/main.go` uses real wiring by default.
- Setting `APP_MODE=mock` switches to mock wiring and logs a clear warning.
- In production mode (however determined), `APP_MODE=mock` causes a hard failure.

#### Option B: separate entrypoint for demo/mock
Create `cmd/api-demo/main.go` (or `cmd/demo-api/main.go`) and keep `cmd/api` real-only.

**Tasks**
1. Remove mock wiring from `cmd/api/container.go`.
2. Move mock wiring into `cmd/api-demo/container.go` (or similar).
3. Document both in README/DEV docs.

**Acceptance criteria**
- `cmd/api` never uses mocks.
- Demo behavior is available via separate command.

---

## 2) DI Container: create “real” HTTP handlers (bridge to application layer)

### Problem statement
Even if repositories/use cases exist, `cmd/api` currently instantiates mock handlers/services. The container should instead wire:
- repositories (MongoDB)
- use cases (application layer)
- HTTP handlers (interface layer) consuming those use cases via **consumer-side interfaces**

### Desired end state
- `internal/handler/http/*` constructors accept interfaces/use cases required for real behavior.
- `cmd/api/container.go` provides those dependencies from `setupRepositories()` and `setupUseCases()`.

### Tasks (high-level)
1. Define minimal service interfaces on the handler side (consumer-side) where needed (already partially done in several handlers).
2. Provide real implementations:
   - either adapt application use cases directly (preferred)
   - or add thin application services wrapping multiple use cases
3. Update `setupUseCases()` to build all needed use cases (not only notifications).
4. Update `setupHTTPHandlersReal()` to create:
   - `AuthHandler`, `WorkspaceHandler`, `ChatHandler`, `MessageHandler`, `TaskHandler`, `NotificationHandler`, `UserHandler`
   - `WSHandler` (websocket handler)
5. Remove or hard-gate placeholder endpoints (see section 4).

**Acceptance criteria**
- No mock constructors are invoked in real mode.
- All declared routes point to initialized handlers (no nil placeholders).
- Integration tests (if present) exercise real code paths.

---

## 3) Readiness/health: remove `AcquireContext()` pattern and use request context properly

### Problem statement
Readiness callback currently derives context using `Echo.AcquireContext().Request().Context()`, which is unsafe and can be disconnected from the active request.

### Desired end state
- Health endpoints use `echo.Context` from the current request.
- `IsReady(ctx)` and `GetHealthStatus(ctx)` are called with `ctx.Request().Context()`.

### Tasks
1. Change router health registration API to accept a function that can receive request context.
   - Prefer: `func(ctx context.Context) bool` or `func(c echo.Context) bool`
2. Implement `/ready` handler to call:
   - `c.IsReady(ctx.Request().Context())`
3. Avoid any usage of “borrowed” context from Echo internals for readiness/health.

**Acceptance criteria**
- No `AcquireContext()` usage for readiness.
- Readiness respects request cancellation/deadline where relevant.

---

## 4) Health endpoints: eliminate duplication and unify contract

### Problem statement
There are multiple sets of constants and potentially multiple implementations:
- `Container` has `healthStatusHealthy/unhealthy/degraded`
- `routes.go` has `statusHealthy/unhealthy/ready/not ready`
- there is both `router.RegisterHealthEndpoints` and `SetupHealthEndpoints`

### Desired end state
A single consistent contract, minimal and stable:

- `GET /health` (liveness):
  - Always `200`
  - Body: `{"status":"healthy"}`
- `GET /ready` (readiness):
  - `200` if ready, else `503`
  - Body:
    ```json
    {
      "status": "ready|not_ready",
      "components": [
        {"name":"mongodb","status":"healthy|unhealthy|degraded","message":"..."},
        {"name":"redis","status":"..."},
        {"name":"websocket_hub","status":"..."},
        {"name":"eventbus","status":"..."}
      ]
    }
    ```
- Optional `GET /health/details`:
  - `200` or `503` based on aggregated status (documented)

### Tasks
1. Decide one mechanism:
   - Keep `router.RegisterHealthEndpoints(...)` and delete `SetupHealthEndpoints(...)`, **or**
   - Remove `RegisterHealthEndpoints` and use explicit handlers only.
2. Consolidate constants:
   - Define `healthy/unhealthy/degraded/ready/not_ready` in a single place.
3. Ensure OpenAPI and README match actual behavior.

**Acceptance criteria**
- Only one implementation path exists.
- Status strings and payload shape are identical everywhere.

---

## 5) “Placeholder endpoints”: remove or explicitly gate

### Problem statement
Some route registration functions create placeholder handlers returning `501 Not Implemented` when handler is nil. This can be useful during early development, but becomes technical debt and can hide wiring mistakes.

### Desired end state
- In real mode: no placeholder endpoints.
- If placeholders are desired for mock/demo mode: they are allowed only there and clearly labeled.

### Tasks
1. In real mode:
   - Treat missing handler as fatal during startup (`NewContainer` returns error).
2. In mock/demo mode:
   - Either provide full mock handlers, or register only the endpoints that are actually supported.
3. Add a startup validation step, e.g. `c.ValidateWiring()` that checks all required fields are non-nil.

**Acceptance criteria**
- Real mode fails fast when wiring is incomplete.
- No `501` placeholders exist in real mode.

---

## 6) Verification checklist (definition of done)

### Manual checks
- Start dependencies (MongoDB, Redis).
- Run `go run cmd/api/main.go`.
- Verify:
  - `GET /health` returns 200 with correct JSON.
  - `GET /ready` returns 200 only if MongoDB/Redis are reachable and hub state is correct.
  - Core API endpoints do not return `501`.

### Automated checks
- Unit tests:
  - Health endpoints return consistent payload and status codes.
  - `Container` wiring validation fails on missing dependencies in real mode.
- If E2E exists:
  - Ensure `make test-e2e` (or equivalent) passes using real wiring.

---

## Notes / non-goals
- This document intentionally does not propose reorganizing the entire PR into smaller PRs (though it’s recommended operationally).
- This document does not cover documentation updates, OpenAPI quality, or other subsystem reviews (eventbus/websocket/etc.) beyond what impacts `cmd/api` wiring and health endpoints.