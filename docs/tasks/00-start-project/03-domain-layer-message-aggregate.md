# Task 03: Domain Layer — Message Aggregate (Phase 1.7)

**Фаза:** 1 - Domain Layer (Extension)
**Приоритет:** High
**Статус:** ✅ **COMPLETED**
**Дата создания:** 2025-10-17
**Предыдущая задача:** [02-domain-layer.md](./02-domain-layer.md) ✅

## Цель

Реализовать полноценный Message aggregate — основную сущность для сообщений в чатах. Message aggregate будет включать поддержку редактирования, удаления, реакций (эмоджи), вложений (attachments) и тредов (replies).

**Принцип:** Domain-first approach — чистая бизнес-логика без зависимостей от инфраструктуры.

---

## Контекст

В Phase 1.4 (Chat Aggregate) была упрощенная концепция Message, но полноценная сущность не была реализована. Message — это ключевой aggregate для системы чатов, требующий:

- Редактирование сообщений с историей
- Мягкое удаление (soft delete)
- Эмоджи реакции от пользователей
- Файловые вложения
- Поддержка тредов (ответы на сообщения)
- Валидация прав доступа (только автор может редактировать)

---

## Подзадачи

### 1.7.1 Message Value Objects

**Описание:** Создать value objects для вложений и реакций.

#### MessageAttachment Value Object

**Файл:** `internal/domain/message/attachment.go`

**Реализация:**
```go
package message

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment представляет файловое вложение к сообщению
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment создает новое вложение
func NewAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) (Attachment, error) {
	if fileID.IsZero() {
		return Attachment{}, errs.ErrInvalidInput
	}
	if fileName == "" {
		return Attachment{}, errs.ErrInvalidInput
	}
	if fileSize <= 0 {
		return Attachment{}, errs.ErrInvalidInput
	}
	if mimeType == "" {
		return Attachment{}, errs.ErrInvalidInput
	}

	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}, nil
}

// Getters
func (a Attachment) FileID() uuid.UUID   { return a.fileID }
func (a Attachment) FileName() string    { return a.fileName }
func (a Attachment) FileSize() int64     { return a.fileSize }
func (a Attachment) MimeType() string    { return a.mimeType }
```

#### MessageReaction Value Object

**Файл:** `internal/domain/message/reaction.go`

**Реализация:**
```go
package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Reaction представляет эмоджи реакцию на сообщение
type Reaction struct {
	userID    uuid.UUID
	emojiCode string
	addedAt   time.Time
}

// NewReaction создает новую реакцию
func NewReaction(userID uuid.UUID, emojiCode string) (Reaction, error) {
	if userID.IsZero() {
		return Reaction{}, errs.ErrInvalidInput
	}
	if emojiCode == "" {
		return Reaction{}, errs.ErrInvalidInput
	}

	return Reaction{
		userID:    userID,
		emojiCode: emojiCode,
		addedAt:   time.Now(),
	}, nil
}

// Getters
func (r Reaction) UserID() uuid.UUID    { return r.userID }
func (r Reaction) EmojiCode() string    { return r.emojiCode }
func (r Reaction) AddedAt() time.Time   { return r.addedAt }
```

**Критерии выполнения:**
- [ ] Attachment value object создан с валидацией
- [ ] Reaction value object создан с валидацией
- [ ] Unit tests для value objects

---

### 1.7.2 Message Aggregate Root

**Файл:** `internal/domain/message/message.go`

