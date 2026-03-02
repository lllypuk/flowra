# PR-08: Force Task Projection Sync for Chat-Driven Mutations

## Goal

Guarantee that board data is updated when typed chat entities are created or changed via chat-driven flows.

## Why This PR

The smoke path creates and mutates task chats from chat UI routes. Those writes currently rely on async projection through event delivery, which can lag or be absent in local dev and causes board to show `0 tasks`.
Backward compatibility is not required, so we can simplify write-to-projection behavior aggressively.

## In Scope

- Add explicit `tasks_read_model` sync after typed chat creation from chat template flows.
- Add explicit `tasks_read_model` sync after status/priority/assignee/due-date actions from sidebar/chat action endpoints.
- Remove redundant logic if it conflicts with deterministic projection sync.

## Out of Scope

- Full redesign of projection architecture.
- Removal of event bus projection handlers.

## Planned Changes

- Introduce projector dependency into chat-driven handlers/services where write completes:
  - `ChatTemplateHandler.ChatCreate` for typed chats.
  - `ActionService` call sites used by chat sidebar actions.
- Call `taskProjector.RebuildOne(chatID)` after successful write command.
- Add structured logs for sync failures and return appropriate HTTP error when sync is required by UI contract.

## Implementation Steps

1. Add a small interface adapter for task projection sync (`RebuildOne` only).
2. Wire adapter in container for handlers using chat-driven writes.
3. Update chat create flow to sync projection immediately for `task/bug/epic`.
4. Update action flow (`#status`, `#priority`, `#assignee`, `#due`) to sync projection after command success.
5. Add unit tests for each path asserting `RebuildOne` is called with chat ID.
6. Add integration test: create typed chat in chats UI path -> board returns card in same workspace.

## Definition of Done

- Creating typed chat from chats page produces board card without needing worker start.
- Sidebar changes are reflected on board after reload.
- Existing event-driven projection path remains operational.

## Test Plan

- Unit tests for handler/service wiring and projector invocation.
- Integration test for chat-create -> board visibility.
- Manual smoke re-run using `tests/e2e/frontend/SMOKE_CHECKLIST.md`.

## Risks

- Risk: additional sync call may increase write latency.
- Mitigation: sync only for typed chat flows and keep operation scoped to single `chatID`.

## Dependencies

- None.
