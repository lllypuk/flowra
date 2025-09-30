# API Contracts — HTTP REST + WebSocket

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

API системы состоит из двух частей:
- **REST API** — CRUD операции, запросы данных
- **WebSocket API** — real-time уведомления, обновления

## Общие принципы

### Base URL

```
Production:  https://api.teamsup.com
Development: http://localhost:8080
```

### Versioning

**URL prefix:** `/api/v1`

```
GET /api/v1/chats
GET /api/v1/workspaces
```

### Authentication

**Bearer Token (JWT):**

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

Все endpoints (кроме `/auth/*` и `/invite/*`) требуют авторизации.

### Content Type

```http
Content-Type: application/json
Accept: application/json
```

### Pagination

**Cursor-based pagination:**

```
GET /api/v1/chats?limit=20&cursor=eyJpZCI6ImNoYXQtdXVpZCIsInRzIjoxNjk...
```

**Response:**

```json
{
  "data": [...],
  "pagination": {
    "nextCursor": "eyJpZCI6...",
    "hasMore": true,
    "total": 150
  }
}
```

### Filtering и Sorting

**Query Parameters:**

```
?status=In Progress          - фильтр по одному значению
?type=task,bug               - фильтр по нескольким (comma-separated)
?assignee=user-uuid          - фильтр по assignee
?sort=-createdAt             - сортировка (- = desc, + или без = asc)
?search=authentication       - полнотекстовый поиск
?limit=50                    - количество результатов
?cursor=...                  - курсор для пагинации
```

### Error Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": {
      "status": "Must be one of: To Do, In Progress, Done",
      "assignee": "User not found"
    }
  }
}
```

**Error Codes:**

| Code | HTTP Status | Описание |
|------|-------------|----------|
| `UNAUTHORIZED` | 401 | Не авторизован |
| `FORBIDDEN` | 403 | Нет прав доступа |
| `NOT_FOUND` | 404 | Ресурс не найден |
| `VALIDATION_ERROR` | 400 | Ошибка валидации |
| `CONFLICT` | 409 | Конфликт (например, duplicate) |
| `RATE_LIMIT_EXCEEDED` | 429 | Превышен rate limit |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка сервера |

---

## Authentication Endpoints

### POST /auth/login

**Описание:** Инициирует OAuth 2.0 flow с Keycloak

**Request:**

```http
POST /api/v1/auth/login HTTP/1.1
Content-Type: application/json

{
  "redirectUrl": "http://localhost:3000/dashboard"
}
```

**Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "authUrl": "http://localhost:8090/realms/teams-up/protocol/openid-connect/auth?client_id=teams-up-app&redirect_uri=http://localhost:8080/auth/callback&response_type=code&state=random-state"
}
```

**Frontend action:** Redirect user to `authUrl`

---

### GET /auth/callback

**Описание:** OAuth callback от Keycloak

**Request:**

```http
GET /api/v1/auth/callback?code=AUTH_CODE&state=random-state HTTP/1.1
```

**Response:**

```http
HTTP/1.1 302 Found
Location: http://localhost:3000/dashboard
Set-Cookie: session_id=session-uuid; HttpOnly; Secure; SameSite=Strict; Max-Age=86400
```

**Backend действия:**
1. Обменивает code на токены (Keycloak)
2. Создаёт/обновляет User в БД
3. Создаёт session в Redis
4. Устанавливает session cookie
5. Редиректит на frontend

---

### POST /auth/logout

**Описание:** Выход из системы

**Request:**

```http
POST /api/v1/auth/logout HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
Set-Cookie: session_id=; Max-Age=0
```

**Backend действия:**
1. Удаляет session из Redis
2. Очищает session cookie
3. Опционально: отзывает токен в Keycloak

---

### GET /auth/me

**Описание:** Информация о текущем пользователе

**Request:**

```http
GET /api/v1/auth/me HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "user-uuid",
  "username": "alice",
  "email": "alice@example.com",
  "displayName": "Alice Smith",
  "avatarUrl": "https://cdn.teamsup.com/avatars/alice.jpg",
  "isSystemAdmin": false,
  "workspaces": [
    {
      "id": "workspace-uuid-1",
      "name": "Engineering Team",
      "role": "admin"
    },
    {
      "id": "workspace-uuid-2",
      "name": "Marketing Team",
      "role": "member"
    }
  ],
  "currentWorkspaceId": "workspace-uuid-1"
}
```

---

## Workspace Endpoints

### GET /workspaces

**Описание:** Список workspaces пользователя

**Request:**

```http
GET /api/v1/workspaces HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "data": [
    {
      "id": "workspace-uuid-1",
      "name": "Engineering Team",
      "role": "admin",
      "memberCount": 15,
      "unreadChats": 3,
      "createdAt": "2025-09-01T10:00:00Z",
      "createdBy": {
        "id": "user-uuid",
        "username": "alice",
        "displayName": "Alice Smith"
      }
    },
    {
      "id": "workspace-uuid-2",
      "name": "Marketing Team",
      "role": "member",
      "memberCount": 8,
      "unreadChats": 0,
      "createdAt": "2025-09-15T14:30:00Z",
      "createdBy": {
        "id": "other-user-uuid",
        "username": "bob",
        "displayName": "Bob Johnson"
      }
    }
  ]
}
```

