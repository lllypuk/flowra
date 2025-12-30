# Task 06: Notification Repository — проверка и доработка

## Цель

Проверить и доработать MongoDB репозиторий для Notification, добавив недостающую функцию `Restore` в domain layer и исправив метод `documentToNotification`.

## Контекст

Репозиторий `MongoNotificationRepository` уже создан в `internal/infrastructure/repository/mongodb/notification_repository.go`. Необходимо:

1. Проверить текущую реализацию на корректность
2. Добавить функцию `Restore` в domain/notification если отсутствует
3. Реализовать или исправить `documentToNotification`
4. Добавить недостающие методы для batch операций

## Зависимости

### Уже реализовано

- `internal/infrastructure/repository/mongodb/notification_repository.go` — текущая реализация
- `internal/domain/notification/notification.go` — domain model Notification
- `internal/application/notification/repository.go` — интерфейсы репозитория

### Требуется проверить/изменить

1. `internal/domain/notification/notification.go` — добавить `Restore` если отсутствует
2. `internal/infrastructure/repository/mongodb/notification_repository.go` — проверить и доработать

## Детальное описание

### 1. Анализ текущей структуры Notification

Структура Notification domain model:

```go
type Notification struct {
    id          uuid.UUID
    userID      uuid.UUID
    type_       Type      // mention, task_assigned, status_changed, etc.
    title       string
    body        string
    resourceID  *uuid.UUID // ID связанного ресурса (chat, task, etc.)
    resourceType *string   // Тип ресурса
    isRead      bool
    createdAt   time.Time
    readAt      *time.Time
}
```

### 2. Добавить Restore функцию в domain

Изменить `internal/domain/notification/notification.go`:

```go
// Restore восстанавливает Notification из сохраненных полей (для persistence layer)
// Эта функция должна использоваться ТОЛЬКО репозиторием для восстановления
// сущности из хранилища. Для создания нового уведомления используйте NewNotification.
func Restore(
    id uuid.UUID,
    userID uuid.UUID,
    notificationType Type,
    title string,
    body string,
    resourceID *uuid.UUID,
    resourceType *string,
    isRead bool,
    createdAt time.Time,
    readAt *time.Time,
) *Notification {
    return &Notification{
        id:           id,
        userID:       userID,
        type_:        notificationType,
        title:        title,
        body:         body,
        resourceID:   resourceID,
        resourceType: resourceType,
        isRead:       isRead,
        createdAt:    createdAt,
        readAt:       readAt,
    }
}
```

### 3. Структура документа

```go
// notificationDocument представляет структуру документа в MongoDB
type notificationDocument struct {
    NotificationID string     `bson:"notification_id"`
    UserID         string     `bson:"user_id"`
    Type           string     `bson:"type"`
    Title          string     `bson:"title"`
    Body           string     `bson:"body"`
    ResourceID     *string    `bson:"resource_id,omitempty"`
    ResourceType   *string    `bson:"resource_type,omitempty"`
    IsRead         bool       `bson:"is_read"`
    CreatedAt      time.Time  `bson:"created_at"`
    ReadAt         *time.Time `bson:"read_at,omitempty"`
}
```

### 4. Реализовать documentToNotification

```go
// documentToNotification преобразует Document в Notification
func (r *MongoNotificationRepository) documentToNotification(doc *notificationDocument) (*notificationdomain.Notification, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Парсим UUID
    notificationID := uuid.UUID(doc.NotificationID)
    if notificationID.IsZero() {
        return nil, fmt.Errorf("invalid notification_id: %s", doc.NotificationID)
    }

    userID := uuid.UUID(doc.UserID)
    if userID.IsZero() {
        return nil, fmt.Errorf("invalid user_id: %s", doc.UserID)
    }

    // Парсим resource ID
    var resourceID *uuid.UUID
    if doc.ResourceID != nil {
        rid := uuid.UUID(*doc.ResourceID)
        if !rid.IsZero() {
            resourceID = &rid
        }
    }

    // Восстанавливаем Notification
    notification := notificationdomain.Restore(
        notificationID,
        userID,
        notificationdomain.Type(doc.Type),
        doc.Title,
        doc.Body,
        resourceID,
        doc.ResourceType,
        doc.IsRead,
        doc.CreatedAt,
        doc.ReadAt,
    )

    return notification, nil
}
```

