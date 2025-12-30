# Task 07: MongoDB Indexes

## Цель

Создать оптимальные индексы для всех MongoDB коллекций, обеспечивающие высокую производительность запросов.

## Контекст

Индексы критически важны для производительности MongoDB. Без правильных индексов запросы будут выполнять collection scan, что неприемлемо для production нагрузки.

Индексы должны создаваться:
1. При первом запуске приложения (миграции)
2. В тестах для корректной эмуляции production среды

## Коллекции и индексы

### 1. Events Collection (Event Store)

Коллекция для хранения событий Event Sourcing.

```javascript
// Уникальный индекс для предотвращения дублирования событий
db.events.createIndex(
    { "aggregate_id": 1, "version": 1 },
    { unique: true, name: "idx_events_aggregate_version_unique" }
)

// Индекс для загрузки событий агрегата
db.events.createIndex(
    { "aggregate_id": 1, "version": 1 },
    { name: "idx_events_aggregate_load" }
)

// Индекс для фильтрации по типу события
db.events.createIndex(
    { "event_type": 1, "occurred_at": -1 },
    { name: "idx_events_type_time" }
)

// Индекс для фильтрации по типу агрегата
db.events.createIndex(
    { "aggregate_type": 1, "occurred_at": -1 },
    { name: "idx_events_aggregate_type_time" }
)
```

### 2. Users Collection

```javascript
// Primary key
db.users.createIndex(
    { "user_id": 1 },
    { unique: true, name: "idx_users_id_unique" }
)

// Уникальный индекс для username
db.users.createIndex(
    { "username": 1 },
    { unique: true, name: "idx_users_username_unique" }
)

// Уникальный индекс для email
db.users.createIndex(
    { "email": 1 },
    { unique: true, name: "idx_users_email_unique" }
)

// Индекс для Keycloak ID (sparse - не индексирует null)
db.users.createIndex(
    { "keycloak_id": 1 },
    { unique: true, sparse: true, name: "idx_users_keycloak_unique" }
)

// Индекс для поиска по display_name
db.users.createIndex(
    { "display_name": 1 },
    { name: "idx_users_display_name" }
)

// Индекс для системных администраторов
db.users.createIndex(
    { "is_system_admin": 1 },
    { name: "idx_users_system_admin" }
)
```

### 3. Workspaces Collection

```javascript
// Primary key
db.workspaces.createIndex(
    { "workspace_id": 1 },
    { unique: true, name: "idx_workspaces_id_unique" }
)

// Уникальный индекс для Keycloak группы
db.workspaces.createIndex(
    { "keycloak_group_id": 1 },
    { unique: true, sparse: true, name: "idx_workspaces_keycloak_unique" }
)

// Индекс для поиска по имени
db.workspaces.createIndex(
    { "name": 1 },
    { name: "idx_workspaces_name" }
)

// Индекс для поиска по создателю
db.workspaces.createIndex(
    { "created_by": 1 },
    { name: "idx_workspaces_created_by" }
)

// Индекс для поиска по членам
db.workspaces.createIndex(
    { "members.user_id": 1 },
    { name: "idx_workspaces_members" }
)

// Индекс для поиска приглашений по токену
db.workspaces.createIndex(
    { "invites.token": 1 },
    { name: "idx_workspaces_invite_token" }
)

// Индекс для поиска приглашений по email
db.workspaces.createIndex(
    { "invites.email": 1 },
    { name: "idx_workspaces_invite_email" }
)
```

### 4. Chat Read Model Collection

```javascript
// Primary key
db.chat_read_model.createIndex(
    { "chat_id": 1 },
    { unique: true, name: "idx_chats_id_unique" }
)

// Индекс для workspace
db.chat_read_model.createIndex(
    { "workspace_id": 1, "created_at": -1 },
    { name: "idx_chats_workspace_time" }
)

// Индекс для типа чата
db.chat_read_model.createIndex(
    { "workspace_id": 1, "type": 1, "created_at": -1 },
    { name: "idx_chats_workspace_type_time" }
)

// Индекс для публичных чатов
db.chat_read_model.createIndex(
    { "workspace_id": 1, "is_public": 1 },
    { name: "idx_chats_workspace_public" }
)

// Индекс для участников
db.chat_read_model.createIndex(
    { "participants": 1 },
    { name: "idx_chats_participants" }
)

// Индекс для создателя
db.chat_read_model.createIndex(
    { "created_by": 1 },
    { name: "idx_chats_created_by" }
)

// Индекс для assignee
db.chat_read_model.createIndex(
    { "assigned_to": 1 },
    { sparse: true, name: "idx_chats_assignee" }
)

// Индекс для статуса
db.chat_read_model.createIndex(
    { "status": 1 },
    { sparse: true, name: "idx_chats_status" }
)

// Compound index для фильтрации задач
db.chat_read_model.createIndex(
    { "workspace_id": 1, "type": 1, "status": 1, "assigned_to": 1 },
    { name: "idx_chats_task_filter" }
)
```

