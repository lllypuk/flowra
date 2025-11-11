# Task 1.1.2: MongoDB Repositories

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** Blocked
**Время:** 5-6 дней
**Зависимости:** Task 1.1.1 (MongoDB Event Store)

---

## Проблема

Repository interfaces определены, но MongoDB implementations отсутствуют. Use cases не могут сохранять/загружать данные.

**Нужно реализовать:**
- ChatRepository (event sourcing + read model)
- MessageRepository
- UserRepository
- WorkspaceRepository
- NotificationRepository

---

## Цель

Реализовать все repository interfaces с MongoDB persistence, indexes, и query optimization.

---

## Файлы для создания

```
internal/infrastructure/repository/mongodb/
├── chat_repository.go           (Chat with event sourcing)
├── chat_repository_test.go
├── message_repository.go        (Message CRUD)
├── message_repository_test.go
├── user_repository.go           (User CRUD)
├── user_repository_test.go
├── workspace_repository.go      (Workspace + members)
├── workspace_repository_test.go
├── notification_repository.go   (Notification CRUD)
├── notification_repository_test.go
└── common.go                    (shared utilities)

migrations/mongodb/
├── 002_chat_read_model.js
├── 003_messages.js
├── 004_users.js
├── 005_workspaces.js
└── 006_notifications.js
```

---

## Детальный план реализации

### 1. ChatRepository (chat_repository.go)

**Особенность:** Event sourcing для write, read model для queries.

#### Collections

**events** - уже создана (Task 1.1.1)

**chat_read_model** - denormalized для быстрых queries
```javascript
// migrations/mongodb/002_chat_read_model.js

db.createCollection("chat_read_model");

db.chat_read_model.createIndex(
    { chat_id: 1 },
    { unique: true, name: "chat_id_unique" }
);

db.chat_read_model.createIndex(
    { workspace_id: 1, type: 1 },
    { name: "workspace_type" }
);

db.chat_read_model.createIndex(
    { participants: 1 },
    { name: "participants" }
);

db.chat_read_model.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);
```

#### Implementation

```go
package mongodb

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"

    "github.com/lllypuk/flowra/internal/application/shared"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

type MongoChatRepository struct {
    eventStore     shared.EventStore
    readModelColl  *mongo.Collection
}

func NewMongoChatRepository(client *mongo.Client, db string, eventStore shared.EventStore) *MongoChatRepository {
    return &MongoChatRepository{
        eventStore:    eventStore,
        readModelColl: client.Database(db).Collection("chat_read_model"),
    }
}

// Load - reconstruct aggregate from events (event sourcing)
func (r *MongoChatRepository) Load(ctx context.Context, chatID uuid.UUID) (*chatdomain.Chat, error) {
    events, err := r.eventStore.LoadEvents(ctx, chatID)
    if err != nil {
        return nil, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return nil, &shared.NotFoundError{Resource: "Chat", ID: chatID}
    }

    // Reconstruct aggregate from events
    chat := &chatdomain.Chat{}
    for _, event := range events {
        chat.ApplyEvent(event)
    }

    return chat, nil
}

// Save - save aggregate (append events + update read model)
func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
    uncommittedEvents := chat.UncommittedEvents()
    if len(uncommittedEvents) == 0 {
        return nil  // nothing to save
    }

    // 1. Save events to event store
    err := r.eventStore.SaveEvents(ctx, chat.ID(), uncommittedEvents, chat.Version()-len(uncommittedEvents))
    if err != nil {
        return fmt.Errorf("failed to save events: %w", err)
    }

    // 2. Update read model (denormalized)
    err = r.updateReadModel(ctx, chat)
    if err != nil {
        // Log error but don't fail (read model can be rebuilt)
        // TODO: add logging
    }

    // 3. Clear uncommitted events
    chat.ClearUncommittedEvents()

    return nil
}

// FindByWorkspace - query read model
func (r *MongoChatRepository) FindByWorkspace(
    ctx context.Context,
    workspaceID uuid.UUID,
    chatType *chatdomain.ChatType,
    limit, offset int,
) ([]chatdomain.Chat, error) {
    filter := bson.M{"workspace_id": workspaceID.String()}

    if chatType != nil {
        filter["type"] = string(*chatType)
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))

    cursor, err := r.readModelColl.Find(ctx, filter, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to find chats: %w", err)
    }
    defer cursor.Close(ctx)

    var chats []chatdomain.Chat
    for cursor.Next(ctx) {
        var doc chatReadModelDoc
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }

        // Reconstruct from events (or use read model snapshot)
        chat, err := r.Load(ctx, uuid.MustParse(doc.ChatID))
        if err != nil {
            continue  // skip failed loads
        }

        chats = append(chats, *chat)
    }

    return chats, nil
}

// updateReadModel - update denormalized read model
func (r *MongoChatRepository) updateReadModel(ctx context.Context, chat *chatdomain.Chat) error {
    doc := chatReadModelDoc{
        ChatID:       chat.ID().String(),
        WorkspaceID:  chat.WorkspaceID().String(),
        Type:         string(chat.Type()),
        Title:        chat.Title(),
        IsPublic:     chat.IsPublic(),
        CreatedBy:    chat.CreatedBy().String(),
        CreatedAt:    chat.CreatedAt(),
        Participants: mapParticipants(chat.Participants()),
    }

    // Task-specific fields
    if chat.Type() == chatdomain.ChatTypeTask || chat.Type() == chatdomain.ChatTypeBug {
        doc.Status = (*string)(&chat.Status())
        if assignedTo := chat.AssignedTo(); assignedTo != nil {
            assignedToStr := assignedTo.String()
            doc.AssignedTo = &assignedToStr
        }
        // ... other fields
    }

    filter := bson.M{"chat_id": chat.ID().String()}
    update := bson.M{"$set": doc}
    opts := options.Update().SetUpsert(true)

    _, err := r.readModelColl.UpdateOne(ctx, filter, update, opts)
    return err
}

type chatReadModelDoc struct {
    ChatID       string    `bson:"chat_id"`
    WorkspaceID  string    `bson:"workspace_id"`
    Type         string    `bson:"type"`
    Title        string    `bson:"title"`
    IsPublic     bool      `bson:"is_public"`
    CreatedBy    string    `bson:"created_by"`
    CreatedAt    time.Time `bson:"created_at"`
    Participants []string  `bson:"participants"`
    Status       *string   `bson:"status,omitempty"`
    AssignedTo   *string   `bson:"assigned_to,omitempty"`
}
```

