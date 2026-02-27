# PR-05: Move Task-Only Fields into Chat Domain

## Goal

Move remaining entity fields (especially attachments and detail attributes) from Task aggregate behavior into Chat domain events/state.

## Why This PR

If any mutable entity field stays writable only through Task aggregate, `Chat = SoT` is incomplete.

## In Scope

- Add chat domain events and behavior for attachments.
- Add missing typed entity detail fields to chat state if needed by UI.
- Update handlers/templates/services to read/write through chat-backed model.

## Out of Scope

- Full task package removal (next PR).

## Planned Changes

- Add `chat.attachment_added` and `chat.attachment_removed` events.
- Add chat aggregate methods for attachment mutation with validation.
- Update attachment routes to target chat identity (`chat_id` or `task_id==chat_id`).
- Keep response contracts stable for frontend.

## Implementation Steps

1. Extend chat domain event definitions and apply methods.
2. Wire new commands through tag/action or direct chat command handlers.
3. Update projection mapping for attachments into `tasks_read_model`.
4. Update task sidebar data assembly to source attachment state from chat projection.
5. Remove residual attachment writes to task aggregate.

## Definition of Done

- Attachments no longer depend on task aggregate write logic.
- Sidebar/board/task detail behavior unchanged from user perspective.
- Event streams for typed entities remain chat-only.

## Test Plan

- Unit tests for new chat attachment commands and event application.
- Integration tests for add/remove attachment flows.
- UI-level regression checks for sidebar attachment list refresh.

## Risks

- Risk: attachment metadata format mismatch.
- Mitigation: reuse existing file metadata schema and adapter layer.

## Dependencies

- PR-04.