### 5. Реализовать notificationToDocument

```go
// notificationToDocument преобразует Notification в Document
func (r *MongoNotificationRepository) notificationToDocument(n *notificationdomain.Notification) notificationDocument {
    doc := notificationDocument{
        NotificationID: n.ID().String(),
        UserID:         n.UserID().String(),
        Type:           string(n.Type()),
        Title:          n.Title(),
        Body:           n.Body(),
        ResourceType:   n.ResourceType(),
        IsRead:         n.IsRead(),
        CreatedAt:      n.CreatedAt(),
        ReadAt:         n.ReadAt(),
    }

    if n.ResourceID() != nil {
        resourceID := n.ResourceID().String()
        doc.ResourceID = &resourceID
    }

    return doc
}
```

### 6. Добавить batch методы

```go
// SaveBatch сохраняет несколько уведомлений за один запрос
func (r *MongoNotificationRepository) SaveBatch(
    ctx context.Context,
    notifications []*notificationdomain.Notification,
) error {
    if len(notifications) == 0 {
        return nil
    }

    docs := make([]any, len(notifications))
    for i, n := range notifications {
        if n == nil {
            return errs.ErrInvalidInput
        }
        docs[i] = r.notificationToDocument(n)
    }

    _, err := r.collection.InsertMany(ctx, docs)
    if err != nil {
        return HandleMongoError(err, "notifications")
    }

    return nil
}

// DeleteByUser удаляет все уведомления пользователя
func (r *MongoNotificationRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
    if userID.IsZero() {
        return errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": userID.String()}
    _, err := r.collection.DeleteMany(ctx, filter)
    if err != nil {
        return HandleMongoError(err, "notifications")
    }

    return nil
}

// DeleteOlderThan удаляет уведомления старше указанной даты
func (r *MongoNotificationRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int, error) {
    filter := bson.M{
        "created_at": bson.M{"$lt": before},
    }

    result, err := r.collection.DeleteMany(ctx, filter)
    if err != nil {
        return 0, HandleMongoError(err, "notifications")
    }

    return int(result.DeletedCount), nil
}

// DeleteReadOlderThan удаляет прочитанные уведомления старше указанной даты
func (r *MongoNotificationRepository) DeleteReadOlderThan(ctx context.Context, before time.Time) (int, error) {
    filter := bson.M{
        "is_read":    true,
        "created_at": bson.M{"$lt": before},
    }

    result, err := r.collection.DeleteMany(ctx, filter)
    if err != nil {
        return 0, HandleMongoError(err, "notifications")
    }

    return int(result.DeletedCount), nil
}
```

### 7. Добавить методы группировки

