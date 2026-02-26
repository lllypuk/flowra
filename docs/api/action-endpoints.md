# Action Endpoints (UI / HTMX)

This document describes the HTTP action endpoints used by the UI for chat/task field changes that are routed through the message-based action system.

OpenAPI coverage note: the `/actions/*` paths are documented in `docs/api/openapi.yaml`; this page is the narrative reference focused on UI/HTMX usage and implementation nuances.

## Scope

These endpoints are different from direct task update endpoints (for example `PUT /tasks/:task_id/status`):

- They are designed for UI actions (especially HTMX forms).
- They create system messages / action events instead of directly mutating fields in place.
- They return an empty success response plus an `Hx-Trigger` header for frontend refresh behavior.

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

## Response Format

### Success responses (action endpoints)

Action endpoints return an empty body on success:

- Chat action endpoints: `200 OK` + `Hx-Trigger: chatUpdated`
- Task action endpoints: `204 No Content` + `Hx-Trigger: taskUpdated`

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
| POST | `/workspaces/:workspace_id/chats/:id/actions/status` | `status` (required) | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/priority` | `priority` (required) | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/assignee` | `assignee_id` (optional; empty clears) | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/due-date` | `due_date` (optional `YYYY-MM-DD`; empty clears) | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/close` | none | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/reopen` | none | `200` empty | `chatUpdated` |
| POST | `/workspaces/:workspace_id/chats/:id/actions/rename` | `title` (required) | `200` empty | `chatUpdated` |

### Task action endpoints

These endpoints resolve `task_id -> chat_id` internally, then delegate to the same action service used by chat actions.

| Method | Path | Body fields | Success | Trigger |
|---|---|---|---|---|
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/status` | `status` (required) | `204` empty | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/priority` | `priority` (required) | `204` empty | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/assignee` | `assignee_id` (optional; empty clears) | `204` empty | `taskUpdated` |
| POST | `/workspaces/:workspace_id/tasks/:task_id/actions/due-date` | `due_date` (optional `YYYY-MM-DD`; empty clears) | `204` empty | `taskUpdated` |

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
  - Returns `204 No Content` + `Hx-Trigger: taskUpdated`
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
- `CONCURRENT_MODIFICATION` (`409`)
- `INVALID_STATE` (`422`)
- `INVALID_TRANSITION` (`422`)
- `INTERNAL_ERROR` (`500`)

## Frontend Integration Notes (HTMX)

Current UI usage patterns (examples in `web/templates/chat/task-sidebar.html` and `web/templates/task/sidebar.html`):

- `hx-post` with `name` attributes matching handler form fields
- `hx-trigger="change"` for selects/date inputs
- `hx-swap="none"` because the server returns no body
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
- Response envelope: `internal/infrastructure/httpserver/response.go`