---

### 2. MessageRepository (message_repository.go)

**Collection: messages**

```javascript
// migrations/mongodb/003_messages.js

db.createCollection("messages");

db.messages.createIndex(
    { message_id: 1 },
    { unique: true, name: "message_id_unique" }
);

db.messages.createIndex(
    { chat_id: 1, created_at: -1 },
    { name: "chat_created" }
);

db.messages.createIndex(
    { parent_id: 1, created_at: 1 },
    { name: "parent_created", sparse: true }
);

db.messages.createIndex(
    { sent_by: 1 },
    { name: "sent_by" }
);
```

```go
type MongoMessageRepository struct {
    collection *mongo.Collection
}

func (r *MongoMessageRepository) FindByID(ctx context.Context, messageID uuid.UUID) (*messagedomain.Message, error) {
    // ...
}

func (r *MongoMessageRepository) FindByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]messagedomain.Message, error) {
    // ...
}

func (r *MongoMessageRepository) Save(ctx context.Context, msg *messagedomain.Message) error {
    // ...
}
```

---

### 3. UserRepository (user_repository.go)

**Collection: users**

```javascript
// migrations/mongodb/004_users.js

db.createCollection("users");

db.users.createIndex({ user_id: 1 }, { unique: true });
db.users.createIndex({ username: 1 }, { unique: true });
db.users.createIndex({ email: 1 }, { unique: true });
db.users.createIndex({ keycloak_id: 1 }, { unique: true, sparse: true });
```

---

### 4. WorkspaceRepository (workspace_repository.go)

**Collections: workspaces, workspace_members**

```javascript
// migrations/mongodb/005_workspaces.js

db.createCollection("workspaces");
db.workspaces.createIndex({ workspace_id: 1 }, { unique: true });
db.workspaces.createIndex({ keycloak_group_id: 1 }, { unique: true, sparse: true });

db.createCollection("workspace_members");
db.workspace_members.createIndex(
    { workspace_id: 1, user_id: 1 },
    { unique: true }
);
db.workspace_members.createIndex({ user_id: 1 });
```

---

### 5. NotificationRepository (notification_repository.go)

**Collection: notifications**

```javascript
// migrations/mongodb/006_notifications.js

db.createCollection("notifications");

db.notifications.createIndex({ notification_id: 1 }, { unique: true });
db.notifications.createIndex(
    { user_id: 1, read_at: 1, created_at: -1 },
    { name: "user_unread_created" }
);
db.notifications.createIndex({ created_at: -1 });
```

```go
func (r *MongoNotificationRepository) FindByUser(
    ctx context.Context,
    userID uuid.UUID,
    unreadOnly bool,
    limit, offset int,
) ([]notificationdomain.Notification, error) {
    filter := bson.M{"user_id": userID.String()}

    if unreadOnly {
        filter["read_at"] = nil
    }

    // ...
}
```

---

## Тестирование

### Unit Tests (с mock MongoDB)

```go
func TestChatRepository_LoadAndSave(t *testing.T) {
    // Setup
    client := testutil.SetupMongoDBv2(t)
    eventStore := eventstore.NewMongoEventStore(client, "test")
    repo := mongodb.NewMongoChatRepository(client, "test", eventStore)

    // Create chat
    chat, _ := chatdomain.NewChat(workspaceID, chatdomain.ChatTypeDiscussion, "Test", true, userID)

    // Save
    err := repo.Save(ctx, chat)
    require.NoError(t, err)

    // Load
    loaded, err := repo.Load(ctx, chat.ID())
    require.NoError(t, err)
    assert.Equal(t, chat.ID(), loaded.ID())
}
```

### Integration Tests

```go
func TestChatRepository_FindByWorkspace_Integration(t *testing.T) {
    // Test with real MongoDB
}
```

---

## Performance Targets

- **Save operation:** < 10ms (95th percentile)
- **Load aggregate:** < 20ms (95th percentile)
- **Query operations:** < 50ms (95th percentile)
- **Concurrent writes:** Support 100+ req/sec

---

## Критерии успеха

- ✅ **All 5 repositories implemented**
- ✅ **Event sourcing works for Chat**
- ✅ **Read model queries fast (<50ms)**
- ✅ **All indexes created**
- ✅ **Test coverage >80%**
- ✅ **Integration tests pass**
- ✅ **Performance targets met**

---

## Следующий шаг

→ **Task 1.1.3: Redis Repositories**
