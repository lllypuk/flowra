# PR-10: Fix Nullable Field Cleanup in Chat Read Model

## Goal

Ensure nullable typed-chat fields are correctly removed in chat read model when values are cleared.

## Why This PR

Current read-model update uses `$set` only. When assignee or due date is removed, stale fields can remain in `chats_read_model`, causing inconsistent UI state after reload.

## In Scope

- Update chat read-model write logic to use `$unset` for removed nullable fields.
- Cover `assigned_to`, `due_date`, and bug-only optional fields where applicable.
- Add tests for set -> clear transitions.

## Out of Scope

- Projection architecture changes.
- Board query logic changes unrelated to nullable cleanup.

## Planned Changes

- Refactor `MongoChatRepository.updateReadModel` to build both:
  - `$set` for present values.
  - `$unset` for absent values that may exist from prior state.
- Add targeted tests for:
  - assign -> unassign.
  - due-date set -> due-date clear.
  - bug severity set -> severity clear (if supported by domain path).

## Implementation Steps

1. Introduce helper to compose update document with `$set` and `$unset`.
2. Ensure typed/non-typed chat transitions clean up stale task fields.
3. Add repository tests for transition scenarios.
4. Validate sidebar and chat header state after reload in manual check.

## Definition of Done

- Clearing assignee and due date removes stale values from chat read model document.
- Reloaded UI reflects cleared state consistently.
- No regression for fields that remain set.

## Test Plan

- Unit tests for update document generation.
- Repository integration tests against MongoDB.
- Manual verification in chat sidebar with reload.

## Risks

- Risk: overly broad `$unset` may remove valid data.
- Mitigation: explicit field whitelist and transition-focused tests.

## Dependencies

- None.
