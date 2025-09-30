# Security Model and Access Control

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

Система использует **Keycloak** для управления пользователями, аутентификации и авторизации. Максимум логики по пользователям и ролям вынесено в Keycloak, приложение работает с JWT токенами stateless.

## Архитектурные принципы

- **Keycloak как Source of Truth** для пользователей, ролей, workspace membership
- **Stateless авторизация** через JWT токены
- **RBAC** (Role-Based Access Control) на уровне Keycloak
- **Workspace isolation** — пользователи работают в контексте workspace
- **Self-service** — пользователи могут создавать workspace и приглашать других

## Иерархия доступа

```
System Level (Keycloak Realm)
    ↓
Workspace Level (Keycloak Groups)
    ↓
Chat Level (Application)
    ↓
Message Level (Application)
```

---

## Keycloak Configuration

### Realm Structure

```
Realm: teams-up

Realm Roles:
├─ user              — базовая роль (все зарегистрированные пользователи)
└─ system-admin      — суперадмин (полный доступ ко всему)

Client: teams-up-app
├─ Client ID: teams-up-app
├─ Protocol: openid-connect
├─ Access Type: confidential
├─ Valid Redirect URIs:
│  └─ http://localhost:8080/auth/callback
│  └─ https://app.teams-up.com/auth/callback
│
└─ Client Roles:
   ├─ workspace-admin   — администратор workspace
   └─ workspace-member  — участник workspace

Groups (динамически создаются):
├─ Group: "Engineering Team"
│  ├─ ID: keycloak-group-uuid-1
│  ├─ Attributes:
│  │  ├─ workspace_id: "workspace-uuid-1"
│  │  └─ created_at: "2025-09-30T10:00:00Z"
│  └─ Members:
│     ├─ alice@example.com [workspace-admin]
│     ├─ bob@example.com [workspace-member]
│     └─ charlie@example.com [workspace-member]
│
└─ Group: "Marketing Team"
   ├─ ID: keycloak-group-uuid-2
   ├─ Attributes:
   │  └─ workspace_id: "workspace-uuid-2"
   └─ Members:
      └─ diana@example.com [workspace-admin]
```

### Token Mappers

**Обязательные mappers для JWT:**

```yaml
Mappers:
  1. Group Membership Mapper:
     Name: groups
     Mapper Type: Group Membership
     Token Claim Name: groups
     Full group path: ON
     Add to ID token: ON
     Add to access token: ON
     Add to userinfo: ON

  2. User Attribute Mapper:
     Name: workspace_id
     Mapper Type: User Attribute
     User Attribute: workspace_id
     Token Claim Name: workspace_id
     Claim JSON Type: String

  3. Audience Mapper:
     Name: audience
     Included Client Audience: teams-up-app

  4. Client Roles Mapper:
     Name: client-roles
     Client ID: teams-up-app
     Token Claim Name: resource_access.teams-up-app.roles
     Add to access token: ON
```

### JWT Token Structure

```json
{
  "sub": "user-uuid",
  "email": "alice@example.com",
  "preferred_username": "alice",
  "name": "Alice Smith",
  "email_verified": true,

  "realm_access": {
    "roles": ["user"]
  },

  "resource_access": {
    "teams-up-app": {
      "roles": ["workspace-admin", "workspace-member"]
    }
  },

  "groups": [
    "/Engineering Team",
    "/Marketing Team"
  ],

  "iss": "http://localhost:8090/realms/teams-up",
  "aud": "teams-up-app",
  "exp": 1727692800,
  "iat": 1727689200
}
```

---

## Domain Model

### Workspace