**Реализация:**
```go
package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Message представляет сообщение в чате
type Message struct {
	id              uuid.UUID
	chatID          uuid.UUID
	authorID        uuid.UUID
	content         string
	parentMessageID uuid.UUID // для тредов
	createdAt       time.Time
	editedAt        *time.Time
	isDeleted       bool
	deletedAt       *time.Time
	attachments     []Attachment
	reactions       []Reaction
}

// NewMessage создает новое сообщение
func NewMessage(
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
	parentMessageID uuid.UUID,
) (*Message, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if authorID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if content == "" {
		return nil, errs.ErrInvalidInput
	}

	return &Message{
		id:              uuid.NewUUID(),
		chatID:          chatID,
		authorID:        authorID,
		content:         content,
		parentMessageID: parentMessageID,
		createdAt:       time.Now(),
		isDeleted:       false,
		attachments:     make([]Attachment, 0),
		reactions:       make([]Reaction, 0),
	}, nil
}

// EditContent редактирует содержимое сообщения
func (m *Message) EditContent(newContent string, editorID uuid.UUID) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if newContent == "" {
		return errs.ErrInvalidInput
	}
	if !m.CanBeEditedBy(editorID) {
		return errs.ErrForbidden
	}

	m.content = newContent
	now := time.Now()
	m.editedAt = &now
	return nil
}

// Delete мягко удаляет сообщение
func (m *Message) Delete(deleterID uuid.UUID) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if !m.CanBeEditedBy(deleterID) {
		return errs.ErrForbidden
	}

	m.isDeleted = true
	now := time.Now()
	m.deletedAt = &now
	return nil
}

// AddReaction добавляет реакцию
func (m *Message) AddReaction(userID uuid.UUID, emojiCode string) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if m.HasReaction(userID, emojiCode) {
		return errs.ErrAlreadyExists
	}

	reaction, err := NewReaction(userID, emojiCode)
	if err != nil {
		return err
	}

	m.reactions = append(m.reactions, reaction)
	return nil
}

// RemoveReaction удаляет реакцию
func (m *Message) RemoveReaction(userID uuid.UUID, emojiCode string) error {
	if !m.HasReaction(userID, emojiCode) {
		return errs.ErrNotFound
	}

	newReactions := make([]Reaction, 0, len(m.reactions)-1)
	for _, r := range m.reactions {
		if !(r.UserID() == userID && r.EmojiCode() == emojiCode) {
			newReactions = append(newReactions, r)
		}
	}
	m.reactions = newReactions
	return nil
}

// AddAttachment добавляет вложение
func (m *Message) AddAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}

	attachment, err := NewAttachment(fileID, fileName, fileSize, mimeType)
	if err != nil {
		return err
	}

	m.attachments = append(m.attachments, attachment)
	return nil
}

// HasReaction проверяет наличие реакции от пользователя
func (m *Message) HasReaction(userID uuid.UUID, emojiCode string) bool {
	for _, r := range m.reactions {
		if r.UserID() == userID && r.EmojiCode() == emojiCode {
			return true
		}
	}
	return false
}

// CanBeEditedBy проверяет, может ли пользователь редактировать сообщение
func (m *Message) CanBeEditedBy(userID uuid.UUID) bool {
	return m.authorID == userID
}

// IsEdited проверяет, было ли сообщение отредактировано
func (m *Message) IsEdited() bool {
	return m.editedAt != nil
}

// IsReply проверяет, является ли сообщение ответом (в треде)
func (m *Message) IsReply() bool {
	return !m.parentMessageID.IsZero()
}

// Getters

func (m *Message) ID() uuid.UUID              { return m.id }
func (m *Message) ChatID() uuid.UUID          { return m.chatID }
func (m *Message) AuthorID() uuid.UUID        { return m.authorID }
func (m *Message) Content() string            { return m.content }
func (m *Message) ParentMessageID() uuid.UUID { return m.parentMessageID }
func (m *Message) CreatedAt() time.Time       { return m.createdAt }
func (m *Message) EditedAt() *time.Time       { return m.editedAt }
func (m *Message) IsDeleted() bool            { return m.isDeleted }
func (m *Message) DeletedAt() *time.Time      { return m.deletedAt }

func (m *Message) Attachments() []Attachment {
	attachments := make([]Attachment, len(m.attachments))
	copy(attachments, m.attachments)
	return attachments
}

func (m *Message) Reactions() []Reaction {
	reactions := make([]Reaction, len(m.reactions))
	copy(reactions, m.reactions)
	return reactions
}
```

**Критерии выполнения:**
- [ ] Message aggregate создан с полями
- [ ] NewMessage конструктор с валидацией
- [ ] EditContent() с проверкой прав
- [ ] Delete() с soft delete
- [ ] AddReaction() / RemoveReaction()
- [ ] AddAttachment()
- [ ] Геттеры и проверки состояния

---

### 1.7.3 Message Repository Interface

**Файл:** `internal/domain/message/repository.go`

