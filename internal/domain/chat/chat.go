package chat

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Type представляет тип чата
type Type string

const (
	// TypeDiscussion обычное обсуждение
	TypeDiscussion Type = "discussion"
	// TypeTask чат-задача
	TypeTask Type = "task"
	// TypeBug чат-баг
	TypeBug Type = "bug"
	// TypeEpic чат-эпик
	TypeEpic Type = "epic"
)

// Chat представляет чат aggregate root с Event Sourcing
type Chat struct {
	id           uuid.UUID
	workspaceID  uuid.UUID
	chatType     Type
	isPublic     bool
	createdBy    uuid.UUID
	createdAt    time.Time
	participants []Participant

	// Event sourcing
	version           int
	uncommittedEvents []event.DomainEvent
}

// NewChat создает новый чат
func NewChat(
	workspaceID uuid.UUID,
	chatType Type,
	isPublic bool,
	createdBy uuid.UUID,
) (*Chat, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if createdBy.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if !isValidChatType(chatType) {
		return nil, errs.ErrInvalidInput
	}

	chat := &Chat{
		id:                uuid.NewUUID(),
		workspaceID:       workspaceID,
		chatType:          chatType,
		isPublic:          isPublic,
		createdBy:         createdBy,
		createdAt:         time.Now(),
		participants:      make([]Participant, 0),
		version:           0,
		uncommittedEvents: make([]event.DomainEvent, 0),
	}

	// Создатель автоматически становится admin
	chat.addParticipantInternal(createdBy, RoleAdmin)

	return chat, nil
}

// AddParticipant добавляет участника в чат
func (c *Chat) AddParticipant(userID uuid.UUID, role Role) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}
	if c.HasParticipant(userID) {
		return errs.ErrAlreadyExists
	}

	c.addParticipantInternal(userID, role)
	return nil
}

func (c *Chat) addParticipantInternal(userID uuid.UUID, role Role) {
	participant := NewParticipant(userID, role)
	c.participants = append(c.participants, participant)
}

// RemoveParticipant удаляет участника из чата
func (c *Chat) RemoveParticipant(userID uuid.UUID) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}
	if !c.HasParticipant(userID) {
		return errs.ErrNotFound
	}
	if userID == c.createdBy {
		return errs.ErrInvalidInput // Создатель не может покинуть чат
	}

	newParticipants := make([]Participant, 0, len(c.participants)-1)
	for _, p := range c.participants {
		if p.UserID() != userID {
			newParticipants = append(newParticipants, p)
		}
	}
	c.participants = newParticipants
	return nil
}

// ConvertToTask конвертирует Discussion в Task/Bug/Epic
func (c *Chat) ConvertToTask(newType Type, title string) error {
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if !isValidTaskType(newType) {
		return errs.ErrInvalidInput
	}
	if title == "" {
		return errs.ErrInvalidInput
	}

	c.chatType = newType
	return nil
}

// HasParticipant проверяет, является ли пользователь участником
func (c *Chat) HasParticipant(userID uuid.UUID) bool {
	for _, p := range c.participants {
		if p.UserID() == userID {
			return true
		}
	}
	return false
}

// IsParticipantAdmin проверяет, является ли участник администратором
func (c *Chat) IsParticipantAdmin(userID uuid.UUID) bool {
	for _, p := range c.participants {
		if p.UserID() == userID && p.IsAdmin() {
			return true
		}
	}
	return false
}

// FindParticipant находит участника по ID
func (c *Chat) FindParticipant(userID uuid.UUID) *Participant {
	for _, p := range c.participants {
		if p.UserID() == userID {
			pCopy := p
			return &pCopy
		}
	}
	return nil
}

// IsTyped проверяет, является ли чат типизированным (не Discussion)
func (c *Chat) IsTyped() bool {
	return c.chatType != TypeDiscussion
}

// GetTaskEntityType возвращает соответствующий тип TaskEntity
func (c *Chat) GetTaskEntityType() (task.EntityType, error) {
	switch c.chatType {
	case TypeTask:
		return task.TypeTask, nil
	case TypeBug:
		return task.TypeBug, nil
	case TypeEpic:
		return task.TypeEpic, nil
	case TypeDiscussion:
		return task.TypeDiscussion, nil
	default:
		return "", errs.ErrInvalidState
	}
}

// Event Sourcing methods

// Apply применяет событие для восстановления состояния
func (c *Chat) Apply(e event.DomainEvent) error {
	switch evt := e.(type) {
	case *Created:
		c.id = uuid.UUID(evt.AggregateID())
		c.workspaceID = evt.WorkspaceID
		c.chatType = evt.Type
		c.isPublic = evt.IsPublic
		c.createdBy = evt.CreatedBy
		c.createdAt = evt.CreatedAt
		c.version = evt.Version()
	case *ParticipantAdded:
		c.addParticipantInternal(evt.UserID, evt.Role)
		c.version = evt.Version()
	case *TypeChanged:
		c.chatType = evt.NewType
		c.version = evt.Version()
	default:
		// Неизвестные события игнорируем (forward compatibility)
		return nil
	}
	return nil
}

// GetUncommittedEvents возвращает новые события
func (c *Chat) GetUncommittedEvents() []event.DomainEvent {
	return c.uncommittedEvents
}

// MarkEventsAsCommitted помечает события как зафиксированные
func (c *Chat) MarkEventsAsCommitted() {
	c.uncommittedEvents = make([]event.DomainEvent, 0)
}

// Getters

// ID возвращает ID чата
func (c *Chat) ID() uuid.UUID { return c.id }

// WorkspaceID возвращает ID workspace
func (c *Chat) WorkspaceID() uuid.UUID { return c.workspaceID }

// Type возвращает тип чата
func (c *Chat) Type() Type { return c.chatType }

// IsPublic возвращает признак публичности
func (c *Chat) IsPublic() bool { return c.isPublic }

// CreatedBy возвращает ID создателя
func (c *Chat) CreatedBy() uuid.UUID { return c.createdBy }

// CreatedAt возвращает время создания
func (c *Chat) CreatedAt() time.Time { return c.createdAt }

// Participants возвращает копию списка участников
func (c *Chat) Participants() []Participant {
	participants := make([]Participant, len(c.participants))
	copy(participants, c.participants)
	return participants
}

// Version возвращает версию aggregate для optimistic locking
func (c *Chat) Version() int { return c.version }

// Validation helpers

func isValidChatType(t Type) bool {
	return t == TypeDiscussion || t == TypeTask || t == TypeBug || t == TypeEpic
}

func isValidTaskType(t Type) bool {
	return t == TypeTask || t == TypeBug || t == TypeEpic
}
