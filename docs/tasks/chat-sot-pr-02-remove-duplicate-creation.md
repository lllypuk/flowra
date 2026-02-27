# PR-02: Remove Duplicate Entity Creation Paths

## Goal

Guarantee single creation path for typed entities and remove duplicate task creation side effects.

## Why This PR

Current architecture has overlapping creation flows (sync and async), which can create duplicate logical entities and dirty event streams.

## In Scope

- Disable async task creation from chat type change handler.
- Remove explicit second creation call in board flow.
- Keep behavior stable: creating typed chat still creates a usable typed entity view.

## Out of Scope

- Full projection rewrite.
- Removal of all task packages.

## Planned Changes

- Stop registering `TaskCreationHandler` in event bus wiring.
- Remove dead wiring from container setup.
- In board create flow, create only typed chat; do not call second `CreateTask` command.
- Add temporary bootstrap behavior if needed:
  - when typed chat is created, upsert `tasks_read_model` from chat state using `task_id = chat_id`.

## Implementation Steps

1. Remove handler registration from eventbus registry wiring.
2. Remove container dependency on `TaskCreationHandler`.
3. Update board handler create flow to call only chat creation.
4. Ensure UI still resolves task sidebar for new typed chat.
5. Add regression tests for "one create request -> one logical entity".

## Definition of Done

- No runtime code path creates typed entity twice.
- No `TaskCreationHandler` subscription remains active.
- Board create action works with one backend command path.

## Test Plan

- Unit tests for container wiring and route flow.
- Integration test: create typed chat from board and assert one entity record.
- Integration test: no duplicate writes after concurrent retries.

## Risks

- Risk: board expects immediate task read model data.
- Mitigation: temporary typed-chat bootstrap into `tasks_read_model` until PR-04.

## Dependencies

- PR-01 accepted.
