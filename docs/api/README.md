# Flowra API

Base path `/api/v1`, JSON. The current surface is the lead feed, owner questions and task actions; workspace and
chat routes are kept from the earlier product and not developed. The full route list is in
[openapi.yaml](./openapi.yaml).

| File | Content |
|---|---|
| [openapi.yaml](./openapi.yaml) | OpenAPI 3.1 specification |
| [feed-questions.md](./feed-questions.md) | Owner questions and answers, `lead-task` contract |
| [action-endpoints.md](./action-endpoints.md) | Chat and task actions, outcome contract |
| [websocket-protocol.md](./websocket-protocol.md) | WebSocket protocol |
| [postman_collection.json](./postman_collection.json) | Postman collection |

## Authentication

Every route except `/health`, `/ready`, `/health/details` and `/metrics` (outside `/api/v1`) needs the owner
token, which the server only checks against configured digests and never issues; setup is in
[DEPLOYMENT.md](../DEPLOYMENT.md#auth-configuration).

```bash
curl http://localhost:8080/api/v1/feed/status -H "Authorization: Bearer $token"
```

- API clients send `Authorization: Bearer <token>`. The web sign-in sets the `flowra_session` cookie, which the
  same middleware accepts when there is no `Authorization` header.
- A missing, malformed or unknown token is `401 UNAUTHORIZED`. `404` means the token passed and the resource
  does not exist.
- A cookie-authenticated write to the owner answer route, and a cookie-authenticated WebSocket handshake, must
  carry a matching `Origin` or `Referer` (`403 ORIGIN_MISMATCH`); Bearer requests skip the check.
- The token is never accepted in the URL, the WebSocket included.

## Responses

```json
{"success": true, "data": {}}
{"success": false, "error": {"code": "FEED_TASK_NOT_FOUND", "message": "feed task not found"}}
```

Handlers pick the code; branch on `error.code`, not on the message.

## Lead feed

Read projection of the lead's journal, republished by `leadfeed import`. Every response carries `revision` and
`as_of` of the snapshot it was read from (both empty before the first import).

| Method | Path | Result |
|---|---|---|
| GET | `/feed/status` | Projection health: `published`, `healthy`, counts, last import attempt |
| GET | `/feed/tasks` | Page of tasks; filters `status`, `waiting_on`; `limit`, `cursor` |
| GET | `/feed/tasks/{id}` | One task with its transitions |
| GET | `/feed/tasks/{id}/records` | Page of the task thread; `limit`, `cursor` |

Paging:

- `limit` defaults to 50; above 200 it is clamped to 200; zero or non-numeric is `400 INVALID_LIMIT`.
- `next_cursor` is bound to the revision it was issued in. If the feed was republished meanwhile, the next page
  is `409 FEED_REVISION_CHANGED`: restart without a cursor. A malformed cursor is `400 INVALID_CURSOR`.
- `status` in a task is the source value; `status_label` and `group` are for display only.

| Code | HTTP | When |
|---|---|---|
| `FEED_TASK_NOT_FOUND` | 404 | No such task in the snapshot |
| `FEED_REVISION_CHANGED` | 409 | Cursor from an older revision |
| `INVALID_CURSOR`, `INVALID_LIMIT` | 400 | Bad paging parameters |
| `FEED_UNAVAILABLE` | 503 | The snapshot could not be read; retry |

## Owner questions

`GET /feed/questions`, `GET /feed/questions/{qid}`, `POST /feed/questions/{qid}/answer` (requires
`Idempotency-Key`). States, codes and the delivery to the lead: [feed-questions.md](./feed-questions.md).

## Task actions

Changes of status, priority, assignee and due date answer with a defined outcome: applied, not applied, or
unknown (`503 OUTCOME_UNKNOWN`, retry with the same `Idempotency-Key`). Routes and the contract:
[action-endpoints.md](./action-endpoints.md).

## Workspaces, chats and the rest

Kept from the earlier product, not developed. Workspace-scoped routes need workspace membership; the list is in
[openapi.yaml](./openapi.yaml).

- `/auth/me`, `/users/*`, `/notifications/*`
- `/workspaces/{id}` with members, `/chats` (and `/actions/*`), `/chats/{chat_id}/messages`, `/messages/{id}`,
  `/tasks`
- file routes registered by the file handler
- `GET /ws`: WebSocket (cookie or Bearer, no token in the URL), see [websocket-protocol.md](./websocket-protocol.md)

## Development

```bash
make dev                              # infrastructure, worker and API
npx @redocly/cli lint openapi.yaml    # validate the specification
```
