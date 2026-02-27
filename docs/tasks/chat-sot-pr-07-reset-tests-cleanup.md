# PR-07: Data Reset Tooling, Test Hardening, and Final Cleanup

## Goal

Finalize `Chat = SoT` transition with deterministic local reset workflow, robust regression tests, and documentation cleanup.

## Why This PR

Since there is no production data, we should optimize developer feedback and ensure every environment can be reset to the new model safely.

## In Scope

- Add data reset command/script for local/dev/test.
- Expand integration/e2e regression suite for chat-sourced entity model.
- Remove stale docs and references to task write ownership.

## Out of Scope

- New product features.

## Planned Changes

- Add `make reset-data` target and supporting script:
  - drop `events`, `chat_read_model`, `task_read_model`, `outbox`, `repair_queue`
  - recreate indexes
- Add anti-regression tests:
  - no duplicate entity creation
  - all writes emit `chat.*`
  - board/task/chat views remain consistent
  - close/reopen real state transition
- Update docs (`CLAUDE.md`, architecture notes, README fragments if needed).

## Implementation Steps

1. Implement reset script and make target.
2. Add CI/local instructions for using reset when switching branches.
3. Add integration tests for full typed entity lifecycle.
4. Add e2e smoke checklist for board + chat sidebar.
5. Remove stale documentation and TODOs that contradict Chat ownership.

## Definition of Done

- One command resets local data to clean post-refactor model.
- Test suite covers critical chat-sourced entity scenarios.
- Documentation no longer references `Task = SoT` behavior.

## Test Plan

- Run `make lint`.
- Run `make test`.
- Run `make test-integration`.
- Run `make test-e2e` and `make test-e2e-frontend` when environment available.

## Risks

- Risk: reset tool accidentally used in wrong environment.
- Mitigation: hard guard in script to allow only local/dev configs.

## Dependencies

- PR-06.