---

### POST /workspaces

**Описание:** Создать новый workspace

**Request:**

```json
{
  "name": "My New Team"
}
```

**Validation:**
- `name`: required, min 3, max 50 chars

**Response:**

```http
HTTP/1.1 201 Created
Location: /api/v1/workspaces/new-workspace-uuid
```

```json
{
  "id": "new-workspace-uuid",
  "name": "My New Team",
  "role": "admin",
  "memberCount": 1,
  "unreadChats": 0,
  "createdAt": "2025-09-30T10:00:00Z",
  "createdBy": {
    "id": "current-user-uuid",
    "username": "alice",
    "displayName": "Alice Smith"
  }
}
```

**Backend действия:**
1. Создаёт Group в Keycloak
2. Добавляет текущего пользователя в Group с ролью `workspace-admin`
3. Создаёт Workspace в БД
4. Возвращает Workspace

---

### GET /w/{id}

**Описание:** Получить workspace

**Request:**

```http
GET /api/v1/w/workspace-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "workspace-uuid",
  "name": "Engineering Team",
  "role": "admin",
  "memberCount": 15,
  "createdAt": "2025-09-01T10:00:00Z",
  "createdBy": {
    "id": "user-uuid",
    "username": "alice",
    "displayName": "Alice Smith"
  },
  "settings": {
    "defaultChatVisibility": "private",
    "allowMembersToInvite": false
  }
}
```

**Errors:**
- `404 NOT_FOUND` — workspace не существует
- `403 FORBIDDEN` — нет доступа к workspace

---

### PUT /w/{id}

**Описание:** Обновить workspace

**Права:** только `workspace-admin`

**Request:**

```json
{
  "name": "Updated Team Name",
  "settings": {
    "defaultChatVisibility": "public"
  }
}
```

**Response:**

```json
{
  "id": "workspace-uuid",
  "name": "Updated Team Name",
  "role": "admin",
  "memberCount": 15,
  "createdAt": "2025-09-01T10:00:00Z",
  "createdBy": {...},
  "settings": {
    "defaultChatVisibility": "public",
    "allowMembersToInvite": false
  }
}
```

---

### DELETE /w/{id}

**Описание:** Удалить workspace

**Права:** только `workspace-admin`

**Request:**

```http
DELETE /api/v1/w/workspace-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Backend действия:**
1. Удаляет все чаты workspace
2. Удаляет Workspace из БД
3. Удаляет Group из Keycloak

**Предупреждение:** Необратимая операция, требуется подтверждение на frontend.

---

### GET /w/{id}/members

**Описание:** Список участников workspace

**Request:**

```http
GET /api/v1/w/workspace-uuid/members?limit=50&cursor=... HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "data": [
    {
      "id": "user-uuid-1",
      "username": "alice",
      "email": "alice@example.com",
      "displayName": "Alice Smith",
      "avatarUrl": "...",
      "role": "admin",
      "joinedAt": "2025-09-01T10:00:00Z"
    },
    {
      "id": "user-uuid-2",
      "username": "bob",
      "email": "bob@example.com",
      "displayName": "Bob Johnson",
      "avatarUrl": "...",
      "role": "member",
      "joinedAt": "2025-09-05T14:20:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "eyJpZCI6...",
    "hasMore": false,
    "total": 15
  }
}
```

---

### DELETE /w/{id}/members/{userId}

**Описание:** Удалить участника из workspace

**Права:** только `workspace-admin`

**Request:**

```http
DELETE /api/v1/w/workspace-uuid/members/user-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Backend действия:**
1. Удаляет пользователя из Keycloak Group
2. Пользователь теряет доступ ко всем чатам workspace

**Errors:**
- `403 FORBIDDEN` — не workspace-admin
- `400 BAD_REQUEST` — нельзя удалить себя (должен быть хотя бы один admin)

---

### PUT /w/{id}/members/{userId}/role

**Описание:** Изменить роль участника

**Права:** только `workspace-admin`

**Request:**

```json
{
  "role": "admin"
}
```

**Validation:**
- `role`: required, enum: `admin`, `member`

**Response:**

```json
{
  "id": "user-uuid",
  "username": "bob",
  "displayName": "Bob Johnson",
  "role": "admin",
  "joinedAt": "2025-09-05T14:20:00Z"
}
```

**Backend действия:**
- Добавляет/удаляет Client Role `workspace-admin` в Keycloak Group

---

### POST /w/{id}/invites

**Описание:** Создать invite link

**Права:** только `workspace-admin`

**Request:**

```json
{
  "expiresIn": "7d",
  "maxUses": 10
}
```

**Validation:**
- `expiresIn`: optional, format: `{number}{unit}` (h, d, w), default: `7d`
- `maxUses`: optional, int, default: `null` (unlimited)

**Response:**

```http
HTTP/1.1 201 Created
```

```json
{
  "id": "invite-uuid",
  "token": "random-secure-token",
  "url": "https://teamsup.com/invite/random-secure-token",
  "workspaceId": "workspace-uuid",
  "createdBy": {
    "id": "user-uuid",
    "username": "alice",
    "displayName": "Alice Smith"
  },
  "expiresAt": "2025-10-07T10:00:00Z",
  "maxUses": 10,
  "usedCount": 0,
  "isActive": true,
  "createdAt": "2025-09-30T10:00:00Z"
}
```

