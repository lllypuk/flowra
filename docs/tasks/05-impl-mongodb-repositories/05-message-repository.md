# Task 05: Message Repository — проверка и доработка

## Цель

Проверить и доработать MongoDB репозиторий для Message, добавив недостающую функцию `Restore` в domain layer и исправив метод `documentToMessage`.

## Контекст

Репозиторий `MongoMessageRepository` уже создан в `internal/infrastructure/repository/mongodb/message_repository.go`. Необходимо:

1. Проверить текущую реализацию на корректность
2. Добавить функцию `Restore` в domain/message если отсутствует
3. Реализовать или исправить `documentToMessage`
4. Добавить недостающие методы для работы с тредами и реакциями

## Зависимости

### Уже реализовано

- `internal/infrastructure/repository/mongodb/message_repository.go` — текущая реализация
- `internal/domain/message/message.go` — domain model Message
- `internal/domain/message/attachment.go` — Attachment value object
- `internal/domain/message/reaction.go` — Reaction value object
- `internal/application/message/repository.go` — интерфейсы репозитория

### Требуется проверить/изменить

1. `internal/domain/message/message.go` — добавить `Restore` если отсутствует
2. `internal/infrastructure/repository/mongodb/message_repository.go` — проверить и доработать

## Детальное описание

### 1. Анализ текущей структуры Message

Структура Message domain model:

```go
type Message struct {
    id              uuid.UUID
    chatID          uuid.UUID
    authorID        uuid.UUID
    content         string
    attachments     []Attachment
    reactions       []Reaction
    parentMessageID *uuid.UUID  // Для тредов
    isEdited        bool
    isDeleted       bool
    createdAt       time.Time
    updatedAt       time.Time
}
```

### 2. Добавить Restore функцию в domain

Изменить `internal/domain/message/message.go`:

```go
// Restore восстанавливает Message из сохраненных полей (для persistence layer)
// Эта функция должна использоваться ТОЛЬКО репозиторием для восстановления
// сущности из хранилища. Для создания нового сообщения используйте NewMessage.
func Restore(
    id uuid.UUID,
    chatID uuid.UUID,
    authorID uuid.UUID,
    content string,
    attachments []Attachment,
    reactions []Reaction,
    parentMessageID *uuid.UUID,
    isEdited bool,
    isDeleted bool,
    createdAt time.Time,
    updatedAt time.Time,
) *Message {
    return &Message{
        id:              id,
        chatID:          chatID,
        authorID:        authorID,
        content:         content,
        attachments:     attachments,
        reactions:       reactions,
        parentMessageID: parentMessageID,
        isEdited:        isEdited,
        isDeleted:       isDeleted,
        createdAt:       createdAt,
        updatedAt:       updatedAt,
    }
}

// RestoreAttachment восстанавливает Attachment из сохраненных полей
func RestoreAttachment(
    id uuid.UUID,
    filename string,
    contentType string,
    size int64,
    url string,
) Attachment {
    return Attachment{
        id:          id,
        filename:    filename,
        contentType: contentType,
        size:        size,
        url:         url,
    }
}

// RestoreReaction восстанавливает Reaction из сохраненных полей
func RestoreReaction(
    emoji string,
    userID uuid.UUID,
    createdAt time.Time,
) Reaction {
    return Reaction{
        emoji:     emoji,
        userID:    userID,
        createdAt: createdAt,
    }
}
```

### 3. Структуры документов

```go
// messageDocument представляет структуру документа в MongoDB
type messageDocument struct {
    MessageID       string               `bson:"message_id"`
    ChatID          string               `bson:"chat_id"`
    AuthorID        string               `bson:"author_id"`
    Content         string               `bson:"content"`
    Attachments     []attachmentDocument `bson:"attachments,omitempty"`
    Reactions       []reactionDocument   `bson:"reactions,omitempty"`
    ParentMessageID *string              `bson:"parent_message_id,omitempty"`
    IsEdited        bool                 `bson:"is_edited"`
    IsDeleted       bool                 `bson:"is_deleted"`
    CreatedAt       time.Time            `bson:"created_at"`
    UpdatedAt       time.Time            `bson:"updated_at"`
}

// attachmentDocument представляет вложение в документе
type attachmentDocument struct {
    AttachmentID string `bson:"attachment_id"`
    Filename     string `bson:"filename"`
    ContentType  string `bson:"content_type"`
    Size         int64  `bson:"size"`
    URL          string `bson:"url"`
}

// reactionDocument представляет реакцию в документе
type reactionDocument struct {
    Emoji     string    `bson:"emoji"`
    UserID    string    `bson:"user_id"`
    CreatedAt time.Time `bson:"created_at"`
}
```