```go
// CountByType возвращает количество уведомлений по типам для пользователя
func (r *MongoNotificationRepository) CountByType(
    ctx context.Context,
    userID uuid.UUID,
) (map[notificationdomain.Type]int, error) {
    if userID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    pipeline := bson.A{
        bson.M{"$match": bson.M{"user_id": userID.String()}},
        bson.M{"$group": bson.M{
            "_id":   "$type",
            "count": bson.M{"$sum": 1},
        }},
    }

    cursor, err := r.collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, HandleMongoError(err, "notifications")
    }
    defer cursor.Close(ctx)

    result := make(map[notificationdomain.Type]int)
    for cursor.Next(ctx) {
        var item struct {
            Type  string `bson:"_id"`
            Count int    `bson:"count"`
        }
        if decodeErr := cursor.Decode(&item); decodeErr != nil {
            continue
        }
        result[notificationdomain.Type(item.Type)] = item.Count
    }

    return result, nil
}

// FindByType находит уведомления определенного типа
func (r *MongoNotificationRepository) FindByType(
    ctx context.Context,
    userID uuid.UUID,
    notificationType notificationdomain.Type,
    offset, limit int,
) ([]*notificationdomain.Notification, error) {
    if userID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "user_id": userID.String(),
        "type":    string(notificationType),
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "notifications")
    }
    defer cursor.Close(ctx)

    var notifications []*notificationdomain.Notification
    for cursor.Next(ctx) {
        var doc notificationDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        n, docErr := r.documentToNotification(&doc)
        if docErr != nil {
            continue
        }

        notifications = append(notifications, n)
    }

    if notifications == nil {
        notifications = make([]*notificationdomain.Notification, 0)
    }

    return notifications, nil
}

// FindByResource находит уведомления связанные с ресурсом
func (r *MongoNotificationRepository) FindByResource(
    ctx context.Context,
    resourceID uuid.UUID,
    resourceType string,
) ([]*notificationdomain.Notification, error) {
    if resourceID.IsZero() || resourceType == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "resource_id":   resourceID.String(),
        "resource_type": resourceType,
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}})

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "notifications")
    }
    defer cursor.Close(ctx)

    var notifications []*notificationdomain.Notification
    for cursor.Next(ctx) {
        var doc notificationDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        n, docErr := r.documentToNotification(&doc)
        if docErr != nil {
            continue
        }

        notifications = append(notifications, n)
    }

    if notifications == nil {
        notifications = make([]*notificationdomain.Notification, 0)
    }

    return notifications, nil
}
```

### 8. Улучшить методы отметки прочитанным

```go
// MarkAsRead отмечает уведомление как прочитанное
func (r *MongoNotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
    if id.IsZero() {
        return errs.ErrInvalidInput
    }

    now := time.Now().UTC()
    filter := bson.M{"notification_id": id.String()}
    update := bson.M{
        "$set": bson.M{
            "is_read": true,
            "read_at": now,
        },
    }

    result, err := r.collection.UpdateOne(ctx, filter, update)
    if err != nil {
        return HandleMongoError(err, "notification")
    }

    if result.MatchedCount == 0 {
        return errs.ErrNotFound
    }

    return nil
}

// MarkAllAsRead отмечает все уведомления пользователя как прочитанные
func (r *MongoNotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
    if userID.IsZero() {
        return errs.ErrInvalidInput
    }

    now := time.Now().UTC()
    filter := bson.M{
        "user_id": userID.String(),
        "is_read": false,
    }
    update := bson.M{
        "$set": bson.M{
            "is_read": true,
            "read_at": now,
        },
    }

    _, err := r.collection.UpdateMany(ctx, filter, update)
    if err != nil {
        return HandleMongoError(err, "notifications")
    }

    return nil
}

// MarkManyAsRead отмечает несколько уведомлений как прочитанные
func (r *MongoNotificationRepository) MarkManyAsRead(ctx context.Context, ids []uuid.UUID) error {
    if len(ids) == 0 {
        return nil
    }

    idStrings := make([]string, len(ids))
    for i, id := range ids {
        if id.IsZero() {
            return errs.ErrInvalidInput
        }
        idStrings[i] = id.String()
    }

    now := time.Now().UTC()
    filter := bson.M{
        "notification_id": bson.M{"$in": idStrings},
    }
    update := bson.M{
        "$set": bson.M{
            "is_read": true,
            "read_at": now,
        },
    }

    _, err := r.collection.UpdateMany(ctx, filter, update)
    if err != nil {
        return HandleMongoError(err, "notifications")
    }

    return nil
}
```

## Тестирование

### Тесты для documentToNotification

