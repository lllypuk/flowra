# WebSocket Protocol (Real-time API)

This document describes the current WebSocket protocol used by Flowra for real-time updates.

It covers:

- connection/authentication
- client-to-server messages (`subscribe`, `unsubscribe`, `chat.typing`, `ping`)
- server-to-client messages (acks, errors, presence, typing, domain event broadcasts)
- payload naming conventions and frontend parsing caveats
- reconnection and re-subscription behavior

## Endpoint

- Canonical endpoint: `GET /api/v1/ws`

Notes:

- The route is registered under the authenticated API group (`/api/v1/ws`).
- `/api/v1/ws` is the only WebSocket path; `/ws` is not registered.

## Authentication

The WebSocket endpoint requires authentication.

Server-side behavior (`internal/handler/websocket/handler.go`):

- The route is behind the auth middleware: the browser sends its `flowra_session` cookie, other clients send
  `Authorization: Bearer <token>` on the handshake. The query string is never read for credentials: owner tokens do
  not expire and would end up in browser history, proxy logs and `Referer`.
- A cookie-authenticated handshake must carry an `Origin` (or `Referer`) equal to the request `Host`; otherwise the
  server returns `403` `ORIGIN_MISMATCH`. A handshake with an `Authorization` header skips this check, so non-browser
  clients need no `Origin`. The check is `middleware.RequireSameOriginForCookie`, shared with the feed's write routes.

If authentication fails, the server returns `401` JSON (no upgrade).

