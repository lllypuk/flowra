# PR-09: Unify Read-Model Collection Names

## Goal

Eliminate collection name drift by standardizing on one naming scheme for chat/task read models.

## Why This PR

Code currently mixes singular and plural collection names (`task_read_model` vs `tasks_read_model`, `chat_read_model` vs `chats_read_model`). This causes reset/rebuild/repair tooling and runtime queries to operate on different collections.
Since backward compatibility is not required, we can adopt a single canonical naming scheme and remove legacy variants directly.

## In Scope

- Select canonical names for read-model collections (recommended: plural to match runtime usage).
- Update all app code, tooling, worker code, and tests to use the same constants.
- Ensure reset/rebuild/index creation affect the same collections used by runtime handlers.

## Out of Scope

- Schema changes beyond naming unification.

## Planned Changes

- Replace hardcoded collection strings with shared constants everywhere.
- Align constants in `internal/infrastructure/mongodb/indexes.go` with runtime usage.
- Remove legacy collection-name paths from runtime/tooling code.
- Update:
  - API container wiring.
  - worker startup/projection wiring.
  - reset and rebuild tools.
  - repository tests and integration fixtures.

## Implementation Steps

1. Define canonical read-model collection constants.
2. Refactor all direct string usages to constants.
3. Update reset tool to drop canonical collections.
4. Update index registration for canonical collections.
5. Remove legacy singular/plural variants in code and tests (no compatibility layer).
6. Add tests ensuring one consistent collection path is used end-to-end.
7. Verify with local DB that only canonical collections are read/written by new runs.

## Definition of Done

- No runtime/tooling path writes to alternate collection names.
- `make reset-data` clears exactly the collections used by API + worker.
- Board/task/chat read paths work against same canonical sources.

## Test Plan

- Unit tests for index constants and repository collection usage.
- Integration test for reset + create chat + board query.
- Manual DB verification: canonical collections populated, legacy names untouched.

## Risks

- Risk: missing a leftover hardcoded string in low-traffic code path.
- Mitigation: exhaustive `rg` check in CI for legacy names.

## Dependencies

- PR-08 can proceed independently, but this PR should be merged before final smoke sign-off.