**Реализация:**
```go
package message

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Pagination параметры пагинации
type Pagination struct {
	Limit  int
	Offset int
}

// Repository определяет интерфейс репозитория сообщений
type Repository interface {
	// FindByID находит сообщение по ID
	FindByID(ctx context.Context, id uuid.UUID) (*Message, error)

	// FindByChatID находит сообщения в чате с пагинацией
	FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*Message, error)

	// FindThread находит все ответы в треде
	FindThread(ctx context.Context, parentMessageID uuid.UUID) ([]*Message, error)

	// CountByChatID возвращает количество сообщений в чате
	CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error)

	// Save сохраняет сообщение
	Save(ctx context.Context, message *Message) error

	// Delete удаляет сообщение (hard delete, используется редко)
	Delete(ctx context.Context, id uuid.UUID) error
}
```

**Критерии выполнения:**
- [ ] Repository interface определен
- [ ] Методы для поиска (FindByID, FindByChatID, FindThread)
- [ ] Pagination struct для пагинации
- [ ] CountByChatID для подсчета
- [ ] Save и Delete методы

---

### 1.7.4 Message Domain Events

**Файл:** `internal/domain/message/events.go`

**Реализация:**
```go
package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	EventTypeMessageCreated          = "message.created"
	EventTypeMessageEdited           = "message.edited"
	EventTypeMessageDeleted          = "message.deleted"
	EventTypeMessageReactionAdded    = "message.reaction.added"
	EventTypeMessageReactionRemoved  = "message.reaction.removed"
	EventTypeMessageAttachmentAdded  = "message.attachment.added"
)

// Created событие создания сообщения
type Created struct {
	event.BaseEvent
	ChatID          uuid.UUID
	AuthorID        uuid.UUID
	Content         string
	ParentMessageID uuid.UUID
	CreatedAt       time.Time
}

// NewCreated создает событие Created
func NewCreated(
	messageID uuid.UUID,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
	parentMessageID uuid.UUID,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:       event.NewBaseEvent(EventTypeMessageCreated, messageID.String(), "Message", 1, metadata),
		ChatID:          chatID,
		AuthorID:        authorID,
		Content:         content,
		ParentMessageID: parentMessageID,
		CreatedAt:       time.Now(),
	}
}

// Edited событие редактирования сообщения
type Edited struct {
	event.BaseEvent
	NewContent string
	EditedAt   time.Time
}

// NewEdited создает событие Edited
func NewEdited(messageID uuid.UUID, newContent string, version int, metadata event.Metadata) *Edited {
	return &Edited{
		BaseEvent:  event.NewBaseEvent(EventTypeMessageEdited, messageID.String(), "Message", version, metadata),
		NewContent: newContent,
		EditedAt:   time.Now(),
	}
}

// Deleted событие удаления сообщения
type Deleted struct {
	event.BaseEvent
	DeletedBy uuid.UUID
	DeletedAt time.Time
}

// NewDeleted создает событие Deleted
func NewDeleted(messageID uuid.UUID, deletedBy uuid.UUID, version int, metadata event.Metadata) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeMessageDeleted, messageID.String(), "Message", version, metadata),
		DeletedBy: deletedBy,
		DeletedAt: time.Now(),
	}
}

// ReactionAdded событие добавления реакции
type ReactionAdded struct {
	event.BaseEvent
	UserID    uuid.UUID
	EmojiCode string
	AddedAt   time.Time
}

// NewReactionAdded создает событие ReactionAdded
func NewReactionAdded(messageID uuid.UUID, userID uuid.UUID, emojiCode string, version int, metadata event.Metadata) *ReactionAdded {
	return &ReactionAdded{
		BaseEvent: event.NewBaseEvent(EventTypeMessageReactionAdded, messageID.String(), "Message", version, metadata),
		UserID:    userID,
		EmojiCode: emojiCode,
		AddedAt:   time.Now(),
	}
}

// ReactionRemoved событие удаления реакции
type ReactionRemoved struct {
	event.BaseEvent
	UserID    uuid.UUID
	EmojiCode string
	RemovedAt time.Time
}

// NewReactionRemoved создает событие ReactionRemoved
func NewReactionRemoved(messageID uuid.UUID, userID uuid.UUID, emojiCode string, version int, metadata event.Metadata) *ReactionRemoved {
	return &ReactionRemoved{
		BaseEvent: event.NewBaseEvent(EventTypeMessageReactionRemoved, messageID.String(), "Message", version, metadata),
		UserID:    userID,
		EmojiCode: emojiCode,
		RemovedAt: time.Now(),
	}
}

// AttachmentAdded событие добавления вложения
type AttachmentAdded struct {
	event.BaseEvent
	FileID   uuid.UUID
	FileName string
	FileSize int64
	MimeType string
	AddedAt  time.Time
}

// NewAttachmentAdded создает событие AttachmentAdded
func NewAttachmentAdded(
	messageID uuid.UUID,
	fileID uuid.UUID,
	fileName string,
	fileSize int64,
	mimeType string,
	version int,
	metadata event.Metadata,
) *AttachmentAdded {
	return &AttachmentAdded{
		BaseEvent: event.NewBaseEvent(EventTypeMessageAttachmentAdded, messageID.String(), "Message", version, metadata),
		FileID:    fileID,
		FileName:  fileName,
		FileSize:  fileSize,
		MimeType:  mimeType,
		AddedAt:   time.Now(),
	}
}
```