### 5. Task Read Model Collection

```javascript
// Primary key
db.task_read_model.createIndex(
    { "task_id": 1 },
    { unique: true, name: "idx_tasks_id_unique" }
)

// Уникальный индекс для chat_id (one task per chat)
db.task_read_model.createIndex(
    { "chat_id": 1 },
    { unique: true, name: "idx_tasks_chat_unique" }
)

// Индекс для assignee
db.task_read_model.createIndex(
    { "assigned_to": 1, "status": 1 },
    { sparse: true, name: "idx_tasks_assignee_status" }
)

// Индекс для статуса
db.task_read_model.createIndex(
    { "status": 1, "priority": 1 },
    { name: "idx_tasks_status_priority" }
)

// Индекс для типа сущности
db.task_read_model.createIndex(
    { "entity_type": 1 },
    { name: "idx_tasks_entity_type" }
)

// Индекс для создателя
db.task_read_model.createIndex(
    { "created_by": 1 },
    { name: "idx_tasks_created_by" }
)

// Индекс для сортировки по времени
db.task_read_model.createIndex(
    { "created_at": -1 },
    { name: "idx_tasks_created_at" }
)

// Индекс для дедлайнов
db.task_read_model.createIndex(
    { "due_date": 1 },
    { sparse: true, name: "idx_tasks_due_date" }
)

// Compound index для дашборда
db.task_read_model.createIndex(
    { "assigned_to": 1, "status": 1, "due_date": 1 },
    { name: "idx_tasks_dashboard" }
)
```

### 6. Messages Collection

```javascript
// Primary key
db.messages.createIndex(
    { "message_id": 1 },
    { unique: true, name: "idx_messages_id_unique" }
)

// Основной индекс для загрузки сообщений чата
db.messages.createIndex(
    { "chat_id": 1, "created_at": -1 },
    { name: "idx_messages_chat_time" }
)

// Индекс для тредов
db.messages.createIndex(
    { "parent_message_id": 1, "created_at": 1 },
    { sparse: true, name: "idx_messages_thread" }
)

// Индекс для автора
db.messages.createIndex(
    { "author_id": 1, "created_at": -1 },
    { name: "idx_messages_author_time" }
)

// Compound index для фильтрации не удаленных сообщений
db.messages.createIndex(
    { "chat_id": 1, "is_deleted": 1, "created_at": -1 },
    { name: "idx_messages_chat_active" }
)

// Text index для полнотекстового поиска
db.messages.createIndex(
    { "content": "text" },
    { name: "idx_messages_content_text", default_language: "russian" }
)

// Compound index для поиска по автору в чате
db.messages.createIndex(
    { "chat_id": 1, "author_id": 1 },
    { name: "idx_messages_chat_author" }
)
```

### 7. Notifications Collection

```javascript
// Primary key
db.notifications.createIndex(
    { "notification_id": 1 },
    { unique: true, name: "idx_notifications_id_unique" }
)

// Основной индекс для загрузки уведомлений пользователя
db.notifications.createIndex(
    { "user_id": 1, "created_at": -1 },
    { name: "idx_notifications_user_time" }
)

// Индекс для непрочитанных уведомлений
db.notifications.createIndex(
    { "user_id": 1, "is_read": 1, "created_at": -1 },
    { name: "idx_notifications_user_unread" }
)

// Индекс для фильтрации по типу
db.notifications.createIndex(
    { "user_id": 1, "type": 1, "created_at": -1 },
    { name: "idx_notifications_user_type" }
)

// Индекс для связанного ресурса
db.notifications.createIndex(
    { "resource_id": 1, "resource_type": 1 },
    { sparse: true, name: "idx_notifications_resource" }
)

// Индекс для очистки старых уведомлений
db.notifications.createIndex(
    { "created_at": 1 },
    { name: "idx_notifications_cleanup" }
)

// TTL индекс для автоматического удаления (опционально)
// Удаляет уведомления старше 90 дней
// db.notifications.createIndex(
//     { "created_at": 1 },
//     { expireAfterSeconds: 7776000, name: "idx_notifications_ttl" }
// )
```

## Реализация в Go

### Файл миграции индексов

Создать `internal/infrastructure/mongodb/indexes.go`:

```go
package mongodb

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// IndexDefinition описывает индекс для создания
type IndexDefinition struct {
    Collection string
    Keys       bson.D
    Options    *options.IndexOptionsBuilder
}

// CreateAllIndexes создает все необходимые индексы
func CreateAllIndexes(ctx context.Context, db *mongo.Database) error {
    indexes := getAllIndexDefinitions()

    for _, idx := range indexes {
        coll := db.Collection(idx.Collection)
        model := mongo.IndexModel{
            Keys:    idx.Keys,
            Options: idx.Options,
        }

        _, err := coll.Indexes().CreateOne(ctx, model)
        if err != nil {
            return fmt.Errorf("failed to create index on %s: %w", idx.Collection, err)
        }
    }

    return nil
}

// getAllIndexDefinitions возвращает все определения индексов
func getAllIndexDefinitions() []IndexDefinition {
    var indexes []IndexDefinition

    // Events
    indexes = append(indexes, getEventIndexes()...)

    // Users
    indexes = append(indexes, getUserIndexes()...)

    // Workspaces
    indexes = append(indexes, getWorkspaceIndexes()...)

    // Chat Read Model
    indexes = append(indexes, getChatReadModelIndexes()...)

    // Task Read Model
    indexes = append(indexes, getTaskReadModelIndexes()...)

    // Messages
    indexes = append(indexes, getMessageIndexes()...)

    // Notifications
    indexes = append(indexes, getNotificationIndexes()...)

    return indexes
}

func getEventIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "events",
            Keys:       bson.D{{Key: "aggregate_id", Value: 1}, {Key: "version", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_events_aggregate_version_unique"),
        },
        {
            Collection: "events",
            Keys:       bson.D{{Key: "event_type", Value: 1}, {Key: "occurred_at", Value: -1}},
            Options:    options.Index().SetName("idx_events_type_time"),
        },
        {
            Collection: "events",
            Keys:       bson.D{{Key: "aggregate_type", Value: 1}, {Key: "occurred_at", Value: -1}},
            Options:    options.Index().SetName("idx_events_aggregate_type_time"),
        },
    }
}

func getUserIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "users",
            Keys:       bson.D{{Key: "user_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_users_id_unique"),
        },
        {
            Collection: "users",
            Keys:       bson.D{{Key: "username", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_users_username_unique"),
        },
        {
            Collection: "users",
            Keys:       bson.D{{Key: "email", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_users_email_unique"),
        },
        {
            Collection: "users",
            Keys:       bson.D{{Key: "keycloak_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetSparse(true).SetName("idx_users_keycloak_unique"),
        },
    }
}

func getWorkspaceIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "workspaces",
            Keys:       bson.D{{Key: "workspace_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_workspaces_id_unique"),
        },
        {
            Collection: "workspaces",
            Keys:       bson.D{{Key: "keycloak_group_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetSparse(true).SetName("idx_workspaces_keycloak_unique"),
        },
        {
            Collection: "workspaces",
            Keys:       bson.D{{Key: "members.user_id", Value: 1}},
            Options:    options.Index().SetName("idx_workspaces_members"),
        },
        {
            Collection: "workspaces",
            Keys:       bson.D{{Key: "invites.token", Value: 1}},
            Options:    options.Index().SetName("idx_workspaces_invite_token"),
        },
    }
}

func getChatReadModelIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "chat_read_model",
            Keys:       bson.D{{Key: "chat_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_chats_id_unique"),
        },
        {
            Collection: "chat_read_model",
            Keys:       bson.D{{Key: "workspace_id", Value: 1}, {Key: "created_at", Value: -1}},
            Options:    options.Index().SetName("idx_chats_workspace_time"),
        },
        {
            Collection: "chat_read_model",
            Keys:       bson.D{{Key: "participants", Value: 1}},
            Options:    options.Index().SetName("idx_chats_participants"),
        },
        {
            Collection: "chat_read_model",
            Keys:       bson.D{{Key: "workspace_id", Value: 1}, {Key: "type", Value: 1}, {Key: "status", Value: 1}},
            Options:    options.Index().SetName("idx_chats_workspace_type_status"),
        },
    }
}

func getTaskReadModelIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "task_read_model",
            Keys:       bson.D{{Key: "task_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_tasks_id_unique"),
        },
        {
            Collection: "task_read_model",
            Keys:       bson.D{{Key: "chat_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_tasks_chat_unique"),
        },
        {
            Collection: "task_read_model",
            Keys:       bson.D{{Key: "assigned_to", Value: 1}, {Key: "status", Value: 1}},
            Options:    options.Index().SetSparse(true).SetName("idx_tasks_assignee_status"),
        },
        {
            Collection: "task_read_model",
            Keys:       bson.D{{Key: "status", Value: 1}, {Key: "priority", Value: 1}},
            Options:    options.Index().SetName("idx_tasks_status_priority"),
        },
    }
}

func getMessageIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "messages",
            Keys:       bson.D{{Key: "message_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_messages_id_unique"),
        },
        {
            Collection: "messages",
            Keys:       bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: -1}},
            Options:    options.Index().SetName("idx_messages_chat_time"),
        },
        {
            Collection: "messages",
            Keys:       bson.D{{Key: "parent_message_id", Value: 1}, {Key: "created_at", Value: 1}},
            Options:    options.Index().SetSparse(true).SetName("idx_messages_thread"),
        },
        {
            Collection: "messages",
            Keys:       bson.D{{Key: "chat_id", Value: 1}, {Key: "is_deleted", Value: 1}, {Key: "created_at", Value: -1}},
            Options:    options.Index().SetName("idx_messages_chat_active"),
        },
    }
}

func getNotificationIndexes() []IndexDefinition {
    return []IndexDefinition{
        {
            Collection: "notifications",
            Keys:       bson.D{{Key: "notification_id", Value: 1}},
            Options:    options.Index().SetUnique(true).SetName("idx_notifications_id_unique"),
        },
        {
            Collection: "notifications",
            Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
            Options:    options.Index().SetName("idx_notifications_user_time"),
        },
        {
            Collection: "notifications",
            Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "is_read", Value: 1}, {Key: "created_at", Value: -1}},
            Options:    options.Index().SetName("idx_notifications_user_unread"),
        },
        {
            Collection: "notifications",
            Keys:       bson.D{{Key: "created_at", Value: 1}},
            Options:    options.Index().SetName("idx_notifications_cleanup"),
        },
    }
}
```

