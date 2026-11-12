# Action Endpoints (UI / HTMX)

This document describes the HTTP action endpoints that change a chat or task field. The UI and agents use the
same endpoints.

OpenAPI coverage note: the `/actions/*` paths are documented in `docs/api/openapi.yaml`; this page is the narrative reference focused on UI/HTMX usage and implementation nuances.

## Scope

Direct task updates (`PUT /tasks/:task_id/{status,assignee,priority,due-date}`) run through the same service; see
[Task action endpoints](#task-action-endpoints) for how their answer differs.

- They apply the change to the chat inside the request and answer with its outcome (see below).
- After the change is applied, a human-readable line ("✅ … set due date to …") is posted to the thread. It is a
  record for people, not part of the outcome: if posting it fails, the answer is still "applied".
- They set an `Hx-Trigger` header for frontend refresh behavior.

## Base URL

- Base API prefix: `/api/v1`
- All action endpoints in this document are workspace-scoped:
  - `/api/v1/workspaces/:workspace_id/...`

## Authentication and Access

All endpoints require:

- Authentication (`Authorization: Bearer <token>` or valid session cookie for the HTMX frontend)
- Workspace access (membership or system admin, via workspace middleware)

## Request Binding (Important for HTMX)

Handlers bind both JSON and form payloads. For HTMX requests (`application/x-www-form-urlencoded`), the field names must match the `form` tags exactly.

Examples:

- `status`
- `priority`
- `assignee_id`
- `due_date`
- `title`

## Outcome Contract

Every answer is one of three outcomes. "200 and then not applied" does not happen.

| Outcome | HTTP | `error.code` | What the client does |
|---|---|---|---|
| Applied | `200` | — | Nothing. `data.version` is the chat version after the change |
| Applied earlier with this key | `200` | — | Nothing. `data.replayed` is `true`, nothing was written now |
| Not applied: validation or state | `400`, `403`, `404`, `422` | `INVALID_*`, `FORBIDDEN`, `NOT_FOUND`, `INVALID_STATE` | Fix the request |
| Not applied: concurrent writers | `409` | `CONCURRENT_MODIFICATION` | Retry (same key or none) |
| Not applied: key reused | `422` | `IDEMPOTENCY_KEY_REUSED` | Use a new key for a different request |
| Unknown: the commit failed midway | `503` | `OUTCOME_UNKNOWN` | Retry with the same `Idempotency-Key` |

Success body:

```json
{"success": true, "data": {"chat_id": "<uuid>", "version": 42, "replayed": false}}
```

- Concurrent changes of one chat queue up in the server and are retried with a reload of the chat a bounded number
  of times; a `409` means the change was not applied.
- `Idempotency-Key` (header, optional, up to 200 characters) makes a retry safe: a key already applied by the same
  user on the same chat returns the stored outcome instead of applying the change twice. Use a fresh UUID per
  intended change. Without a key the outcomes are the same, but after a `503` the client has to read the chat to
  learn whether the change landed.
- A key is recorded only when the change wrote something. If the chat already had the value, the key is not
  stored: a retry with it applies the value again, possibly over a change someone made in between, and reusing it
  for a different request is not rejected. This is a known limitation: the first request wrote nothing, so the
  retry behaves like a late request with the same value.
- After a `503` the server rebuilds the chat and task read models, so a read shows the change if the commit
  landed. Live updates (WebSocket, notifications) for that change may be missing.
- Only chat participants may change a chat (`403 FORBIDDEN` otherwise).

### Changes through tags in message text

A message such as `#status Done` posted through `POST .../messages` is answered when the message is saved; the
tag runs after the response and its failure shows only as a bot reply in the thread. That path has no outcome
contract. Agents that need to know whether a change landed use the action endpoints.

### Error responses

Errors use the standard API envelope:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_STATUS",
    "message": "status is required"
  }
}
```

## Endpoint Summary

### Chat action endpoints

These endpoints operate on a chat directly and create system messages for chat/task-like field changes.

| Method | Path | Body fields | Success | Trigger |
|---|---|---|---|---|
| POST | `/workspaces/:workspace_id/chats/:id/actions/status` | `status` (required) | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/priority` | `priority` (required) | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/assignee` | `assignee_id` (optional; empty clears) | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/due-date` | `due_date` (optional `YYYY-MM-DD`; empty clears) | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/close` | none | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/reopen` | none | `200` outcome | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/rename` | `title` (required) | `200` outcome | `chatUpdated` |

### Task action endpoints

These endpoints resolve `task_id -> chat_id` internally, then delegate to the same action service used by chat
actions. `PUT /tasks/:task_id/{status,assignee,priority,due-date}` does too: same outcomes and key, but its success
body is the task read back after the change.

| Method | Path | Body fields | Success | Trigger |
|---|---|---|---|---|
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/status` | `status` (required) | `200` outcome | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/priority` | `priority` (required) | `200` outcome | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/assignee` | `assignee_id` (optional; empty clears) | `200` outcome | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/due-date` | `due_date` (optional `YYYY-MM-DD`; empty clears) | `200` outcome | `taskUpdated` |

## Endpoint Details

### 1. Change chat status

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/status`
- Request fields:
  - `status` (string, required)