---

### GET /w/{id}/invites

**Описание:** Список invite links

**Права:** только `workspace-admin`

**Request:**

```http
GET /api/v1/w/workspace-uuid/invites HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "data": [
    {
      "id": "invite-uuid-1",
      "token": "token-1",
      "url": "https://teamsup.com/invite/token-1",
      "expiresAt": "2025-10-07T10:00:00Z",
      "maxUses": 10,
      "usedCount": 3,
      "isActive": true,
      "createdAt": "2025-09-30T10:00:00Z",
      "createdBy": {...}
    }
  ]
}
```

---

### DELETE /w/{id}/invites/{inviteId}

**Описание:** Удалить/деактивировать invite

**Права:** только `workspace-admin`

**Request:**

```http
DELETE /api/v1/w/workspace-uuid/invites/invite-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

---

### POST /invite/{token}

**Описание:** Принять приглашение в workspace

**Аутентификация:** Требуется (если не авторизован → редирект на login)

**Request:**

```http
POST /api/v1/invite/random-secure-token HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 200 OK
```

```json
{
  "workspace": {
    "id": "workspace-uuid",
    "name": "Engineering Team",
    "role": "member",
    "memberCount": 16
  },
  "redirectUrl": "/w/workspace-uuid"
}
```

**Backend действия:**
1. Валидирует token (существует, не истёк, не превышен maxUses)
2. Добавляет пользователя в Keycloak Group с ролью `workspace-member`
3. Инкрементирует `usedCount`
4. Возвращает workspace

**Errors:**
- `404 NOT_FOUND` — invite не найден
- `400 BAD_REQUEST` — invite истёк или превышен maxUses
- `409 CONFLICT` — пользователь уже в workspace

---

## Chat Endpoints

### GET /w/{wid}/chats

**Описание:** Список чатов workspace

**Request:**

```http
GET /api/v1/w/workspace-uuid/chats?type=task,bug&status=In Progress&assignee=user-uuid&sort=-createdAt&limit=50&cursor=... HTTP/1.1
Authorization: Bearer {token}
```

**Query Parameters:**
- `type`: filter by type (task, bug, epic, discussion)
- `status`: filter by status
- `assignee`: filter by assignee
- `isPublic`: filter by visibility (true/false)
- `search`: full-text search in title
- `sort`: sort field (createdAt, updatedAt, title)
- `limit`: page size (default: 50, max: 100)
- `cursor`: pagination cursor

**Response:**

```json
{
  "data": [
    {
      "id": "chat-uuid-1",
      "workspaceId": "workspace-uuid",
      "type": "task",
      "title": "Implement OAuth authentication",
      "isPublic": true,
      "status": "In Progress",
      "assignee": {
        "id": "user-uuid",
        "username": "alice",
        "displayName": "Alice Smith",
        "avatarUrl": "..."
      },
      "priority": "High",
      "dueDate": "2025-10-20T00:00:00Z",
      "participantCount": 5,
      "unreadCount": 2,
      "createdAt": "2025-09-20T10:00:00Z",
      "updatedAt": "2025-09-30T15:30:00Z",
      "createdBy": {...}
    },
    {
      "id": "chat-uuid-2",
      "workspaceId": "workspace-uuid",
      "type": "bug",
      "title": "Login error on Chrome",
      "isPublic": false,
      "status": "Fixed",
      "assignee": {...},
      "severity": "Critical",
      "participantCount": 3,
      "unreadCount": 0,
      "createdAt": "2025-09-25T14:00:00Z",
      "updatedAt": "2025-09-29T16:45:00Z",
      "createdBy": {...}
    }
  ],
  "pagination": {
    "nextCursor": "eyJpZCI6...",
    "hasMore": true,
    "total": 145
  }
}
```

---

### POST /w/{wid}/chats

**Описание:** Создать новый чат

**Request:**

```json
{
  "type": "task",
  "title": "Implement feature X",
  "isPublic": false,
  "initialMessage": "Нужно реализовать фичу X #priority High"
}
```

**Validation:**
- `type`: optional, enum: `discussion`, `task`, `bug`, `epic`, default: `discussion`
- `title`: required if type != discussion, min 3, max 200 chars
- `isPublic`: optional, boolean, default: `false`
- `initialMessage`: optional, string

**Response:**

```http
HTTP/1.1 201 Created
Location: /api/v1/chats/new-chat-uuid
```

```json
{
  "id": "new-chat-uuid",
  "workspaceId": "workspace-uuid",
  "type": "task",
  "title": "Implement feature X",
  "isPublic": false,
  "status": "To Do",
  "assignee": null,
  "priority": "High",
  "dueDate": null,
  "participants": [
    {
      "id": "current-user-uuid",
      "username": "alice",
      "displayName": "Alice Smith",
      "role": "admin",
      "joinedAt": "2025-09-30T10:00:00Z"
    }
  ],
  "participantCount": 1,
  "unreadCount": 0,
  "createdAt": "2025-09-30T10:00:00Z",
  "updatedAt": "2025-09-30T10:00:00Z",
  "createdBy": {...}
}
```

**Backend действия:**
1. Создаёт Chat aggregate
2. Если `type` != `discussion` → создаёт TaskEntity
3. Если `initialMessage` → создаёт первое сообщение, парсит теги
4. Добавляет создателя как participant (admin)
5. Публикует события (ChatCreated, TaskCreated, MessagePosted)

---

### GET /chats/{id}

**Описание:** Получить детали чата

**Request:**

```http
GET /api/v1/chats/chat-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "chat-uuid",
  "workspaceId": "workspace-uuid",
  "type": "task",
  "title": "Implement OAuth authentication",
  "isPublic": true,
  "status": "In Progress",
  "assignee": {
    "id": "user-uuid",
    "username": "alice",
    "displayName": "Alice Smith",
    "avatarUrl": "..."
  },
  "priority": "High",
  "dueDate": "2025-10-20T00:00:00Z",
  "customFields": {
    "sprint": "Sprint-42",
    "component": "Auth"
  },
  "participants": [
    {
      "id": "user-uuid-1",
      "username": "alice",
      "displayName": "Alice Smith",
      "avatarUrl": "...",
      "role": "admin",
      "joinedAt": "2025-09-20T10:00:00Z"
    },
    {
      "id": "user-uuid-2",
      "username": "bob",
      "displayName": "Bob Johnson",
      "avatarUrl": "...",
      "role": "member",
      "joinedAt": "2025-09-22T14:30:00Z"
    }
  ],
  "participantCount": 5,
  "unreadCount": 2,
  "createdAt": "2025-09-20T10:00:00Z",
  "updatedAt": "2025-09-30T15:30:00Z",
  "createdBy": {...},
  "accessLevel": "write"
}
```

**Errors:**
- `404 NOT_FOUND` — чат не найден
- `403 FORBIDDEN` — нет доступа (приватный чат, не участник)

---

### PUT /chats/{id}

**Описание:** Обновить чат (metadata)

**Права:** только `chat admin`

**Request:**

```json
{
  "title": "Updated title",
  "isPublic": true
}
```

**Response:**

```json
{
  "id": "chat-uuid",
  "workspaceId": "workspace-uuid",
  "type": "task",
  "title": "Updated title",
  "isPublic": true,
  "status": "In Progress",
  "assignee": {...},
  "priority": "High",
  "participants": [...],
  "createdAt": "2025-09-20T10:00:00Z",
  "updatedAt": "2025-09-30T16:00:00Z"
}
```

**Примечание:** Для изменения `status`, `assignee`, `priority` используются теги в сообщениях.

---

### DELETE /chats/{id}

**Описание:** Удалить чат

**Права:** только `chat admin` или `workspace admin`

**Request:**

```http
DELETE /api/v1/chats/chat-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Backend действия:**
1. Soft delete Chat (помечает как deleted)
2. Удаляет TaskEntity (если есть)
3. Скрывает с канбана
4. Сообщения остаются в БД (для audit)

