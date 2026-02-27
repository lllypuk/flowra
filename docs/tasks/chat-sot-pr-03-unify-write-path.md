# PR-03: Unify All Writes Through Chat Command Pipeline

## Goal

Ensure every entity state mutation goes through a single command pipeline rooted in `Chat`.

## Why This PR

As long as both direct task write endpoints and chat command endpoints exist independently, consistency is fragile and reasoning remains difficult.

## In Scope

- Standardize write entrypoint to chat action/tag command flow.
- Adapt legacy direct task write endpoints to same pipeline (compat adapters) or remove them.
- Fix `close/reopen` to execute real domain transitions.

## Out of Scope

- Removal of task query DTOs.
- Full package deletions.

## Planned Changes

- Keep `TaskActionHandler` as canonical task mutation HTTP surface.
- For `PUT /tasks/:id/status|priority|assignee|due-date`:
  - either remove routes
  - or convert handlers into adapters that delegate to `ActionService`/chat pipeline.
- Change `ActionService.Close/Reopen` to submit executable commands (`#close`, `#reopen`) or call chat usecase directly and then post system message.
- Remove `ChatUseCases.CreateTask` dependency from tag executor contract.

## Implementation Steps

1. Enumerate all write routes and map each to chat command equivalent.
2. Switch direct task write handlers to delegation adapters.
3. Update `ActionService` close/reopen behavior.
4. Add idempotency handling for repeated requests.
5. Remove unused wiring to task write usecases.

## Definition of Done

- Every write mutation leads to `chat.*` event emission only.
- `close/reopen` modifies aggregate state, not just message text.
- No endpoint bypasses chat pipeline.

## Test Plan

- Unit tests for each route adapter.
- Integration test matrix for status/priority/assignee/due date/close/reopen.
- Concurrency tests for optimistic locking retries.

## Risks

- Risk: hidden UI path still calls old direct endpoint.
- Mitigation: route audit + log warnings on deprecated handlers.

## Dependencies

- PR-02.
