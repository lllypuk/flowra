# Flowra API Documentation

This directory contains the complete API documentation for the Flowra Chat System with Task Management.

## Overview

Flowra provides a RESTful API for managing workspaces, chats, messages, tasks, and notifications. Additionally, it offers WebSocket connectivity for real-time updates.

## Quick Start

### 1. Start the Development Environment

```bash
# Start infrastructure services
docker-compose up -d mongodb redis keycloak

# Start the API server
go run cmd/api/main.go
```

### 2. Authenticate

All API requests (except `/auth/login` and health checks) require JWT authentication.

```bash
# Obtain tokens via OAuth flow with Keycloak
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"code": "<oauth_code>", "redirect_uri": "http://localhost:8080/callback"}'
```

### 3. Make Authenticated Requests

```bash
curl http://localhost:8080/api/v1/workspaces \
  -H "Authorization: Bearer <access_token>"
```

## API Documentation

### OpenAPI Specification

The complete API is documented in OpenAPI 3.1 format:

- **[openapi.yaml](./openapi.yaml)** - Full API specification with all endpoints, schemas, and examples

### View Documentation

You can view the API documentation using:

1. **Swagger UI** (recommended):
   ```bash
   # Install Swagger UI locally or use the online editor
   open https://editor.swagger.io/
   # Then import openapi.yaml
   ```

2. **Redoc**:
   ```bash
   npx @redocly/cli preview-docs openapi.yaml
   ```

3. **VS Code Extension**:
   - Install "OpenAPI (Swagger) Editor" extension
   - Open `openapi.yaml`

## Base URL

| Environment | URL |
|-------------|-----|
| Local Development | `http://localhost:8080/api/v1` |
| Production | `https://api.flowra.com/api/v1` |

## Authentication

Flowra uses JWT-based authentication via Keycloak SSO.

### Token Flow

1. User initiates OAuth flow with Keycloak
2. Keycloak redirects with authorization code
3. Exchange code for tokens via `/auth/login`
4. Use access token in `Authorization` header
5. Refresh tokens via `/auth/refresh` when expired

### Headers

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

## API Endpoints Overview

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Login with OAuth code |
| POST | `/auth/logout` | Logout current session |
| POST | `/auth/refresh` | Refresh access token |
| GET | `/auth/me` | Get current user |

### Users
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/users/me` | Get current user profile |
| PUT | `/users/me` | Update current user profile |
| GET | `/users/{id}` | Get user by ID |

### Workspaces
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces` | List user workspaces |
| POST | `/workspaces` | Create workspace |
| GET | `/workspaces/{id}` | Get workspace |
| PUT | `/workspaces/{id}` | Update workspace |
| DELETE | `/workspaces/{id}` | Delete workspace |
| POST | `/workspaces/{id}/members` | Add member |
| DELETE | `/workspaces/{id}/members/{user_id}` | Remove member |
| PUT | `/workspaces/{id}/members/{user_id}/role` | Update member role |

### Chats
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces/{id}/chats` | List chats |
| POST | `/workspaces/{id}/chats` | Create chat |
| GET | `/workspaces/{id}/chats/{chat_id}` | Get chat |
| PUT | `/workspaces/{id}/chats/{chat_id}` | Update chat |
| DELETE | `/workspaces/{id}/chats/{chat_id}` | Delete chat |
| POST | `/workspaces/{id}/chats/{chat_id}/participants` | Add participant |
| DELETE | `/workspaces/{id}/chats/{chat_id}/participants/{user_id}` | Remove participant |

### Messages
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces/{id}/chats/{chat_id}/messages` | List messages |
| POST | `/workspaces/{id}/chats/{chat_id}/messages` | Send message |
| PUT | `/messages/{message_id}` | Edit message |
| DELETE | `/messages/{message_id}` | Delete message |

### Tasks
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces/{id}/tasks` | List tasks |
| POST | `/workspaces/{id}/tasks` | Create task |
| GET | `/workspaces/{id}/tasks/{task_id}` | Get task |
| DELETE | `/workspaces/{id}/tasks/{task_id}` | Delete task |
| PUT | `/workspaces/{id}/tasks/{task_id}/status` | Change status |
| PUT | `/workspaces/{id}/tasks/{task_id}/assignee` | Assign task |
| PUT | `/workspaces/{id}/tasks/{task_id}/priority` | Change priority |
| PUT | `/workspaces/{id}/tasks/{task_id}/due-date` | Set due date |

### Notifications
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/notifications` | List notifications |
| GET | `/notifications/unread/count` | Get unread count |
| PUT | `/notifications/{id}/read` | Mark as read |
| PUT | `/notifications/mark-all-read` | Mark all as read |
| DELETE | `/notifications/{id}` | Delete notification |

### WebSocket
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/ws` | WebSocket connection |

### Health Checks
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |

## Response Format

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      // Optional field-level errors
    }
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_ERROR` | 500 | Server error |

## Pagination

List endpoints support pagination via query parameters:

```
GET /api/v1/workspaces?limit=20&offset=0
```

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `limit` | integer | 20 | 100 | Results per page |
| `offset` | integer | 0 | - | Skip results |

### Cursor-based Pagination

Some endpoints (messages) support cursor-based pagination:

```
GET /api/v1/chats/{id}/messages?limit=50&before=<message_id>
```

## Rate Limiting

| Endpoint Type | Limit | Window |
|--------------|-------|--------|
| General API | 100 requests | 1 minute |
| Auth endpoints | 10 requests | 1 minute |
| WebSocket messages | 60 messages | 1 minute |

Rate limit headers:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

## WebSocket API

Connect to `/api/v1/ws` with JWT token for real-time updates.

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws?token=<jwt_token>');
```

### Client Messages

```json
// Subscribe to chat updates
{"type": "subscribe_chat", "chat_id": "uuid"}

// Unsubscribe from chat
{"type": "unsubscribe_chat", "chat_id": "uuid"}

// Send typing indicator
{"type": "typing", "chat_id": "uuid"}

// Keepalive ping
{"type": "ping"}
```

### Server Messages

```json
// New message
{"type": "message_posted", "data": {...}, "timestamp": "..."}

// Message edited
{"type": "message_edited", "data": {...}, "timestamp": "..."}

// Task updated
{"type": "task_updated", "data": {...}, "timestamp": "..."}

// User typing
{"type": "user_typing", "data": {...}, "timestamp": "..."}

// Notification
{"type": "notification", "data": {...}, "timestamp": "..."}
```

## Postman Collection

Import the Postman collection for easy API testing:

1. Open Postman
2. Import → Upload Files → Select `postman_collection.json`
3. Set environment variables:
   - `base_url`: `http://localhost:8080/api/v1`
   - `access_token`: Your JWT token

## Development

### Validate OpenAPI Spec

```bash
# Using npx
npx @redocly/cli lint openapi.yaml

# Using spectral
npx @stoplight/spectral-cli lint openapi.yaml
```

### Generate Client SDK

```bash
# Generate TypeScript client
npx openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o ./sdk/typescript

# Generate Go client
openapi-generator-cli generate -i openapi.yaml -g go -o ./sdk/go
```

## Resources

- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0)
- [Swagger Editor](https://editor.swagger.io/)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [Echo Framework](https://echo.labstack.com/)

---

*Last updated: January 2026*
