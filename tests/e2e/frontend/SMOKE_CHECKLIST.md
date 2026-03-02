# Board + Chat Sidebar Smoke Checklist

Use this checklist after switching branches in Chat=SoT refactor or after touching typed-entity write paths.

## Prerequisites

1. Start infra: `make docker-up`.
2. Reset local data: `make reset-data`.
3. Start full-stack runtime: `make dev` (runs infra + API + worker).
4. Login as `testuser / test123`.

## Smoke Flow

1. Open workspace chats page and create a new `task` chat with a title.
2. Confirm the new item appears in chat sidebar without duplicates.
3. Change status from sidebar action and verify:
   - sidebar badge/status changes;
   - a system message appears in chat timeline with updated status.
4. Change priority from sidebar and verify sidebar + timeline reflect the same value.
5. Assign/unassign user from sidebar and verify assignee is consistent in:
   - sidebar item;
   - task details panel (if opened);
   - board card.
6. Set due date from sidebar and verify date appears in sidebar + board.
7. Open board view and move card between columns.
8. Return to chat list and reopen the same chat:
   - status in sidebar matches board column;
   - no duplicate cards/items were created.

## Pass Criteria

1. No duplicated entities in sidebar or board after mutations.
2. Sidebar, chat timeline system messages, and board represent the same final state.
3. Status/priority/assignee/due date changes are visible after page reload.