Example:

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required"
  }
}
```

## Connection Lifecycle

1. Client connects to `/api/v1/ws`
2. Server authenticates request
3. Server upgrades HTTP connection to WebSocket
4. Server registers a new client in the hub
5. Client sends `subscribe` messages for chat rooms it wants to receive events for

Important:

- Subscriptions are connection-local (stored in the server client instance).
- After reconnect, the client must subscribe again.

## Client -> Server Messages

Client messages are JSON objects with this shape:

```json
{
  "type": "subscribe",
  "chat_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Schema (logical):

- `type` (string, required)
- `chat_id` (UUID string, required for `subscribe`, `unsubscribe`, `chat.typing`)

### Supported client message types

#### `subscribe`

Subscribes the current WebSocket connection to a chat room.

```json
{"type":"subscribe","chat_id":"<chat_uuid>"}
```

Server response (ack):

```json
{"type":"ack","action":"subscribed","chat_id":"<chat_uuid>"}
```

#### `unsubscribe`

Unsubscribes the current connection from a chat room.

```json
{"type":"unsubscribe","chat_id":"<chat_uuid>"}
```

Server response (ack):

```json
{"type":"ack","action":"unsubscribed","chat_id":"<chat_uuid>"}
```

#### `chat.typing`

Broadcasts a typing indicator to other subscribers of the chat.

```json
{"type":"chat.typing","chat_id":"<chat_uuid>"}
```

#### `ping`

Application-level ping message (separate from WebSocket transport ping frames).

```json
{"type":"ping"}
```

Server response:

```json
{"type":"pong"}
```

### Error responses to invalid client messages

For invalid JSON or unsupported message types, the server sends:

```json
{"type":"error","message":"..."}
```

Examples:

- `invalid message format`
- `chat_id is required for subscribe`
- `unknown message type: <type>`

## Server -> Client Messages

There are two broad categories:

1. Hub/control messages (`ack`, `error`, `pong`, `presence.changed`, `chat.typing`)
2. Domain event broadcasts (for example `chat.message.posted`, `chat.closed`, `task.updated`, `notification.new`)

### Hub/control message formats

#### Ack

```json
{"type":"ack","action":"subscribed","chat_id":"<chat_uuid>"}
```

#### Error

```json
{"type":"error","message":"chat_id is required for subscribe"}
```

#### Pong

```json
{"type":"pong"}
```

#### Presence change

```json
{
  "type": "presence.changed",
  "user_id": "<user_uuid>",
  "is_online": true
}
```

#### Typing indicator broadcast

```json
{
  "type": "chat.typing",
  "chat_id": "<chat_uuid>",
  "user_id": "<user_uuid>"
}
```

Note:

- The server typing payload currently includes `user_id`, not `username`.

### Domain event broadcast envelope

Most real-time updates from the event bus are sent using an envelope like:

```json
{
  "type": "chat.message.posted",
  "chat_id": "<chat_uuid>",
  "data": {
    "aggregate_id": "<event_aggregate_uuid>",
    "ChatID": "<chat_uuid>",
    "Content": "Hello"
  }
}
```

Envelope fields:

- `type` (string): WebSocket event type
- `chat_id` (string, optional): top-level chat ID for chat-routed events
- `data` (object or JSON value, optional): event payload (often raw serialized domain event payload)

## Event Type Mapping (Domain -> WebSocket)

The broadcaster maps domain events to WebSocket event types in `internal/infrastructure/websocket/broadcaster.go`.

Common mappings:

- `message.created` -> `chat.message.posted`
- `message.edited` -> `chat.message.edited`
- `message.deleted` -> `chat.message.deleted`
- `chat.status_changed` -> `chat.status_changed`
- `chat.renamed` -> `chat.renamed`
- `chat.priority_set` -> `chat.priority_set`
- `chat.user_assigned` -> `chat.user_assigned`
- `chat.assignee_removed` -> `chat.assignee_removed`
- `chat.due_date_set` -> `chat.due_date_set`
- `chat.due_date_removed` -> `chat.due_date_removed`
- `chat.closed` -> `chat.closed`
- `chat.reopened` -> `chat.reopened`
- `task.created` -> `task.created`
- `task.updated` -> `task.updated`
- `task.status_changed` -> `task.updated`
- `task.assigned` -> `task.updated`
- `notification.created` -> `notification.new`

Routing behavior:

- Chat events are broadcast to subscribers of that chat room.
- `notification.new` is user-specific and sent to the target user's active connections.

## Payload Naming Conventions (Important)

WebSocket `data` payloads may use mixed naming conventions:

- `snake_case` fields (processed payloads / helper payloads)
- `PascalCase` fields (raw serialized Go domain event payloads)

This is expected in the current implementation.

Frontend code should handle both forms when reading key fields (for example `chat_id` vs `ChatID`).

Example defensive parsing:

```javascript
document.body.addEventListener("chat.message.posted", function (evt) {
  var msg = evt.detail || {};
  var chatId = msg.ChatID || msg.chat_id;
  var messageId = msg.aggregate_id || msg.message_id;
  // ...
});
```

## Frontend Integration (HTMX + JS)

### HTMX WebSocket connection

Typical pattern:

```html
<div hx-ext="ws" ws-connect="/api/v1/ws">
  ...
</div>
```

### Parsing incoming messages

Current frontend pattern (`web/static/js/chat.js`):

- Listen to `htmx:wsAfterMessage`
- Parse `evt.detail.message`
- Dispatch `CustomEvent(msg.type, { detail: ... })` on `document.body`

This lets page-specific code listen to events like:

- `chat.message.posted`
- `chat.message.edited`
- `chat.message.deleted`
- `chat.typing`
- `presence.changed`
- `notification.new`

### HTMX v2 socket access caveat

HTMX v2 stores the WebSocket socket differently than older patterns.

Preferred (HTMX v2):

```javascript
var el = document.querySelector('[hx-ext*="ws"]');
var internalData = el['htmx-internal-data'];
var wsWrapper = internalData && internalData.webSocket;
var socket = wsWrapper && wsWrapper.socket;
```

Legacy pattern still exists in some code (`__htmx_ws`) and may not work consistently with HTMX v2.

## Reconnection Behavior

The frontend (`web/static/js/app.js`) implements reconnection with:

- exponential backoff
- jitter
- max attempts
- manual retry via UI status indicator

It listens to HTMX WebSocket events:

- `htmx:wsOpen`
- `htmx:wsError`
- `htmx:wsClose`

and triggers reconnect via:

- `htmx:wsReconnect`

### Re-subscription after reconnect

Because subscriptions are per connection, clients should re-send `subscribe` for active chats after reconnect.

## End-to-End Example (Chat View)

1. Connect:

```text
GET /api/v1/ws
Authorization: Bearer <token>
```

2. Subscribe to a chat:

```json
{"type":"subscribe","chat_id":"<chat_uuid>"}
```

3. Receive ack:

```json
{"type":"ack","action":"subscribed","chat_id":"<chat_uuid>"}
```

4. Another user sends a message, server broadcasts:

```json
{
  "type": "chat.message.posted",
  "chat_id": "<chat_uuid>",
  "data": {
    "aggregate_id": "<message_uuid>",
    "ChatID": "<chat_uuid>",
    "AuthorID": "<user_uuid>",
    "Content": "Hello team"
  }
}
```

5. Frontend parses and dispatches `chat.message.posted`, then refreshes/appends message UI.

## Troubleshooting

### `401` before upgrade

- Check the session cookie or the `Authorization: Bearer` header; `?token=` is ignored
- Use canonical endpoint `/api/v1/ws`

### `403` `ORIGIN_MISMATCH` before upgrade

- A cookie-authenticated client sent no `Origin`/`Referer`, or one that differs from the `Host` (for example, behind
  a proxy that rewrites `Host`); send `Authorization: Bearer` instead of the cookie

### Connected but no chat events

- Confirm the client sent `{"type":"subscribe","chat_id":"..."}` after connection
- Re-subscribe after reconnect
- Verify the event is chat-routed (user notifications are sent as `notification.new`)

### Payload fields not found in frontend code

- Handle both `snake_case` and `PascalCase` field names in event payloads

## Source References (implementation)

- Route registration: `cmd/api/routes.go`
- WS HTTP handler: `internal/handler/websocket/handler.go`; Origin check: `internal/middleware/same_origin.go`
- Hub/client message handling: `internal/infrastructure/websocket/client.go`
- Presence/typing broadcasts: `internal/infrastructure/websocket/hub.go`
- Domain event -> WS mapping: `internal/infrastructure/websocket/broadcaster.go`
- Frontend parsing/reconnect logic: `web/static/js/chat.js`, `web/static/js/app.js`
- Chat subscription example: `web/templates/chat/view.html`