---

### POST /chats/{id}/join

**Описание:** Присоединиться к публичному чату

**Request:**

```http
POST /api/v1/chats/chat-uuid/join HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "chat-uuid",
  "participants": [
    {...},
    {
      "id": "current-user-uuid",
      "username": "alice",
      "displayName": "Alice Smith",
      "role": "member",
      "joinedAt": "2025-09-30T16:30:00Z"
    }
  ],
  "participantCount": 6
}
```

**Errors:**
- `403 FORBIDDEN` — чат не публичный
- `409 CONFLICT` — уже участник

---

### POST /chats/{id}/leave

**Описание:** Покинуть чат

**Request:**

```http
POST /api/v1/chats/chat-uuid/leave HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Errors:**
- `400 BAD_REQUEST` — нельзя покинуть, если единственный admin

---

### POST /chats/{id}/participants

**Описание:** Добавить участника в чат

**Права:** только `chat admin`

**Request:**

```json
{
  "userId": "user-uuid",
  "role": "member"
}
```

**Validation:**
- `userId`: required, UUID
- `role`: optional, enum: `member`, `admin`, default: `member`

**Response:**

```json
{
  "id": "user-uuid",
  "username": "charlie",
  "displayName": "Charlie Brown",
  "avatarUrl": "...",
  "role": "member",
  "joinedAt": "2025-09-30T17:00:00Z"
}
```

**Errors:**
- `404 NOT_FOUND` — пользователь не найден
- `403 FORBIDDEN` — пользователь не в workspace
- `409 CONFLICT` — уже участник

---

### DELETE /chats/{id}/participants/{userId}

**Описание:** Удалить участника из чата

**Права:** только `chat admin`

**Request:**

```http
DELETE /api/v1/chats/chat-uuid/participants/user-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Errors:**
- `400 BAD_REQUEST` — нельзя удалить последнего admin

---

## Message Endpoints

### GET /chats/{id}/messages

**Описание:** Получить сообщения чата

**Request:**

```http
GET /api/v1/chats/chat-uuid/messages?limit=50&cursor=... HTTP/1.1
Authorization: Bearer {token}
```