### 4. Реализовать documentToMessage

```go
// documentToMessage преобразует Document в Message
func (r *MongoMessageRepository) documentToMessage(doc *messageDocument) (*messagedomain.Message, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Парсим UUID
    messageID := uuid.UUID(doc.MessageID)
    if messageID.IsZero() {
        return nil, fmt.Errorf("invalid message_id: %s", doc.MessageID)
    }

    chatID := uuid.UUID(doc.ChatID)
    if chatID.IsZero() {
        return nil, fmt.Errorf("invalid chat_id: %s", doc.ChatID)
    }

    authorID := uuid.UUID(doc.AuthorID)
    if authorID.IsZero() {
        return nil, fmt.Errorf("invalid author_id: %s", doc.AuthorID)
    }

    // Восстанавливаем attachments
    attachments := make([]messagedomain.Attachment, 0, len(doc.Attachments))
    for _, a := range doc.Attachments {
        attID := uuid.UUID(a.AttachmentID)
        if attID.IsZero() {
            continue
        }

        attachment := messagedomain.RestoreAttachment(
            attID,
            a.Filename,
            a.ContentType,
            a.Size,
            a.URL,
        )
        attachments = append(attachments, attachment)
    }

    // Восстанавливаем reactions
    reactions := make([]messagedomain.Reaction, 0, len(doc.Reactions))
    for _, r := range doc.Reactions {
        userID := uuid.UUID(r.UserID)
        if userID.IsZero() {
            continue
        }

        reaction := messagedomain.RestoreReaction(
            r.Emoji,
            userID,
            r.CreatedAt,
        )
        reactions = append(reactions, reaction)
    }

    // Парсим parent message ID
    var parentMessageID *uuid.UUID
    if doc.ParentMessageID != nil {
        pid := uuid.UUID(*doc.ParentMessageID)
        if !pid.IsZero() {
            parentMessageID = &pid
        }
    }

    // Восстанавливаем Message
    message := messagedomain.Restore(
        messageID,
        chatID,
        authorID,
        doc.Content,
        attachments,
        reactions,
        parentMessageID,
        doc.IsEdited,
        doc.IsDeleted,
        doc.CreatedAt,
        doc.UpdatedAt,
    )

    return message, nil
}
```

### 5. Реализовать messageToDocument

```go
// messageToDocument преобразует Message в Document
func (r *MongoMessageRepository) messageToDocument(msg *messagedomain.Message) messageDocument {
    // Преобразуем attachments
    attachments := make([]attachmentDocument, 0, len(msg.Attachments()))
    for _, a := range msg.Attachments() {
        attachments = append(attachments, attachmentDocument{
            AttachmentID: a.ID().String(),
            Filename:     a.Filename(),
            ContentType:  a.ContentType(),
            Size:         a.Size(),
            URL:          a.URL(),
        })
    }

    // Преобразуем reactions
    reactions := make([]reactionDocument, 0, len(msg.Reactions()))
    for _, r := range msg.Reactions() {
        reactions = append(reactions, reactionDocument{
            Emoji:     r.Emoji(),
            UserID:    r.UserID().String(),
            CreatedAt: r.CreatedAt(),
        })
    }

    doc := messageDocument{
        MessageID:   msg.ID().String(),
        ChatID:      msg.ChatID().String(),
        AuthorID:    msg.AuthorID().String(),
        Content:     msg.Content(),
        Attachments: attachments,
        Reactions:   reactions,
        IsEdited:    msg.IsEdited(),
        IsDeleted:   msg.IsDeleted(),
        CreatedAt:   msg.CreatedAt(),
        UpdatedAt:   msg.UpdatedAt(),
    }

    if msg.ParentMessageID() != nil {
        parentID := msg.ParentMessageID().String()
        doc.ParentMessageID = &parentID
    }

    return doc
}
```