```go
type Workspace struct {
    ID              UUID      // наш внутренний ID
    Name            string
    KeycloakGroupID string    // ID группы в Keycloak
    CreatedBy       UUID      // UserID создателя
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Связь с Keycloak:**
- Workspace.KeycloakGroupID → Keycloak Group.id
- Membership управляется в Keycloak (Users в Group)
- Роли (admin/member) хранятся как Client Roles в Group

### User

```go
type User struct {
    ID            UUID      // = Keycloak User.sub
    KeycloakID    string    // = Keycloak User.sub (дубликат для ясности)
    Username      string    // = preferred_username
    Email         string
    DisplayName   string    // = name
    AvatarURL     *string
    IsSystemAdmin bool      // из realm_access.roles.contains("system-admin")
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**Синхронизация:**
- При первом логине → создаём User в нашей БД
- При каждом логине → обновляем Username, Email, DisplayName
- IsSystemAdmin вычисляется из JWT realm roles

### Chat

```go
type Chat struct {
    ID           UUID
    WorkspaceID  UUID          // обязательная принадлежность к workspace
    Type         ChatType      // discussion, task, bug, epic
    IsPublic     bool          // публичный в рамках workspace
    CreatedBy    UUID
    CreatedAt    time.Time
    Participants []Participant
    TaskEntity   *TaskEntity
}

type Participant struct {
    UserID   UUID
    Role     ParticipantRole  // member, admin
    JoinedAt time.Time
}

type ParticipantRole string

const (
    ParticipantMember ParticipantRole = "member"
    ParticipantAdmin  ParticipantRole = "admin"
)
```

**Инварианты:**
- Chat ВСЕГДА принадлежит Workspace
- Создатель чата автоматически становится ParticipantAdmin
- IsPublic = true → все члены workspace могут читать (read-only до Join)

---

## Уровни доступа и права

### System Level (Realm Roles)

**Role: system-admin**

| Действие | Разрешено |
|----------|-----------|
| Доступ ко всем workspace | ✅ |
| Создание/удаление любого workspace | ✅ |
| Управление любым чатом/задачей | ✅ |
| Просмотр логов и метрик | ✅ |
| Изменение системных настроек | ✅ |

**Проверка:**
```go
func IsSystemAdmin(token *jwt.Token) bool {
    realmRoles := token.Get("realm_access.roles").([]interface{})
    for _, role := range realmRoles {
        if role.(string) == "system-admin" {
            return true
        }
    }
    return false
}
```

### Workspace Level (Keycloak Groups)

**Membership:** User является членом Group в Keycloak

**Role: workspace-admin** (Client Role в Group)

| Действие | workspace-admin | workspace-member | non-member |
|----------|----------------|------------------|------------|
| Просмотр публичных чатов | ✅ | ✅ | ❌ |
| Просмотр приватных чатов (если участник) | ✅ | ✅ | ❌ |
| Создание чата | ✅ | ✅ | ❌ |
| Генерация инвайт-ссылок | ✅ | ❌ | ❌ |
| Управление workspace настройками | ✅ | ❌ | ❌ |
| Удаление workspace | ✅ | ❌ | ❌ |
| Просмотр всех участников | ✅ | ✅ | ❌ |
| Удаление участников | ✅ | ❌ | ❌ |
| Назначение workspace-admin | ✅ | ❌ | ❌ |

**Проверка workspace membership:**
```go
func HasWorkspaceAccess(token *jwt.Token, workspaceID UUID) bool {
    // 1. System admin имеет доступ везде
    if IsSystemAdmin(token) {
        return true
    }

    // 2. Получаем Keycloak Group ID для workspace
    workspace := workspaceRepo.FindByID(workspaceID)
    keycloakGroupID := workspace.KeycloakGroupID

    // 3. Проверяем наличие группы в токене
    groups := token.Get("groups").([]interface{})
    for _, group := range groups {
        groupPath := group.(string)
        // Загружаем Group из Keycloak по path, сравниваем ID
        if groupMatchesWorkspace(groupPath, keycloakGroupID) {
            return true
        }
    }

    return false
}

func IsWorkspaceAdmin(token *jwt.Token, workspaceID UUID) bool {
    if IsSystemAdmin(token) {
        return true
    }

    // Проверяем наличие роли workspace-admin для этой группы
    workspace := workspaceRepo.FindByID(workspaceID)

    // Из токена извлекаем client roles
    clientRoles := token.Get("resource_access.teams-up-app.roles").([]interface{})

    // Проверяем, что пользователь в нужной группе И имеет роль admin
    // (детальная реализация зависит от Keycloak mapper для group roles)

    return hasGroupRole(token, workspace.KeycloakGroupID, "workspace-admin")
}
```

### Chat Level (Application)

**Role: ParticipantAdmin / ParticipantMember**

#### Приватный чат (IsPublic = false)

| Действие | Chat Admin | Chat Member | Workspace Member (не в чате) |
|----------|------------|-------------|------------------------------|
| Просмотр чата | ✅ | ✅ | ❌ |
| Просмотр истории сообщений | ✅ | ✅ | ❌ |
| Отправка сообщений | ✅ | ✅ | ❌ |
| Применение тегов (#status, #assignee) | ✅ | ✅ | ❌ |
| Добавление участников | ✅ | ❌ | ❌ |
| Удаление участников | ✅ | ❌ | ❌ |
| Изменение visibility (public/private) | ✅ | ❌ | ❌ |
| Удаление чата | ✅ | ❌ | ❌ |
| Покинуть чат | ✅ | ✅ | ❌ |

#### Публичный чат (IsPublic = true)

| Действие | Chat Admin | Chat Member | Workspace Member (не в чате) |
|----------|------------|-------------|------------------------------|
| Просмотр чата | ✅ | ✅ | ✅ (read-only) |
| Просмотр истории (ВСЯ) | ✅ | ✅ | ✅ |
| Присоединиться (Join) | — | — | ✅ → становится Member |
| Отправка сообщений | ✅ | ✅ | ❌ (нужен Join) |
| Применение тегов | ✅ | ✅ | ❌ (нужен Join) |
| Добавление участников | ✅ | ❌ | ❌ |
| Удаление участников | ✅ | ❌ | ❌ |
| Изменение visibility | ✅ | ❌ | ❌ |
| Удаление чата | ✅ | ❌ | ❌ |

**Специальное правило для публичных чатов:**
- Все workspace members видят публичные чаты на канбане
- Могут открыть и прочитать всю историю
- Для взаимодействия (сообщения, теги) нужно нажать "Join" → становятся Member

**Проверка прав на чат:**
```go
type ChatAccessLevel string

const (
    ChatAccessNone  ChatAccessLevel = "none"
    ChatAccessRead  ChatAccessLevel = "read"
    ChatAccessWrite ChatAccessLevel = "write"
    ChatAccessAdmin ChatAccessLevel = "admin"
)

func GetChatAccessLevel(userID UUID, chat *Chat, token *jwt.Token) ChatAccessLevel {
    // 1. System admin → admin access
    if IsSystemAdmin(token) {
        return ChatAccessAdmin
    }

    // 2. Проверяем workspace membership
    hasWorkspaceAccess := HasWorkspaceAccess(token, chat.WorkspaceID)
    if !hasWorkspaceAccess {
        return ChatAccessNone
    }

    // 3. Проверяем участие в чате
    participant := chat.FindParticipant(userID)
    if participant != nil {
        if participant.Role == ParticipantAdmin {
            return ChatAccessAdmin
        }
        return ChatAccessWrite
    }

    // 4. Публичный чат + workspace member → read access
    if chat.IsPublic {
        return ChatAccessRead
    }

    // 5. Приватный чат + не участник
    return ChatAccessNone
}
```

### Message Level

**Удаление сообщений:**

| Действие | Автор сообщения | Chat Admin | Chat Member | Workspace Member |
|----------|----------------|------------|-------------|------------------|
| Удалить своё сообщение (< 5 мин) | ✅ | ✅ | ❌ | ❌ |
| Удалить своё сообщение (> 5 мин) | ❌ | ✅ | ❌ | ❌ |
| Удалить чужое сообщение | ❌ | ✅ | ❌ | ❌ |

**Примечание:** Удаление сообщения с тегом НЕ откатывает изменения (см. Tag Grammar).

**Редактирование сообщений:**

| Действие | Автор сообщения | Chat Admin | Другие |
|----------|----------------|------------|--------|
| Редактировать своё сообщение (< 5 мин) | ✅ | ✅ | ❌ |
| Редактировать своё сообщение (> 5 мин) | ❌ | ✅ | ❌ |
| Редактировать чужое сообщение | ❌ | ❌ | ❌ |

**Ограничение:** Редактирование сообщения с тегами НЕ пересчитывает команды (MVP). Для изменения статуса нужно отправить новое сообщение.

---

## Workspace Management

### Создание Workspace (Self-Service)

**Процесс:**

```
1. User нажимает "Create Workspace" в UI
2. Frontend → POST /api/workspaces
   {
     "name": "My Team"
   }

3. Backend:
   a) Проверяет JWT токен (должен быть авторизован)
   b) Создаёт Group в Keycloak через Admin API:
      - Group name: "My Team"
      - Group attributes: { workspace_id: "generated-uuid" }
   c) Добавляет текущего пользователя в Group с ролью workspace-admin
   d) Создаёт Workspace в нашей БД:
      - ID: generated-uuid
      - KeycloakGroupID: keycloak-group-id
      - CreatedBy: current-user-id
   e) Возвращает Workspace

4. Frontend: Перенаправляет на /workspaces/{id}
```

**Ограничения (MVP):**
- Любой авторизованный пользователь может создать workspace
- Нет лимита на количество workspace
- V2: можно добавить квоты, подписки

### Приглашение пользователей (Invite Links)

**Архитектура:**

```
InviteLink:
├─ Token: "random-secure-token"
├─ WorkspaceID: UUID
├─ CreatedBy: UUID (workspace-admin)
├─ ExpiresAt: timestamp
├─ MaxUses: int (null = unlimited)
├─ UsedCount: int
└─ IsActive: bool
```

**Процесс генерации инвайта:**

```
1. Workspace Admin нажимает "Invite People"
2. Frontend → POST /api/workspaces/{id}/invites
   {
     "expiresIn": "7d",  // через 7 дней
     "maxUses": 10       // максимум 10 использований
   }

3. Backend:
   a) Проверяет: пользователь является workspace-admin
   b) Генерирует InviteLink:
      - token: crypto/rand 32 bytes → base64
      - expiresAt: now + 7 days
   c) Сохраняет в БД
   d) Возвращает URL: https://app.teams-up.com/invite/{token}

4. Frontend: Показывает URL для копирования
```

**Процесс использования инвайта:**

```
1. Новый пользователь переходит по ссылке: /invite/{token}

2. Если не авторизован:
   a) Редирект на Keycloak login/registration
   b) После успешной регистрации → редирект обратно на /invite/{token}

3. Если авторизован:
   a) Backend проверяет token:
      - Существует?
      - Не истёк?
      - Не превышен maxUses?
   b) Добавляет пользователя в Keycloak Group через Admin API
      - Group ID = workspace.KeycloakGroupID
      - Role = workspace-member
   c) Инкрементирует UsedCount
   d) Редирект на /workspaces/{id}

4. User теперь член workspace, видит публичные чаты
```

**Инвалидация инвайтов:**

```
Workspace Admin может:
- Деактивировать инвайт (IsActive = false)
- Удалить инвайт (soft delete)
- Просмотреть список активных инвайтов
- Просмотреть, кто использовал инвайт (audit log)
```

### Управление участниками

**Просмотр участников:**
```
GET /api/workspaces/{id}/members
→ Загружает участников из Keycloak Group
→ Показывает роли (admin/member)
```

**Удаление участника (только workspace-admin):**
```
DELETE /api/workspaces/{id}/members/{userId}
→ Удаляет пользователя из Keycloak Group через Admin API
→ Пользователь теряет доступ ко всем чатам workspace
```

**Назначение/снятие admin роли:**
```
PUT /api/workspaces/{id}/members/{userId}/role
{ "role": "workspace-admin" }
→ Добавляет Client Role "workspace-admin" пользователю в Group
```

---

## Работа с несколькими Workspace

### Текущий Workspace в сессии

**Подход: Cookie-based current workspace**

```go
// После логина пользователь попадает на страницу выбора workspace
GET /workspaces
→ Список всех workspace, к которым есть доступ

// Выбор workspace
POST /workspaces/{id}/select
→ Устанавливает cookie: current_workspace_id={id}
→ Редирект на /dashboard

// Переключение workspace
Navbar: [Dropdown: Current Workspace ▼]
  → Engineering Team (current)
  → Marketing Team
  → + Create New Workspace

При выборе другого workspace:
→ POST /workspaces/{id}/select
→ Cookie обновляется
→ Reload страницы (или SPA обновляет контекст)
```

**Middleware:**
```go
func CurrentWorkspaceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Читаем cookie
        cookie, err := r.Cookie("current_workspace_id")
        if err != nil {
            // Нет текущего workspace → редирект на выбор
            http.Redirect(w, r, "/workspaces", http.StatusFound)
            return
        }

        workspaceID := cookie.Value
        userID := GetUserIDFromContext(r.Context())

        // Проверяем доступ
        token := GetTokenFromContext(r.Context())
        hasAccess := HasWorkspaceAccess(token, workspaceID)
        if !hasAccess {
            // Потерян доступ → редирект на выбор
            http.Redirect(w, r, "/workspaces", http.StatusFound)
            return
        }

        // Добавляем workspace в контекст
        ctx := context.WithValue(r.Context(), "current_workspace_id", workspaceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**URL Structure:**

```
Вариант A: Workspace в URL (рекомендую)
/w/{workspaceID}/board
/w/{workspaceID}/chats/{chatID}
/w/{workspaceID}/settings

Вариант B: Workspace в cookie (текущий выбор)
/board
/chats/{chatID}
/settings
→ Всегда работаем в контексте current_workspace_id из cookie

Вариант C: Workspace в subdomain
engineering-team.teams-up.com/board
marketing-team.teams-up.com/board
```

**Предлагаю Вариант A для MVP:**
- Явная принадлежность к workspace в URL
- Можно открыть несколько workspace в разных вкладках
- Проще bookmarking и sharing ссылок

### Переключение Workspace

**UX Flow:**

```
User в workspace "Engineering Team":
  URL: /w/workspace-uuid-1/board

Navbar:
  [Engineering Team ▼]
    → Engineering Team ✓
    → Marketing Team
    → ──────────────
    → Create New Workspace

Click "Marketing Team":
  → Редирект на /w/workspace-uuid-2/board
  → Cookie current_workspace_id обновляется (опционально)
```

### Список Workspace

```go
GET /api/workspaces

Response:
[
  {
    "id": "workspace-uuid-1",
    "name": "Engineering Team",
    "role": "admin",  // workspace-admin
    "memberCount": 15,
    "unreadChats": 3
  },
  {
    "id": "workspace-uuid-2",
    "name": "Marketing Team",
    "role": "member",
    "memberCount": 8,
    "unreadChats": 0
  }
]
```

**Получение списка:**
```go
func (s *WorkspaceService) GetUserWorkspaces(userID UUID, token *jwt.Token) ([]Workspace, error) {
    // 1. Извлекаем groups из JWT
    groups := token.Get("groups").([]interface{})

    workspaces := []Workspace{}

    for _, groupPath := range groups {
        // 2. Для каждой группы загружаем workspace из БД
        //    (по KeycloakGroupID или через Keycloak API)
        workspace, err := s.repo.FindByKeycloakGroup(groupPath.(string))
        if err != nil {
            continue // Группа не связана с workspace (может быть другая)
        }

        // 3. Определяем роль пользователя в workspace
        role := s.getUserRoleInWorkspace(token, workspace.ID)

        workspace.Role = role
        workspaces = append(workspaces, workspace)
    }

    return workspaces, nil
}
```

---

## Keycloak Admin API Integration

### Конфигурация

```go
type KeycloakConfig struct {
    URL          string // http://localhost:8090
    Realm        string // teams-up
    ClientID     string // teams-up-app
    ClientSecret string
    AdminUser    string // admin (для service account)
    AdminPass    string
}
```

**Service Account для Admin API:**

```
В Keycloak создать отдельный Client для backend:
Client ID: teams-up-backend
Access Type: confidential
Service Accounts Enabled: ON
Authorization Enabled: ON

Service Account Roles:
  → realm-management:
     - manage-users
     - manage-groups
     - view-users
```

### Основные операции

#### Создание Group (Workspace)

```go
func (kc *KeycloakClient) CreateGroup(name string, attributes map[string][]string) (groupID string, error) {
    token := kc.getAdminToken()

    payload := map[string]interface{}{
        "name": name,
        "attributes": attributes,
    }

    resp, err := http.Post(
        fmt.Sprintf("%s/admin/realms/%s/groups", kc.URL, kc.Realm),
        "application/json",
        jsonBody(payload),
        withAuth(token),
    )

    if err != nil {
        return "", err
    }

    // Location header содержит URL с ID группы
    location := resp.Header.Get("Location")
    groupID = extractGroupIDFromLocation(location)

    return groupID, nil
}
```

#### Добавление пользователя в Group

```go
func (kc *KeycloakClient) AddUserToGroup(userID, groupID string) error {
    token := kc.getAdminToken()

    resp, err := http.Put(
        fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s",
            kc.URL, kc.Realm, userID, groupID),
        "application/json",
        nil,
        withAuth(token),
    )

    return err
}
```

#### Назначение Client Role пользователю в Group

```go
func (kc *KeycloakClient) AssignClientRoleToUserInGroup(
    userID, groupID, clientID, roleName string) error {

    token := kc.getAdminToken()

    // 1. Получить роль по имени
    role := kc.getClientRole(clientID, roleName)

    // 2. Назначить роль пользователю
    payload := []map[string]interface{}{
        {
            "id": role.ID,
            "name": role.Name,
        },
    }

    resp, err := http.Post(
        fmt.Sprintf("%s/admin/realms/%s/users/%s/role-mappings/clients/%s",
            kc.URL, kc.Realm, userID, clientID),
        "application/json",
        jsonBody(payload),
        withAuth(token),
    )

    return err
}
```

#### Удаление пользователя из Group

```go
func (kc *KeycloakClient) RemoveUserFromGroup(userID, groupID string) error {
    token := kc.getAdminToken()

    resp, err := http.Delete(
        fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s",
            kc.URL, kc.Realm, userID, groupID),
        withAuth(token),
    )

    return err
}
```

---

## Authentication Flow

### OAuth 2.0 Authorization Code Flow

```
1. User opens app → /
   ↓