- Validation errors:
  - `INVALID_CHAT_ID`
  - `INVALID_REQUEST`
  - `INVALID_STATUS`

Example (JSON):

```bash
curl -X POST "http://localhost:8080/api/v1/workspaces/<workspace_id>/chats/<chat_id>/actions/status" \
  -H "Authorization: Bearer <token>" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"status":"In Progress"}' -i
```

### 2. Change chat priority

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/priority`
- Request fields:
  - `priority` (string, required)
- Validation errors:
  - `INVALID_CHAT_ID`
  - `INVALID_REQUEST`
  - `INVALID_PRIORITY`

### 3. Change chat assignee

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/assignee`
- Request fields:
  - `assignee_id` (UUID string, optional)
- Notes:
  - Empty `assignee_id` clears the assignee
- Validation errors:
  - `INVALID_CHAT_ID`
  - `INVALID_REQUEST`
  - `INVALID_ASSIGNEE_ID`

### 4. Set chat due date

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/due-date`
- Request fields:
  - `due_date` (date string `YYYY-MM-DD`, optional)
- Notes:
  - Empty `due_date` clears the due date
- Validation errors:
  - `INVALID_CHAT_ID`
  - `INVALID_REQUEST`
  - `INVALID_DATE`

### 5. Close chat

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/close`
- Request body: none
- Validation errors:
  - `INVALID_CHAT_ID`

### 6. Reopen chat

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/reopen`
- Request body: none
- Validation errors:
  - `INVALID_CHAT_ID`

### 7. Rename chat

- Endpoint: `POST /api/v1/workspaces/:workspace_id/chats/:id/actions/rename`
- Request fields:
  - `title` (string, required)
- Validation errors:
  - `INVALID_CHAT_ID`
  - `INVALID_REQUEST`
  - `INVALID_TITLE`

Example (form / HTMX-style):

```bash
curl -X POST "http://localhost:8080/api/v1/workspaces/<workspace_id>/chats/<chat_id>/actions/rename" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "title=New task discussion title" -i
```

### 8. Change task status (via action system)

- Endpoint: `POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/status`
- Request fields:
  - `status` (string, required)
- Success behavior:
  - Resolves task by `task_id`
  - Uses the task's `chat_id` internally
  - Returns `200` with the outcome + `Hx-Trigger: taskUpdated`
- Validation errors:
  - `INVALID_TASK_ID`
  - `INVALID_REQUEST`
  - `INVALID_STATUS`

### 9. Change task priority (via action system)

- Endpoint: `POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/priority`
- Request fields:
  - `priority` (string, required)
- Validation errors:
  - `INVALID_TASK_ID`
  - `INVALID_REQUEST`
  - `INVALID_PRIORITY`

### 10. Change task assignee (via action system)

- Endpoint: `POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/assignee`
- Request fields:
  - `assignee_id` (UUID string, optional)
- Notes:
  - Empty `assignee_id` clears the assignee
- Validation errors:
  - `INVALID_TASK_ID`
  - `INVALID_REQUEST`
  - `INVALID_ASSIGNEE_ID`

### 11. Set task due date (via action system)

- Endpoint: `POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/due-date`
- Request fields:
  - `due_date` (date string `YYYY-MM-DD`, optional)
- Notes:
  - Empty `due_date` clears the due date
- Validation errors:
  - `INVALID_TASK_ID`
  - `INVALID_REQUEST`
  - `INVALID_DATE`

Example (HTMX request from task sidebar)

```http
POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/due-date
Content-Type: application/x-www-form-urlencoded

due_date=2026-02-27
```

## Common Runtime Errors (from application/domain layer)

In addition to handler-level validation errors above, action handlers may return mapped application/domain errors, typically:

- `UNAUTHORIZED` (`401`)
- `FORBIDDEN` (`403`)
- `NOT_FOUND` (`404`)
- `CONCURRENT_MODIFICATION` (`409`) — not applied
- `INVALID_STATE` (`422`)
- `INVALID_TRANSITION` (`422`)
- `IDEMPOTENCY_KEY_REUSED` (`422`)
- `OUTCOME_UNKNOWN` (`503`) — retry with the same `Idempotency-Key`
- `INTERNAL_ERROR` (`500`)

## Frontend Integration Notes (HTMX)

Current UI usage patterns (examples in `web/templates/chat/task-sidebar.html` and `web/templates/task/sidebar.html`):

- `hx-post` with `name` attributes matching handler form fields
- `hx-trigger="change"` for selects/date inputs
- `hx-swap="none"`: the outcome body is for API clients, the UI ignores it
- UI refresh/event handling is triggered by `Hx-Trigger` (`chatUpdated` or `taskUpdated`)

Example:

```html
<select
  name="status"
  hx-post="/api/v1/workspaces/{{workspace_id}}/tasks/{{task_id}}/actions/status"
  hx-trigger="change"
  hx-swap="none">
</select>
```

## Source References (implementation)

- Routes: `cmd/api/routes.go`
- Chat action handler: `internal/handler/http/chat_action_handler.go`
- Task action handler: `internal/handler/http/task_action_handler.go`
- Action service: `internal/service/action_service.go`
- Per-chat runner, retries and keys: `internal/application/chat/command_runner.go`
- Response envelope: `internal/infrastructure/httpserver/response.go`