### 6. Добавить методы для тредов

```go
// FindThread находит все ответы в треде
func (r *MongoMessageRepository) FindThread(
    ctx context.Context,
    parentMessageID uuid.UUID,
) ([]*messagedomain.Message, error) {
    if parentMessageID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "parent_message_id": parentMessageID.String(),
        "is_deleted":        false,
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: 1}})

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "messages")
    }
    defer cursor.Close(ctx)

    var messages []*messagedomain.Message
    for cursor.Next(ctx) {
        var doc messageDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        msg, docErr := r.documentToMessage(&doc)
        if docErr != nil {
            continue
        }

        messages = append(messages, msg)
    }

    if messages == nil {
        messages = make([]*messagedomain.Message, 0)
    }

    return messages, nil
}

// CountThreadReplies возвращает количество ответов в треде
func (r *MongoMessageRepository) CountThreadReplies(
    ctx context.Context,
    parentMessageID uuid.UUID,
) (int, error) {
    if parentMessageID.IsZero() {
        return 0, errs.ErrInvalidInput
    }

    filter := bson.M{
        "parent_message_id": parentMessageID.String(),
        "is_deleted":        false,
    }

    count, err := r.collection.CountDocuments(ctx, filter)
    if err != nil {
        return 0, HandleMongoError(err, "messages")
    }

    return int(count), nil
}
```

### 7. Добавить методы для реакций

```go
// AddReaction добавляет реакцию к сообщению
func (r *MongoMessageRepository) AddReaction(
    ctx context.Context,
    messageID uuid.UUID,
    emoji string,
    userID uuid.UUID,
) error {
    if messageID.IsZero() || emoji == "" || userID.IsZero() {
        return errs.ErrInvalidInput
    }

    reaction := reactionDocument{
        Emoji:     emoji,
        UserID:    userID.String(),
        CreatedAt: time.Now().UTC(),
    }

    filter := bson.M{"message_id": messageID.String()}
    update := bson.M{
        "$push": bson.M{"reactions": reaction},
        "$set":  bson.M{"updated_at": time.Now().UTC()},
    }

    result, err := r.collection.UpdateOne(ctx, filter, update)
    if err != nil {
        return HandleMongoError(err, "message")
    }

    if result.MatchedCount == 0 {
        return errs.ErrNotFound
    }

    return nil
}

// RemoveReaction удаляет реакцию с сообщения
func (r *MongoMessageRepository) RemoveReaction(
    ctx context.Context,
    messageID uuid.UUID,
    emoji string,
    userID uuid.UUID,
) error {
    if messageID.IsZero() || emoji == "" || userID.IsZero() {
        return errs.ErrInvalidInput
    }

    filter := bson.M{"message_id": messageID.String()}
    update := bson.M{
        "$pull": bson.M{
            "reactions": bson.M{
                "emoji":   emoji,
                "user_id": userID.String(),
            },
        },
        "$set": bson.M{"updated_at": time.Now().UTC()},
    }

    result, err := r.collection.UpdateOne(ctx, filter, update)
    if err != nil {
        return HandleMongoError(err, "message")
    }

    if result.MatchedCount == 0 {
        return errs.ErrNotFound
    }

    return nil
}

// GetReactionUsers возвращает пользователей, поставивших определенную реакцию
func (r *MongoMessageRepository) GetReactionUsers(
    ctx context.Context,
    messageID uuid.UUID,
    emoji string,
) ([]uuid.UUID, error) {
    if messageID.IsZero() || emoji == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"message_id": messageID.String()}
    var doc messageDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "message")
    }

    var userIDs []uuid.UUID
    for _, r := range doc.Reactions {
        if r.Emoji == emoji {
            userID := uuid.UUID(r.UserID)
            if !userID.IsZero() {
                userIDs = append(userIDs, userID)
            }
        }
    }

    if userIDs == nil {
        userIDs = make([]uuid.UUID, 0)
    }

    return userIDs, nil
}
```

