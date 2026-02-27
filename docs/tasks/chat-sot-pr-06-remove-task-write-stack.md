# PR-06: Remove Task Aggregate Write Stack

## Goal

Delete obsolete Task write stack and leave only chat-based write path plus query-compatible read access.

## Why This PR

Keeping dead write code increases maintenance burden and can reintroduce drift later.

## In Scope

- Remove task write aggregates/usecases/repository write wiring.
- Remove event serializer registrations for `task.*` business events.
- Keep or replace query DTO/service contracts required by handlers.

## Out of Scope

- Functional changes to user-visible workflows (already done in prior PRs).

## Planned Changes

- Remove or archive:
  - `internal/domain/task` write aggregate logic
  - `internal/application/task` write usecases
  - task write repository code paths in DI
- Retain query-facing models via dedicated package (can stay in `application/task` temporarily).
- Remove dead eventbus handler branches for `task.*` writes.

## Implementation Steps

1. Identify all imports that create task write commands/usecases.
2. Replace with chat command adapters where still needed.
3. Delete unused files and update container wiring.
4. Simplify interfaces to query-only where applicable.
5. Run full compile and linter cleanup.

## Definition of Done

- No code path emits new `task.*` domain write events.
- Build passes without task write stack.
- Handler interfaces reflect actual architecture.

## Test Plan

- Full unit test run.
- Integration tests for main entity workflows.
- Static grep checks for forbidden symbols (`NewCreateTaskUseCase`, `task.EventType...` writes).

## Risks

- Risk: hidden coupling in tests and mocks.
- Mitigation: staged delete with compile-first checkpoints.

## Dependencies

- PR-05.