**Query Parameters:**
- `limit`: page size (default: 50, max: 100)
- `cursor`: pagination cursor (для загрузки истории)
- `since`: timestamp (ISO 8601) — получить сообщения после этой даты

**Response:**

```json
{
  "data": [
    {
      "id": "msg-uuid-1",
      "chatId": "chat-uuid",
      "author": {
        "id": "user-uuid",
        "username": "alice",
        "displayName": "Alice Smith",
        "avatarUrl": "..."
      },
      "content": "Начал работу над задачей",
      "tags": [],
      "isSystemMessage": false,
      "editedAt": null,
      "createdAt": "2025-09-30T10:00:00Z"
    },
    {
      "id": "msg-uuid-2",
      "chatId": "chat-uuid",
      "author": {
        "id": "user-uuid",
        "username": "alice",
        "displayName": "Alice Smith",
        "avatarUrl": "..."
      },
      "content": "Закончил работу\n#status Done",
      "tags": [
        {"key": "status", "value": "Done"}
      ],
      "isSystemMessage": false,
      "editedAt": null,
      "createdAt": "2025-09-30T15:30:00Z"
    },
    {
      "id": "msg-uuid-3",
      "chatId": "chat-uuid",
      "author": {
        "id": "system",
        "username": "system",
        "displayName": "System"
      },
      "content": "✅ Status changed to Done",
      "tags": [],
      "isSystemMessage": true,
      "createdAt": "2025-09-30T15:30:01Z"
    }
  ],
  "pagination": {
    "nextCursor": "eyJpZCI6...",
    "hasMore": true,
    "total": 142
  }
}
```

**Сортировка:** По умолчанию от старых к новым (ascending by createdAt).

---

### POST /chats/{id}/messages

**Описание:** Отправить сообщение в чат

**Права:** `write` или `admin` access level

**Request:**

```json
{
  "content": "Закончил работу\n#status Done #assignee @bob"
}
```

**Validation:**
- `content`: required, min 1, max 10000 chars

**Response:**

```http
HTTP/1.1 201 Created
Location: /api/v1/messages/new-msg-uuid
```

```json
{
  "id": "new-msg-uuid",
  "chatId": "chat-uuid",
  "author": {
    "id": "current-user-uuid",
    "username": "alice",
    "displayName": "Alice Smith",
    "avatarUrl": "..."
  },
  "content": "Закончил работу\n#status Done #assignee @bob",
  "tags": [
    {"key": "status", "value": "Done"},
    {"key": "assignee", "value": "@bob"}
  ],
  "isSystemMessage": false,
  "editedAt": null,
  "createdAt": "2025-09-30T16:00:00Z"
}
```

**Backend действия:**
1. Сохраняет сообщение
2. Публикует `MessagePosted` event
3. Event flow → TagParser → CommandExecutor → уведомления
4. Broadcast через WebSocket

**Errors:**
- `403 FORBIDDEN` — нет write access к чату

---

### PUT /messages/{id}

**Описание:** Редактировать сообщение

**Права:** автор (< 5 мин) или `chat admin`

**Request:**

```json
{
  "content": "Updated message content"
}
```

**Response:**

```json
{
  "id": "msg-uuid",
  "chatId": "chat-uuid",
  "author": {...},
  "content": "Updated message content",
  "tags": [],
  "isSystemMessage": false,
  "editedAt": "2025-09-30T16:10:00Z",
  "createdAt": "2025-09-30T16:00:00Z"
}
```

**Ограничение (MVP):** Редактирование сообщения с тегами НЕ пересчитывает команды.

**Errors:**
- `403 FORBIDDEN` — нет прав редактировать (не автор или > 5 мин)

---

### DELETE /messages/{id}

**Описание:** Удалить сообщение

**Права:** автор (< 5 мин) или `chat admin`

**Request:**