```go
func TestMongoNotificationRepository_Save_And_FindByID(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    // Create notification
    notifID := uuid.NewUUID()
    userID := uuid.NewUUID()
    resourceID := uuid.NewUUID()
    resourceType := "task"

    notif := notificationdomain.NewNotification(
        notifID,
        userID,
        notificationdomain.TypeTaskAssigned,
        "Task Assigned",
        "You have been assigned to task #123",
        &resourceID,
        &resourceType,
    )

    err := repo.Save(ctx, notif)
    require.NoError(t, err)

    // Load notification
    loaded, err := repo.FindByID(ctx, notifID)
    require.NoError(t, err)

    // Verify fields
    assert.Equal(t, notifID, loaded.ID())
    assert.Equal(t, userID, loaded.UserID())
    assert.Equal(t, notificationdomain.TypeTaskAssigned, loaded.Type())
    assert.Equal(t, "Task Assigned", loaded.Title())
    assert.Equal(t, "You have been assigned to task #123", loaded.Body())
    assert.False(t, loaded.IsRead())
}

func TestMongoNotificationRepository_MarkAsRead(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    // Create notification
    notifID := uuid.NewUUID()
    userID := uuid.NewUUID()
    notif := notificationdomain.NewNotification(
        notifID,
        userID,
        notificationdomain.TypeMention,
        "Mention",
        "You were mentioned",
        nil,
        nil,
    )
    _ = repo.Save(ctx, notif)

    // Mark as read
    err := repo.MarkAsRead(ctx, notifID)
    require.NoError(t, err)

    // Verify
    loaded, err := repo.FindByID(ctx, notifID)
    require.NoError(t, err)
    assert.True(t, loaded.IsRead())
    assert.NotNil(t, loaded.ReadAt())
}

func TestMongoNotificationRepository_MarkAllAsRead(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    // Create notifications
    userID := uuid.NewUUID()
    for i := 0; i < 5; i++ {
        notifID := uuid.NewUUID()
        notif := notificationdomain.NewNotification(
            notifID,
            userID,
            notificationdomain.TypeMention,
            fmt.Sprintf("Notification %d", i),
            "Body",
            nil,
            nil,
        )
        _ = repo.Save(ctx, notif)
    }

    // Mark all as read
    err := repo.MarkAllAsRead(ctx, userID)
    require.NoError(t, err)

    // Verify
    unread, err := repo.FindUnreadByUserID(ctx, userID, 10)
    require.NoError(t, err)
    assert.Len(t, unread, 0)
}

func TestMongoNotificationRepository_CountUnreadByUserID(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    // Create notifications
    userID := uuid.NewUUID()
    for i := 0; i < 3; i++ {
        notifID := uuid.NewUUID()
        notif := notificationdomain.NewNotification(
            notifID,
            userID,
            notificationdomain.TypeMention,
            fmt.Sprintf("Notification %d", i),
            "Body",
            nil,
            nil,
        )
        _ = repo.Save(ctx, notif)
    }

    // Count unread
    count, err := repo.CountUnreadByUserID(ctx, userID)
    require.NoError(t, err)
    assert.Equal(t, 3, count)
}

func TestMongoNotificationRepository_DeleteOlderThan(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    // Create old notification (manually set created_at in the past)
    userID := uuid.NewUUID()
    
    // Insert directly with old date
    oldDoc := bson.M{
        "notification_id": uuid.NewUUID().String(),
        "user_id":         userID.String(),
        "type":            "mention",
        "title":           "Old notification",
        "body":            "Body",
        "is_read":         true,
        "created_at":      time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
    }
    _, _ = coll.InsertOne(ctx, oldDoc)

    // Create recent notification
    notifID := uuid.NewUUID()
    notif := notificationdomain.NewNotification(
        notifID,
        userID,
        notificationdomain.TypeMention,
        "Recent notification",
        "Body",
        nil,
        nil,
    )
    _ = repo.Save(ctx, notif)

    // Delete older than 7 days
    deleted, err := repo.DeleteOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
    require.NoError(t, err)
    assert.Equal(t, 1, deleted)

    // Verify recent notification still exists
    _, err = repo.FindByID(ctx, notifID)
    require.NoError(t, err)
}

func TestMongoNotificationRepository_CountByType(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("notifications")
    repo := mongodb.NewMongoNotificationRepository(coll)

    userID := uuid.NewUUID()

    // Create notifications of different types
    types := []notificationdomain.Type{
        notificationdomain.TypeMention,
        notificationdomain.TypeMention,
        notificationdomain.TypeTaskAssigned,
        notificationdomain.TypeStatusChanged,
    }

    for i, typ := range types {
        notifID := uuid.NewUUID()
        notif := notificationdomain.NewNotification(
            notifID,
            userID,
            typ,
            fmt.Sprintf("Notification %d", i),
            "Body",
            nil,
            nil,
        )
        _ = repo.Save(ctx, notif)
    }

    // Count by type
    counts, err := repo.CountByType(ctx, userID)
    require.NoError(t, err)

    assert.Equal(t, 2, counts[notificationdomain.TypeMention])
    assert.Equal(t, 1, counts[notificationdomain.TypeTaskAssigned])
    assert.Equal(t, 1, counts[notificationdomain.TypeStatusChanged])
}
```