### Использование в миграциях

```go
// cmd/migrator/main.go
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

### Использование в тестах

```go
// tests/testutil/mongodb.go
func SetupMongoDB(t *testing.T) (*mongo.Client, func()) {
    // ... setup testcontainer ...

    // Create indexes
    db := client.Database("test_db")
    if err := mongodb.CreateAllIndexes(ctx, db); err != nil {
        t.Fatalf("Failed to create indexes: %v", err)
    }

    return client, cleanup
}
```

## Проверка индексов

### Команды для проверки

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

- [ ] Создать unique index для aggregate_id + version
- [ ] Создать index для event_type + occurred_at
- [ ] Создать index для aggregate_type + occurred_at

### Phase 2: Users

- [ ] Создать unique index для user_id
- [ ] Создать unique index для username
- [ ] Создать unique index для email
- [ ] Создать sparse unique index для keycloak_id

### Phase 3: Workspaces

- [ ] Создать unique index для workspace_id
- [ ] Создать sparse unique index для keycloak_group_id
- [ ] Создать index для members.user_id
- [ ] Создать index для invites.token

### Phase 4: Chat Read Model

- [ ] Создать unique index для chat_id
- [ ] Создать compound index для workspace_id + created_at
- [ ] Создать index для participants
- [ ] Создать compound index для фильтрации

### Phase 5: Task Read Model

- [ ] Создать unique index для task_id
- [ ] Создать unique index для chat_id
- [ ] Создать index для assigned_to + status
- [ ] Создать index для status + priority

### Phase 6: Messages

- [ ] Создать unique index для message_id
- [ ] Создать compound index для chat_id + created_at
- [ ] Создать sparse index для parent_message_id
- [ ] Создать text index для content

### Phase 7: Notifications

- [ ] Создать unique index для notification_id
- [ ] Создать compound index для user_id + created_at
- [ ] Создать compound index для user_id + is_read + created_at
- [ ] Создать index для cleanup (created_at)

### Phase 8: Реализация

- [ ] Создать `internal/infrastructure/mongodb/indexes.go`
- [ ] Добавить функцию `CreateAllIndexes`
- [ ] Интегрировать в migrator
- [ ] Интегрировать в тесты

## Следующие шаги

После завершения этой задачи:

1. **Task 08** — интеграционные тесты с testcontainers

## Референсы

- [MongoDB Index Types](https://www.mongodb.com/docs/manual/indexes/)
- [Compound Indexes](https://www.mongodb.com/docs/manual/core/index-compound/)
- [Index Build Operations](https://www.mongodb.com/docs/manual/core/index-creation/)
- [TTL Indexes](https://www.mongodb.com/docs/manual/core/index-ttl/)