2. Not authenticated → redirect to Keycloak
   GET http://localhost:8090/realms/teams-up/protocol/openid-connect/auth
     ?client_id=teams-up-app
     &redirect_uri=http://localhost:8080/auth/callback
     &response_type=code
     &scope=openid profile email

3. User logs in (or registers) at Keycloak
   ↓
4. Keycloak redirects back:
   GET http://localhost:8080/auth/callback?code=AUTH_CODE

5. Backend exchanges code for tokens:
   POST http://localhost:8090/realms/teams-up/protocol/openid-connect/token
     client_id=teams-up-app
     client_secret=SECRET
     code=AUTH_CODE
     grant_type=authorization_code
     redirect_uri=http://localhost:8080/auth/callback

   Response:
   {
     "access_token": "eyJhbG...",
     "refresh_token": "eyJhbG...",
     "id_token": "eyJhbG...",
     "expires_in": 3600,
     "token_type": "Bearer"
   }

6. Backend:
   a) Валидирует ID token
   b) Извлекает user info из токена
   c) Создаёт/обновляет User в БД
   d) Устанавливает session cookie
   e) Редирект на /workspaces (выбор workspace)
```

### Session Management

```go
// Session в cookie (httpOnly, secure)
type Session struct {
    SessionID    string
    UserID       UUID
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

// Хранение: Redis
Key: session:{sessionID}
Value: {UserID, AccessToken, RefreshToken, ExpiresAt}
TTL: 24 hours
```

**Middleware для проверки сессии:**

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Читаем session cookie
        cookie, err := r.Cookie("session_id")
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }

        // 2. Загружаем session из Redis
        session, err := sessionStore.Get(cookie.Value)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }

        // 3. Проверяем expiration
        if session.ExpiresAt.Before(time.Now()) {
            // Пытаемся обновить токен
            newToken, err := keycloakClient.RefreshToken(session.RefreshToken)
            if err != nil {
                // Не удалось обновить → требуется re-login
                http.Redirect(w, r, "/login", http.StatusFound)
                return
            }

            // Обновляем session
            session.AccessToken = newToken.AccessToken
            session.RefreshToken = newToken.RefreshToken
            session.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
            sessionStore.Update(session)
        }

        // 4. Парсим JWT токен
        token, err := jwt.Parse(session.AccessToken)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }

        // 5. Добавляем в context
        ctx := context.WithValue(r.Context(), "user_id", session.UserID)
        ctx = context.WithValue(ctx, "jwt_token", token)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Logout

```
POST /auth/logout
→ 1. Удаляем session из Redis
→ 2. Удаляем session cookie
→ 3. Опционально: отзываем токен в Keycloak
     POST http://localhost:8090/realms/teams-up/protocol/openid-connect/logout
→ 4. Редирект на /
```

---

## WebSocket Authentication

**Проблема:** WebSocket не поддерживает custom headers после handshake.

**Решение:** Передать токен при подключении.

```javascript
// Frontend
const wsURL = `ws://localhost:8080/ws?token=${accessToken}`;
const ws = new WebSocket(wsURL);
```

**Backend WebSocket handler:**

```go
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
    // 1. Извлекаем токен из query param
    tokenString := r.URL.Query().Get("token")
    if tokenString == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 2. Валидируем JWT
    token, err := jwt.Parse(tokenString)
    if err != nil {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }

    // 3. Извлекаем userID
    userID := token.Get("sub").(string)

    // 4. Upgrade connection
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    // 5. Регистрируем клиента
    client := &Client{
        ID:     uuid.New(),
        UserID: uuid.MustParse(userID),
        Conn:   conn,
        Token:  token,
    }

    h.hub.Register(client)

    // 6. Слушаем сообщения
    go client.ReadPump()
    go client.WritePump()
}
```

**Проверка прав при отправке сообщения через WS:**

```go
func (c *Client) handleMessage(msg WebSocketMessage) {
    switch msg.Type {
    case "chat.message":
        // Проверяем права на чат
        chat := chatRepo.FindByID(msg.ChatID)
        accessLevel := GetChatAccessLevel(c.UserID, chat, c.Token)

        if accessLevel != ChatAccessWrite && accessLevel != ChatAccessAdmin {
            c.sendError("Forbidden: You don't have write access to this chat")
            return
        }

        // Сохраняем сообщение
        messageService.PostMessage(msg.ChatID, c.UserID, msg.Content)
    }
}
```

---

## Security Best Practices

### 1. JWT Validation

```go
func ValidateJWT(tokenString string, config KeycloakConfig) (*jwt.Token, error) {
    // 1. Парсим без валидации (чтобы получить kid)
    token, _ := jwt.Parse(tokenString)

    // 2. Получаем kid (Key ID)
    kid := token.Header["kid"].(string)

    // 3. Загружаем публичный ключ из Keycloak JWKS endpoint
    jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
        config.URL, config.Realm)

    publicKey := getPublicKeyFromJWKS(jwksURL, kid)

    // 4. Валидируем подпись
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return publicKey, nil
    })

    if err != nil {
        return nil, err
    }

    // 5. Проверяем claims
    claims := token.Claims.(*CustomClaims)

    // Audience
    if !contains(claims.Audience, "teams-up-app") {
        return nil, errors.New("invalid audience")
    }

    // Issuer
    expectedIssuer := fmt.Sprintf("%s/realms/%s", config.URL, config.Realm)
    if claims.Issuer != expectedIssuer {
        return nil, errors.New("invalid issuer")
    }

    // Expiration
    if time.Unix(claims.ExpiresAt, 0).Before(time.Now()) {
        return nil, errors.New("token expired")
    }

    return token, nil
}
```

### 2. CORS Configuration

```go
corsConfig := cors.Config{
    AllowOrigins:     []string{"http://localhost:3000", "https://teams-up.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true, // для cookies
    MaxAge:           12 * time.Hour,
}
```

### 3. Rate Limiting

```go
// По IP + UserID
rateLimiter := middleware.RateLimiter{
    RequestsPerMinute: 60,
    BurstSize:         10,
}

// По endpoint
createWorkspaceLimit := middleware.RateLimiter{
    RequestsPerHour: 5, // не более 5 workspace в час
}
```

### 4. Input Validation

```go
type CreateWorkspaceRequest struct {
    Name string `json:"name" validate:"required,min=3,max=50"`
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateWorkspaceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Валидация
    if err := validate.Struct(req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Sanitization
    req.Name = sanitize(req.Name)

    // ...
}
```

### 5. Audit Logging

```go
type AuditLog struct {
    ID          UUID
    UserID      UUID
    Action      string // "workspace.created", "chat.deleted", "user.invited"
    ResourceID  UUID
    ResourceType string // "workspace", "chat", "invite"
    Details     JSONB
    IPAddress   string
    UserAgent   string
    Timestamp   time.Time
}

// Middleware для audit logging
func AuditMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := GetUserIDFromContext(r.Context())

        // Wrap response writer для захвата status code
        wrapped := &responseWriter{ResponseWriter: w}

        next.ServeHTTP(wrapped, r)

        // После выполнения запроса логируем
        if wrapped.statusCode >= 200 && wrapped.statusCode < 300 {
            auditLog := AuditLog{
                UserID:    userID,
                Action:    determineAction(r.Method, r.URL.Path),
                IPAddress: r.RemoteAddr,
                UserAgent: r.UserAgent(),
                Timestamp: time.Now(),
            }
            auditRepo.Save(auditLog)
        }
    })
}
```

---

## Резюме архитектурных решений

| Аспект | Решение | Обоснование |
|--------|---------|-------------|
| **User Management** | Keycloak | SSO, готовая система управления |
| **Роли** | Keycloak Realm & Client Roles | Централизованное управление |
| **Workspace** | Keycloak Groups | Естественный mapping |
| **Membership** | Group Membership | Синхронизация через JWT |
| **Авторизация** | JWT tokens (stateless) | Масштабируемость, простота |
| **Session** | Redis (access/refresh tokens) | Продление сессии без re-login |
| **Создание Workspace** | Self-service | Снижение friction |
| **Приглашения** | Invite links | Простота onboarding |
| **Множественные Workspace** | Cookie + URL | Удобство переключения |
| **WebSocket Auth** | Token в query param | Совместимость с WS |
| **Admin API** | Service Account | Безопасность |

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. ✅ Детальная грамматика тегов
4. ✅ Права доступа и security model
5. **TODO:** Event flow детально (обработка событий, retry, idempotency)
6. **TODO:** API контракты (HTTP + WebSocket)
7. **TODO:** Структура кода (внутри internal/)
8. **TODO:** План реализации MVP (roadmap)