### 8. Добавить методы поиска

```go
// SearchInChat ищет сообщения в чате по тексту
func (r *MongoMessageRepository) SearchInChat(
    ctx context.Context,
    chatID uuid.UUID,
    query string,
    offset, limit int,
) ([]*messagedomain.Message, error) {
    if chatID.IsZero() || query == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "chat_id":    chatID.String(),
        "is_deleted": false,
        "content": bson.M{
            "$regex":   query,
            "$options": "i", // case-insensitive
        },
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "messages")
    }
    defer cursor.Close(ctx)

    var messages []*messagedomain.Message
    for cursor.Next(ctx) {
        var doc messageDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        msg, docErr := r.documentToMessage(&doc)
        if docErr != nil {
            continue
        }

        messages = append(messages, msg)
    }

    if messages == nil {
        messages = make([]*messagedomain.Message, 0)
    }

    return messages, nil
}

// FindByAuthor находит сообщения автора в чате
func (r *MongoMessageRepository) FindByAuthor(
    ctx context.Context,
    chatID uuid.UUID,
    authorID uuid.UUID,
    offset, limit int,
) ([]*messagedomain.Message, error) {
    if chatID.IsZero() || authorID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "chat_id":    chatID.String(),
        "author_id":  authorID.String(),
        "is_deleted": false,
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "messages")
    }
    defer cursor.Close(ctx)

    var messages []*messagedomain.Message
    for cursor.Next(ctx) {
        var doc messageDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        msg, docErr := r.documentToMessage(&doc)
        if docErr != nil {
            continue
        }

        messages = append(messages, msg)
    }

    if messages == nil {
        messages = make([]*messagedomain.Message, 0)
    }

    return messages, nil
}
```

## Тестирование

### Тесты для documentToMessage

```go
func TestMongoMessageRepository_Save_And_FindByID(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("messages")
    repo := mongodb.NewMongoMessageRepository(coll)

    // Create message
    msgID := uuid.NewUUID()
    chatID := uuid.NewUUID()
    authorID := uuid.NewUUID()

    msg := messagedomain.NewMessage(msgID, chatID, authorID, "Test message content")

    err := repo.Save(ctx, msg)
    require.NoError(t, err)

    // Load message
    loaded, err := repo.FindByID(ctx, msgID)
    require.NoError(t, err)

    // Verify fields
    assert.Equal(t, msgID, loaded.ID())
    assert.Equal(t, chatID, loaded.ChatID())
    assert.Equal(t, authorID, loaded.AuthorID())
    assert.Equal(t, "Test message content", loaded.Content())
    assert.False(t, loaded.IsEdited())
    assert.False(t, loaded.IsDeleted())
}

func TestMongoMessageRepository_FindThread(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("messages")
    repo := mongodb.NewMongoMessageRepository(coll)

    // Create parent message
    parentID := uuid.NewUUID()
    chatID := uuid.NewUUID()
    authorID := uuid.NewUUID()

    parent := messagedomain.NewMessage(parentID, chatID, authorID, "Parent message")
    _ = repo.Save(ctx, parent)

    // Create thread replies
    for i := 0; i < 3; i++ {
        replyID := uuid.NewUUID()
        reply := messagedomain.NewThreadReply(replyID, chatID, authorID, fmt.Sprintf("Reply %d", i), parentID)
        _ = repo.Save(ctx, reply)
    }

    // Find thread
    replies, err := repo.FindThread(ctx, parentID)
    require.NoError(t, err)

    assert.Len(t, replies, 3)
}

func TestMongoMessageRepository_AddReaction(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("messages")
    repo := mongodb.NewMongoMessageRepository(coll)

    // Create message
    msgID := uuid.NewUUID()
    chatID := uuid.NewUUID()
    authorID := uuid.NewUUID()
    msg := messagedomain.NewMessage(msgID, chatID, authorID, "Test message")
    _ = repo.Save(ctx, msg)

    // Add reaction
    reactorID := uuid.NewUUID()
    err := repo.AddReaction(ctx, msgID, "👍", reactorID)
    require.NoError(t, err)

    // Verify reaction
    loaded, err := repo.FindByID(ctx, msgID)
    require.NoError(t, err)

    assert.Len(t, loaded.Reactions(), 1)
    assert.Equal(t, "👍", loaded.Reactions()[0].Emoji())
}

func TestMongoMessageRepository_SearchInChat(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("messages")
    repo := mongodb.NewMongoMessageRepository(coll)

    // Create messages
    chatID := uuid.NewUUID()
    authorID := uuid.NewUUID()

    messages := []string{
        "Hello world",
        "World is beautiful",
        "Something else",
    }

    for _, content := range messages {
        msgID := uuid.NewUUID()
        msg := messagedomain.NewMessage(msgID, chatID, authorID, content)
        _ = repo.Save(ctx, msg)
    }

    // Search for "world"
    results, err := repo.SearchInChat(ctx, chatID, "world", 0, 10)
    require.NoError(t, err)

    assert.Len(t, results, 2)
}
```