```http
DELETE /api/v1/messages/msg-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

**Backend действия:**
- Soft delete (помечает как deleted)
- В UI показывается "[message deleted]"
- Удаление сообщения с тегами НЕ откатывает изменения

---

## Task Endpoints (Read Model)

### GET /w/{wid}/tasks

**Описание:** Список задач (фильтрованный, для канбана)

**Request:**

```http
GET /api/v1/w/workspace-uuid/tasks?type=task&status=In Progress&assignee=user-uuid&sort=-updatedAt&limit=100 HTTP/1.1
Authorization: Bearer {token}
```

**Query Parameters:**
- `type`: filter by type (task, bug, epic)
- `status`: filter by status
- `assignee`: filter by assignee
- `priority`: filter by priority
- `dueDate`: filter (before, after, between)
- `search`: full-text search
- `sort`: sort field
- `limit`, `cursor`: pagination

**Response:**

```json
{
  "data": [
    {
      "id": "task-uuid-1",
      "chatId": "chat-uuid-1",
      "workspaceId": "workspace-uuid",
      "type": "task",
      "title": "Implement OAuth",
      "status": "In Progress",
      "assignee": {
        "id": "user-uuid",
        "username": "alice",
        "displayName": "Alice Smith",
        "avatarUrl": "..."
      },
      "priority": "High",
      "dueDate": "2025-10-20T00:00:00Z",
      "customFields": {},
      "createdAt": "2025-09-20T10:00:00Z",
      "updatedAt": "2025-09-30T15:30:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "...",
    "hasMore": false,
    "total": 23
  }
}
```

---

### GET /tasks/{id}

**Описание:** Получить задачу (то же что GET /chats/{id}, но только для typed чатов)

**Request:**

```http
GET /api/v1/tasks/task-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:** (аналогично GET /chats/{id})

---

### GET /w/{wid}/board

**Описание:** Канбан-доска (группировка по статусам)

**Request:**

```http
GET /api/v1/w/workspace-uuid/board?type=task,bug HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "columns": [
    {
      "status": "To Do",
      "tasks": [
        {
          "id": "task-uuid-1",
          "chatId": "chat-uuid-1",
          "title": "Fix bug X",
          "assignee": {...},
          "priority": "High",
          "dueDate": null
        },
        {
          "id": "task-uuid-2",
          "chatId": "chat-uuid-2",
          "title": "Implement feature Y",
          "assignee": null,
          "priority": "Medium",
          "dueDate": "2025-10-15T00:00:00Z"
        }
      ],
      "count": 12
    },
    {
      "status": "In Progress",
      "tasks": [
        {
          "id": "task-uuid-3",
          "chatId": "chat-uuid-3",
          "title": "OAuth implementation",
          "assignee": {...},
          "priority": "High",
          "dueDate": "2025-10-20T00:00:00Z"
        }
      ],
      "count": 5
    },
    {
      "status": "Done",
      "tasks": [],
      "count": 34
    }
  ],
  "totalTasks": 51
}
```

**Примечание:** По умолчанию в каждой колонке показываются первые N задач (например, 10). Для полного списка использовать GET /w/{wid}/tasks с фильтром по status.

---

## Notification Endpoints

### GET /notifications

**Описание:** Список уведомлений текущего пользователя

**Request:**

```http
GET /api/v1/notifications?unreadOnly=true&limit=20&cursor=... HTTP/1.1
Authorization: Bearer {token}
```

**Query Parameters:**
- `unreadOnly`: filter only unread (boolean)
- `type`: filter by type
- `limit`, `cursor`: pagination

**Response:**

```json
{
  "data": [
    {
      "id": "notif-uuid-1",
      "userId": "current-user-uuid",
      "type": "task.status_changed",
      "title": "Task status changed",
      "message": "Task 'OAuth implementation' status changed to Done",
      "resourceId": "task-uuid",
      "resourceType": "task",
      "readAt": null,
      "createdAt": "2025-09-30T15:30:00Z"
    },
    {
      "id": "notif-uuid-2",
      "userId": "current-user-uuid",
      "type": "chat.mentioned",
      "title": "You were mentioned",
      "message": "@alice mentioned you in 'Project discussion'",
      "resourceId": "chat-uuid",
      "resourceType": "chat",
      "readAt": "2025-09-30T16:00:00Z",
      "createdAt": "2025-09-30T14:00:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "...",
    "hasMore": true,
    "total": 45
  },
  "unreadCount": 12
}
```

---

### PUT /notifications/{id}/read

**Описание:** Отметить уведомление как прочитанное

**Request:**

```http
PUT /api/v1/notifications/notif-uuid/read HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "notif-uuid",
  "userId": "current-user-uuid",
  "type": "task.status_changed",
  "title": "Task status changed",
  "message": "...",
  "resourceId": "task-uuid",
  "resourceType": "task",
  "readAt": "2025-09-30T16:30:00Z",
  "createdAt": "2025-09-30T15:30:00Z"
}
```

---

### DELETE /notifications/{id}

**Описание:** Удалить уведомление

**Request:**

```http
DELETE /api/v1/notifications/notif-uuid HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

---

## Admin / DLQ Endpoints

### GET /admin/dlq

**Описание:** Список событий в Dead Letter Queue

**Права:** только `system-admin`

**Request:**

```http
GET /api/v1/admin/dlq?status=pending&eventType=MessagePosted&limit=50&cursor=... HTTP/1.1
Authorization: Bearer {token}
```

**Query Parameters:**
- `status`: filter (pending, replayed, discarded)
- `eventType`: filter by event type
- `limit`, `cursor`: pagination

**Response:**

```json
{
  "data": [
    {
      "id": "dlq-entry-uuid",
      "eventId": "event-uuid",
      "eventType": "MessagePosted",
      "aggregateId": "chat-uuid",
      "payload": {...},
      "error": "Failed to parse tags: invalid syntax",
      "attempts": 6,
      "status": "pending",
      "createdAt": "2025-09-30T10:00:00Z",
      "replayedAt": null,
      "replayedBy": null
    }
  ],
  "pagination": {
    "nextCursor": "...",
    "hasMore": false,
    "total": 3
  }
}
```

---

### POST /admin/dlq/{id}/replay

**Описание:** Повторно обработать событие из DLQ

**Права:** только `system-admin`

**Request:**

```http
POST /api/v1/admin/dlq/dlq-entry-uuid/replay HTTP/1.1
Authorization: Bearer {token}
```

**Response:**

```json
{
  "id": "dlq-entry-uuid",
  "eventId": "event-uuid",
  "status": "replayed",
  "replayedAt": "2025-09-30T17:00:00Z",
  "replayedBy": {
    "id": "admin-user-uuid",
    "username": "admin",
    "displayName": "Admin User"
  }
}
```

**Backend действия:**
1. Загружает событие из DLQ
2. Публикует событие в Event Bus заново
3. Обновляет статус в DLQ

---

### POST /admin/dlq/{id}/discard

**Описание:** Отметить событие как "discarded" (игнорировать)

**Права:** только `system-admin`

**Request:**

```json
{
  "reason": "Invalid event, can be safely ignored"
}
```

**Response:**

```json
{
  "id": "dlq-entry-uuid",
  "status": "discarded",
  "discardedAt": "2025-09-30T17:00:00Z",
  "discardedBy": {...},
  "discardReason": "Invalid event, can be safely ignored"
}
```

---

## WebSocket API

### Connection

**URL:** `ws://localhost:8080/ws?token={accessToken}`

**Authentication:** Token в query parameter

**Handshake:**

```
Client → Server:
WebSocket Upgrade Request
GET /ws?token=eyJhbG... HTTP/1.1
Upgrade: websocket

Server → Client:
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
```

**После подключения:**

```json
Server → Client:
{
  "type": "connected",
  "userId": "user-uuid",
  "timestamp": "2025-09-30T10:00:00Z"
}
```

---

### Message Format

**Base structure:**

```json
{
  "type": "message.type",
  "data": { ... },
  "timestamp": "2025-09-30T10:00:00Z"
}
```

---

### Client → Server Messages

#### Subscribe to Chat

```json
{
  "type": "subscribe.chat",
  "chatId": "chat-uuid"
}
```

**Backend:**
- Добавляет клиента в список подписчиков чата
- Проверяет права доступа (участник чата или публичный в workspace)

**Response (если успешно):**

```json
{
  "type": "subscribed.chat",
  "chatId": "chat-uuid",
  "timestamp": "2025-09-30T10:00:00Z"
}
```

**Response (если ошибка):**

```json
{
  "type": "error",
  "code": "FORBIDDEN",
  "message": "You don't have access to this chat"
}
```

---

#### Unsubscribe from Chat

```json
{
  "type": "unsubscribe.chat",
  "chatId": "chat-uuid"
}
```

**Response:**

```json
{
  "type": "unsubscribed.chat",
  "chatId": "chat-uuid",
  "timestamp": "2025-09-30T10:00:00Z"
}
```

---

#### Subscribe to Workspace

```json
{
  "type": "subscribe.workspace",
  "workspaceId": "workspace-uuid"
}
```

**Backend:**
- Проверяет membership в workspace
- Добавляет клиента в список подписчиков workspace
- Клиент будет получать обновления канбана

---

#### Typing Indicator

```json
{
  "type": "chat.typing",
  "chatId": "chat-uuid"
}
```

**Backend:**
- Broadcast другим участникам чата
- Автоматически сбрасывается через 3 секунды

---

#### Ping (Keepalive)

```json
{
  "type": "ping"
}
```

**Response:**

```json
{
  "type": "pong",
  "timestamp": "2025-09-30T10:00:00Z"
}
```

---

### Server → Client Messages

#### New Message Posted

```json
{
  "type": "chat.message.posted",
  "data": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "author": {
      "id": "user-uuid",
      "username": "alice",
      "displayName": "Alice Smith",
      "avatarUrl": "..."
    },
    "content": "Закончил работу\n#status Done",
    "tags": [
      {"key": "status", "value": "Done"}
    ],
    "isSystemMessage": false,
    "createdAt": "2025-09-30T15:30:00Z"
  },
  "timestamp": "2025-09-30T15:30:00.123Z"
}
```

**Broadcast:** Всем подписчикам чата

---

#### Message Edited

```json
{
  "type": "chat.message.edited",
  "data": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "content": "Updated content",
    "editedAt": "2025-09-30T15:35:00Z"
  },
  "timestamp": "2025-09-30T15:35:00.456Z"
}
```

---

#### Message Deleted

```json
{
  "type": "chat.message.deleted",
  "data": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid"
  },
  "timestamp": "2025-09-30T15:40:00.789Z"
}
```

---

#### Task Updated (Канбан)

```json
{
  "type": "task.updated",
  "data": {
    "taskId": "task-uuid",
    "chatId": "chat-uuid",
    "workspaceId": "workspace-uuid",
    "changes": {
      "status": {
        "old": "In Progress",
        "new": "Done"
      }
    },
    "updatedAt": "2025-09-30T15:30:00Z"
  },
  "timestamp": "2025-09-30T15:30:00.345Z"
}
```

**Broadcast:** Всем подписчикам workspace

**Frontend action:** Переместить карточку на канбане из колонки "In Progress" в "Done"

---

#### User Typing

```json
{
  "type": "chat.typing",
  "data": {
    "chatId": "chat-uuid",
    "userId": "user-uuid",
    "username": "bob",
    "displayName": "Bob Johnson"
  },
  "timestamp": "2025-09-30T15:30:00.567Z"
}
```

**Broadcast:** Всем подписчикам чата (кроме отправителя)

**Frontend:** Показать "Bob is typing..." (скрыть через 3 сек)

---

#### User Joined/Left Chat

```json
{
  "type": "chat.participant.joined",
  "data": {
    "chatId": "chat-uuid",
    "user": {
      "id": "user-uuid",
      "username": "charlie",
      "displayName": "Charlie Brown",
      "avatarUrl": "..."
    },
    "role": "member",
    "joinedAt": "2025-09-30T16:00:00Z"
  },
  "timestamp": "2025-09-30T16:00:00.123Z"
}
```

```json
{
  "type": "chat.participant.left",
  "data": {
    "chatId": "chat-uuid",
    "userId": "user-uuid",
    "username": "charlie"
  },
  "timestamp": "2025-09-30T16:10:00.456Z"
}
```

---

#### Notification

```json
{
  "type": "notification.new",
  "data": {
    "id": "notif-uuid",
    "type": "task.status_changed",
    "title": "Task status changed",
    "message": "Task 'OAuth implementation' status changed to Done",
    "resourceId": "task-uuid",
    "resourceType": "task",
    "createdAt": "2025-09-30T15:30:00Z"
  },
  "timestamp": "2025-09-30T15:30:01.234Z"
}
```

**Broadcast:** Только конкретному пользователю

---

#### Error

```json
{
  "type": "error",
  "code": "FORBIDDEN",
  "message": "You don't have access to this chat",
  "context": {
    "chatId": "chat-uuid"
  },
  "timestamp": "2025-09-30T15:30:00.789Z"
}
```

---

### Reconnection Strategy

**Frontend должен:**

1. **Обработать disconnect:**
   - WebSocket connection lost
   - Показать UI индикатор "Reconnecting..."

2. **Exponential backoff reconnect:**
   - Попытка 1: сразу
   - Попытка 2: через 1s
   - Попытка 3: через 2s
   - Попытка 4: через 4s
   - Попытка 5: через 8s
   - Max delay: 30s

3. **После reconnect:**
   - Отправить все `subscribe.chat` и `subscribe.workspace` заново
   - Запросить пропущенные сообщения через REST API:
     ```
     GET /chats/{id}/messages?since={lastMessageTimestamp}
     ```

4. **Пример кода:**

```javascript
class WebSocketClient {
  constructor(token) {
    this.token = token;
    this.reconnectAttempts = 0;
    this.maxReconnectDelay = 30000;
    this.subscriptions = new Set();
  }

  connect() {
    this.ws = new WebSocket(`ws://localhost:8080/ws?token=${this.token}`);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;

      // Восстанавливаем подписки
      this.subscriptions.forEach(sub => {
        this.send(sub);
      });
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.reconnect();
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };
  }

  reconnect() {
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts),
      this.maxReconnectDelay
    );

    console.log(`Reconnecting in ${delay}ms...`);
    this.reconnectAttempts++;

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  subscribe(type, id) {
    const message = { type: `subscribe.${type}`, [`${type}Id`]: id };
    this.subscriptions.add(message);
    this.send(message);
  }

  send(message) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }
}
```

---

## Rate Limiting

### Global Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /auth/login | 5 requests | per minute |
| POST /workspaces | 5 requests | per hour |
| POST /w/{id}/invites | 10 requests | per hour |
| POST /chats/{id}/messages | 60 requests | per minute |
| GET * | 300 requests | per minute |

### Response Headers

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1696077600
```