**Критерии выполнения:**
- [ ] 6 domain events определены (Created, Edited, Deleted, ReactionAdded, ReactionRemoved, AttachmentAdded)
- [ ] Конструкторы с метаданными
- [ ] Все события реализуют event.DomainEvent

---

### 1.7.5 Message Unit Tests

**Файл:** `internal/domain/message/message_test.go`

**Тесты:**
- NewMessage() создание с валидацией
- EditContent() редактирование с проверкой прав
- Delete() мягкое удаление
- AddReaction() / RemoveReaction() управление реакциями
- AddAttachment() добавление вложений
- Edge cases:
  - Редактирование чужого сообщения (ErrForbidden)
  - Редактирование удаленного сообщения (ErrInvalidState)
  - Дублирующиеся реакции (ErrAlreadyExists)
  - Пустой контент (ErrInvalidInput)

**Критерии выполнения:**
- [ ] Тесты для всех методов Message
- [ ] Тесты для value objects (Attachment, Reaction)
- [ ] Edge cases покрыты
- [ ] Coverage > 85%

---

## Deliverable

После выполнения всех подзадач должно быть готово:

✅ **Message Aggregate**
- Полноценная сущность сообщения с бизнес-логикой
- Редактирование с отслеживанием времени
- Мягкое удаление
- Поддержка тредов (replies)

✅ **Value Objects**
- Attachment для файловых вложений
- Reaction для эмоджи реакций

✅ **Repository Interface**
- Методы поиска с пагинацией
- Поддержка тредов (FindThread)

✅ **Domain Events**
- 6 событий для отслеживания изменений
- Интеграция с event bus

✅ **Unit Tests**
- Coverage > 85%
- Все edge cases покрыты

---

## Проверка выполнения

```bash
# 1. Проверка структуры
ls -la internal/domain/message/

# 2. Проверка компиляции
go build ./internal/domain/message/...

# 3. Запуск unit tests
go test ./internal/domain/message/... -v -cover

# 4. Проверка покрытия
go test ./internal/domain/message/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o message_coverage.html

# 5. Линтинг
golangci-lint run ./internal/domain/message/...
```

Все команды должны выполняться успешно, покрытие > 85%.

---

## Следующие шаги

После завершения Message Aggregate Phase 1 будет полностью завершена:
- 6 aggregates (User, Workspace, Notification, Task, Chat, **Message**)
- Полная доменная модель для MVP

Далее: **Phase 2: Application Layer** — Use Cases, Command/Query handlers.

---

## Примечания

- **Soft Delete:** Message.Delete() не удаляет физически, только устанавливает флаг
- **Thread Support:** ParentMessageID позволяет создавать ветки обсуждений
- **Права доступа:** Только автор может редактировать/удалять
- **Реакции:** Один пользователь — одна реакция на эмоджи
- **Attachments:** Ссылка на файлы через FileID (сами файлы хранятся отдельно)

**Важно:** Как и все domain entities, Message не зависит от инфраструктуры.
