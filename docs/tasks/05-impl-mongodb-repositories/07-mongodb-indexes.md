# Task 07: MongoDB Indexes

**Status: ✅ Complete**

## Цель

Создать оптимальные индексы для всех MongoDB коллекций, обеспечивающие высокую производительность запросов.

## Контекст

Индексы критически важны для производительности MongoDB. Без правильных индексов запросы будут выполнять collection scan, что неприемлемо для production нагрузки.

Индексы должны создаваться:
1. При первом запуске приложения (миграции)
2. В тестах для корректной эмуляции production среды

## Реализация

### Расположение файлов

```
internal/infrastructure/mongodb/
├── indexes.go          # Определения всех индексов
└── indexes_test.go     # Тесты (16 тестов)

tests/testutil/
├── mongodb_shared.go   # Интеграция с CreateAllIndexes
└── db.go               # Интеграция с CreateAllIndexes
```

### Основные функции

| Функция | Описание |
|---------|----------|
| `CreateAllIndexes(ctx, db)` | Создает все индексы (идемпотентная) |
| `EnsureIndexes(ctx, db)` | Алиас для `CreateAllIndexes` |
| `CreateCollectionIndexes(ctx, db, name)` | Создает индексы для конкретной коллекции |
| `GetAllIndexDefinitions()` | Возвращает все определения индексов |

### Коллекции и индексы

#### 1. Events Collection (Event Store) — 3 индекса

| Поля | Тип | Имя |
|------|-----|-----|
| `aggregate_id`, `version` | unique | `idx_events_aggregate_version_unique` |
| `event_type`, `occurred_at` | compound | `idx_events_type_time` |
| `aggregate_type`, `occurred_at` | compound | `idx_events_aggregate_type_time` |

#### 2. Users Collection — 6 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `user_id` | unique | `idx_users_id_unique` |
| `username` | unique | `idx_users_username_unique` |
| `email` | unique | `idx_users_email_unique` |
| `keycloak_id` | unique, sparse | `idx_users_keycloak_unique` |
| `display_name` | regular | `idx_users_display_name` |
| `is_system_admin` | regular | `idx_users_system_admin` |

#### 3. Workspaces Collection — 5 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `workspace_id` | unique | `idx_workspaces_id_unique` |
| `keycloak_group_id` | unique, sparse | `idx_workspaces_keycloak_unique` |
| `name` | regular | `idx_workspaces_name` |
| `created_by` | regular | `idx_workspaces_created_by` |
| `invites.token` | regular | `idx_workspaces_invite_token` |

#### 4. Workspace Members Collection — 3 индекса

| Поля | Тип | Имя |
|------|-----|-----|
| `user_id`, `workspace_id` | unique, compound | `idx_members_user_workspace_unique` |
| `workspace_id` | regular | `idx_members_workspace` |
| `user_id` | regular | `idx_members_user` |

#### 5. Chat Read Model Collection — 9 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `chat_id` | unique | `idx_chats_id_unique` |
| `workspace_id`, `created_at` | compound | `idx_chats_workspace_time` |
| `workspace_id`, `type`, `created_at` | compound | `idx_chats_workspace_type_time` |
| `workspace_id`, `is_public` | compound | `idx_chats_workspace_public` |
| `participants` | regular (array) | `idx_chats_participants` |
| `created_by` | regular | `idx_chats_created_by` |
| `assigned_to` | sparse | `idx_chats_assignee` |
| `status` | sparse | `idx_chats_status` |
| `workspace_id`, `type`, `status`, `assigned_to` | compound | `idx_chats_task_filter` |

#### 6. Task Read Model Collection — 9 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `task_id` | unique | `idx_tasks_id_unique` |
| `chat_id` | unique | `idx_tasks_chat_unique` |
| `assigned_to`, `status` | compound, sparse | `idx_tasks_assignee_status` |
| `status`, `priority` | compound | `idx_tasks_status_priority` |
| `entity_type` | regular | `idx_tasks_entity_type` |
| `created_by` | regular | `idx_tasks_created_by` |
| `created_at` | regular | `idx_tasks_created_at` |
| `due_date` | sparse | `idx_tasks_due_date` |
| `assigned_to`, `status`, `due_date` | compound | `idx_tasks_dashboard` |

#### 7. Messages Collection — 7 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `message_id` | unique | `idx_messages_id_unique` |
| `chat_id`, `created_at` | compound | `idx_messages_chat_time` |
| `parent_id`, `created_at` | compound, sparse | `idx_messages_thread` |
| `sent_by`, `created_at` | compound | `idx_messages_author_time` |
| `chat_id`, `is_deleted`, `created_at` | compound | `idx_messages_chat_active` |
| `content` | text | `idx_messages_content_text` |
| `chat_id`, `sent_by` | compound | `idx_messages_chat_author` |