### Rate Limit Exceeded Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 30

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again in 30 seconds.",
    "retryAfter": 30
  }
}
```

---

## API Versioning Strategy

### Deprecation Process

1. **Анонс deprecation** (минимум за 3 месяца)
   - Документация обновляется
   - Response header: `X-API-Deprecated: true`
   - Response header: `X-API-Sunset: 2026-01-01T00:00:00Z`

2. **Поддержка обеих версий** (переходный период)
   - `/api/v1/...` (deprecated)
   - `/api/v2/...` (новая версия)

3. **Отключение старой версии**
   - После sunset date
   - `/api/v1/...` → 410 Gone

```http
HTTP/1.1 410 Gone

{
  "error": {
    "code": "API_VERSION_DEPRECATED",
    "message": "API v1 is no longer available. Please use v2.",
    "upgradeUrl": "https://docs.teamsup.com/api/migration/v1-to-v2"
  }
}
```

---

## Резюме

### REST API

- **Base:** `/api/v1`
- **Auth:** Bearer JWT
- **Format:** JSON
- **Pagination:** Cursor-based
- **Errors:** Structured с кодами

### WebSocket API

- **URL:** `ws://host/ws?token={jwt}`
- **Protocol:** Custom JSON
- **Subscriptions:** Chat, Workspace
- **Real-time:** Messages, Tasks, Notifications

### Key Endpoints (MVP)

- ✅ Authentication (OAuth flow)
- ✅ Workspaces (CRUD, members, invites)
- ✅ Chats (CRUD, participants)
- ✅ Messages (CRUD)
- ✅ Tasks (read model, board)
- ✅ Notifications
- ✅ Admin/DLQ

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. ✅ Детальная грамматика тегов
4. ✅ Права доступа и security model
5. ✅ Event flow детально
6. ✅ API контракты (HTTP + WebSocket)
7. **TODO:** Структура кода (внутри internal/)
8. **TODO:** План реализации MVP (roadmap)
