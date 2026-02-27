# PR-04: Rebuild Projections from `chat.*` Events Only

## Goal

Make read models derive exclusively from chat event streams and remove dependence on task event streams.

## Why This PR

After write unification, read model consistency requires projection ownership to be unified too.

## In Scope

- Implement/adjust projector to build `tasks_read_model` from typed chat state.
- Normalize event metadata contracts (aggregate type casing and filtering).
- Ensure board/task queries still work with existing API shape.

## Out of Scope

- Task package deletion.
- Frontend redesign.

## Planned Changes

- Add `ChatToTaskReadModelProjector` (or extend existing projector layer).
- Trigger projection updates on relevant `chat.*` events:
  - `chat.type_changed`
  - `chat.status_changed`
  - `chat.priority_set`
  - `chat.user_assigned` / `chat.assignee_removed`
  - `chat.due_date_set` / `chat.due_date_removed`
  - `chat.severity_set`
  - `chat.closed` / `chat.reopened`
- Normalize aggregate type checks to avoid case mismatch bugs.

## Implementation Steps

1. Define mapping `Chat -> tasks_read_model` fields.
2. Implement projector update pipeline and hook into event bus processing.
3. Add full rebuild command for `tasks_read_model` from `events` collection.
4. Update consistency checks and health checks.
5. Remove assumptions that `task.*` events exist.

## Definition of Done

- `tasks_read_model` remains correct with only `chat.*` writes.
- Rebuild process produces same results as incremental projection.
- Board and task detail pages remain functional.

## Test Plan

- Projector unit tests per chat event type.
- Integration test: replay event stream and validate read model snapshot.
- Health check test for projection consistency.

## Risks

- Risk: field mapping mismatch between chat and task views.
- Mitigation: snapshot tests for rendered task cards/sidebar data.

## Dependencies

- PR-03.