> **Примечание**: Поля `parent_id` и `sent_by` соответствуют фактическим полям в `messageDocument` (не `parent_message_id` и `author_id`).

#### 8. Notifications Collection — 6 индексов

| Поля | Тип | Имя |
|------|-----|-----|
| `notification_id` | unique | `idx_notifications_id_unique` |
| `user_id`, `created_at` | compound | `idx_notifications_user_time` |
| `user_id`, `read_at`, `created_at` | compound | `idx_notifications_user_unread` |
| `user_id`, `type`, `created_at` | compound | `idx_notifications_user_type` |
| `resource_id` | sparse | `idx_notifications_resource` |
| `created_at` | regular | `idx_notifications_cleanup` |

> **Примечание**: Поле `read_at` (nullable timestamp) используется вместо `is_read` boolean. NULL означает непрочитанное.

### Общая статистика

| Коллекция | Кол-во индексов |
|-----------|-----------------|
| events | 3 |
| users | 6 |
| workspaces | 5 |
| workspace_members | 3 |
| chat_read_model | 9 |
| task_read_model | 9 |
| messages | 7 |
| notifications | 6 |
| **Итого** | **48** |

## Использование

### В приложении (миграции)

```go
package main

import (
    "context"
    "log"

    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

func main() {
    ctx := context.Background()

    client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    defer client.Disconnect(ctx)

    db := client.Database("flowra")

    log.Println("Creating indexes...")
    if err := mongodb.CreateAllIndexes(ctx, db); err != nil {
        log.Fatalf("Failed to create indexes: %v", err)
    }

    log.Println("Indexes created successfully")
}
```

### В тестах

Индексы создаются автоматически при использовании `testutil.SetupTestMongoDB()`:

```go
func TestSomething(t *testing.T) {
    db := testutil.SetupTestMongoDB(t)
    // Индексы уже созданы!
    
    // ... тест
}
```

## Проверка индексов

### Команды MongoDB Shell

```javascript
// Показать все индексы коллекции
db.users.getIndexes()

// Проверить использование индекса в запросе
db.users.find({ username: "test" }).explain("executionStats")

// Показать статистику индексов
db.users.aggregate([{ $indexStats: {} }])
```

### Ожидаемый результат explain

```json
{
    "winningPlan": {
        "stage": "FETCH",
        "inputStage": {
            "stage": "IXSCAN",
            "indexName": "idx_users_username_unique"
        }
    }
}
```

## Checklist

### Phase 1: Event Store

- [x] Создать unique index для aggregate_id + version
- [x] Создать index для event_type + occurred_at
- [x] Создать index для aggregate_type + occurred_at

### Phase 2: Users

- [x] Создать unique index для user_id
- [x] Создать unique index для username
- [x] Создать unique index для email
- [x] Создать sparse unique index для keycloak_id

### Phase 3: Workspaces

- [x] Создать unique index для workspace_id
- [x] Создать sparse unique index для keycloak_group_id
- [x] Создать index для members.user_id (в отдельной коллекции workspace_members)
- [x] Создать index для invites.token

### Phase 4: Chat Read Model

- [x] Создать unique index для chat_id
- [x] Создать compound index для workspace_id + created_at
- [x] Создать index для participants
- [x] Создать compound index для фильтрации

### Phase 5: Task Read Model

- [x] Создать unique index для task_id
- [x] Создать unique index для chat_id
- [x] Создать index для assigned_to + status
- [x] Создать index для status + priority

### Phase 6: Messages

- [x] Создать unique index для message_id
- [x] Создать compound index для chat_id + created_at
- [x] Создать sparse index для parent_id
- [x] Создать text index для content

### Phase 7: Notifications

- [x] Создать unique index для notification_id
- [x] Создать compound index для user_id + created_at
- [x] Создать compound index для user_id + read_at + created_at
- [x] Создать index для cleanup (created_at)

### Phase 8: Реализация

- [x] Создать `internal/infrastructure/mongodb/indexes.go`
- [x] Добавить функцию `CreateAllIndexes`
- [ ] Интегрировать в migrator (cmd/migrator пока не реализован)
- [x] Интегрировать в тесты (testutil/mongodb_shared.go, testutil/db.go)

## Следующие шаги

После завершения этой задачи:

1. **Task 08** — интеграционные тесты с testcontainers

## Референсы

- [MongoDB Index Types](https://www.mongodb.com/docs/manual/indexes/)
- [Compound Indexes](https://www.mongodb.com/docs/manual/core/index-compound/)
- [Index Build Operations](https://www.mongodb.com/docs/manual/core/index-creation/)
- [TTL Indexes](https://www.mongodb.com/docs/manual/core/index-ttl/)