## Индексы для Messages

Добавить в `07-mongodb-indexes.md`:

```javascript
// Messages Collection
db.messages.createIndex({ "message_id": 1 }, { unique: true })
db.messages.createIndex({ "chat_id": 1, "created_at": -1 })
db.messages.createIndex({ "chat_id": 1, "author_id": 1 })
db.messages.createIndex({ "parent_message_id": 1 })
db.messages.createIndex({ "author_id": 1 })

// Text index for search
db.messages.createIndex({ "content": "text" })

// Compound indexes
db.messages.createIndex({ "chat_id": 1, "is_deleted": 1, "created_at": -1 })
```

## Checklist

### Phase 1: Domain layer

- [ ] Проверить существование функции `Restore` в `message.go`
- [ ] Добавить `Restore` если отсутствует
- [ ] Добавить `RestoreAttachment` если отсутствует
- [ ] Добавить `RestoreReaction` если отсутствует
- [ ] Проверить наличие всех необходимых getters

### Phase 2: Document structures

- [ ] Проверить/создать `messageDocument` структуру
- [ ] Проверить/создать `attachmentDocument` структуру
- [ ] Проверить/создать `reactionDocument` структуру

### Phase 3: Core methods

- [ ] Реализовать/исправить `documentToMessage`
- [ ] Реализовать/исправить `messageToDocument`
- [ ] Проверить метод `Save`
- [ ] Проверить метод `FindByID`
- [ ] Проверить метод `FindByChatID`

### Phase 4: Thread methods

- [ ] Добавить/проверить метод `FindThread`
- [ ] Добавить метод `CountThreadReplies`

### Phase 5: Reaction methods

- [ ] Добавить метод `AddReaction`
- [ ] Добавить метод `RemoveReaction`
- [ ] Добавить метод `GetReactionUsers`

### Phase 6: Search methods

- [ ] Добавить метод `SearchInChat`
- [ ] Добавить метод `FindByAuthor`

### Phase 7: Interface update

- [ ] Обновить `QueryRepository` интерфейс с новыми методами
- [ ] Убедиться, что `MongoMessageRepository` реализует все методы

### Phase 8: Тестирование

- [ ] Добавить тест `Save_And_FindByID`
- [ ] Добавить тест `FindThread`
- [ ] Добавить тест `AddReaction`
- [ ] Добавить тест `SearchInChat`
- [ ] Проверить, что все существующие тесты проходят

## Следующие шаги

После завершения этой задачи:

1. **Task 06** — проверка и доработка NotificationRepository
2. **Task 07** — создание индексов MongoDB

## Референсы

- Существующий код: `internal/infrastructure/repository/mongodb/message_repository.go`
- Domain model: `internal/domain/message/message.go`
- Интерфейсы: `internal/application/message/repository.go`
- Аналогичная реализация: `user_repository.go`, `workspace_repository.go`