## Индексы для Notifications

Добавить в `07-mongodb-indexes.md`:

```javascript
// Notifications Collection
db.notifications.createIndex({ "notification_id": 1 }, { unique: true })
db.notifications.createIndex({ "user_id": 1, "created_at": -1 })
db.notifications.createIndex({ "user_id": 1, "is_read": 1 })
db.notifications.createIndex({ "user_id": 1, "type": 1 })
db.notifications.createIndex({ "resource_id": 1, "resource_type": 1 })
db.notifications.createIndex({ "created_at": 1 })  // For TTL cleanup

// Compound indexes
db.notifications.createIndex({ "user_id": 1, "is_read": 1, "created_at": -1 })

// TTL index for automatic cleanup (optional)
// db.notifications.createIndex({ "created_at": 1 }, { expireAfterSeconds: 7776000 }) // 90 days
```

## Checklist

### Phase 1: Domain layer

- [ ] Проверить существование функции `Restore` в `notification.go`
- [ ] Добавить `Restore` если отсутствует
- [ ] Проверить наличие всех необходимых getters
- [ ] Проверить/добавить notification types (Type enum)

### Phase 2: Document structure

- [ ] Проверить/создать `notificationDocument` структуру
- [ ] Убедиться в корректности BSON тегов

### Phase 3: Core methods

- [ ] Реализовать/исправить `documentToNotification`
- [ ] Реализовать/исправить `notificationToDocument`
- [ ] Проверить метод `Save`
- [ ] Проверить метод `FindByID`
- [ ] Проверить метод `FindByUserID`
- [ ] Проверить метод `FindUnreadByUserID`

### Phase 4: Batch methods

- [ ] Добавить метод `SaveBatch`
- [ ] Добавить метод `DeleteByUser`
- [ ] Добавить метод `DeleteOlderThan`
- [ ] Добавить метод `DeleteReadOlderThan`

### Phase 5: Grouping methods

- [ ] Добавить метод `CountByType`
- [ ] Добавить метод `FindByType`
- [ ] Добавить метод `FindByResource`

### Phase 6: Read status methods

- [ ] Проверить/улучшить метод `MarkAsRead`
- [ ] Проверить/улучшить метод `MarkAllAsRead`
- [ ] Добавить метод `MarkManyAsRead`

### Phase 7: Interface update

- [ ] Обновить `CommandRepository` интерфейс с batch методами
- [ ] Обновить `QueryRepository` интерфейс с grouping методами
- [ ] Убедиться, что `MongoNotificationRepository` реализует все методы

### Phase 8: Тестирование

- [ ] Добавить тест `Save_And_FindByID`
- [ ] Добавить тест `MarkAsRead`
- [ ] Добавить тест `MarkAllAsRead`
- [ ] Добавить тест `CountUnreadByUserID`
- [ ] Добавить тест `DeleteOlderThan`
- [ ] Добавить тест `CountByType`
- [ ] Проверить, что все существующие тесты проходят

## Следующие шаги

После завершения этой задачи:

1. **Task 07** — создание индексов MongoDB
2. **Task 08** — интеграционные тесты с testcontainers

## Референсы

- Существующий код: `internal/infrastructure/repository/mongodb/notification_repository.go`
- Domain model: `internal/domain/notification/notification.go`
- Интерфейсы: `internal/application/notification/repository.go`
- Аналогичная реализация: `user_repository.go`, `message_repository.go`
