package chat

import (
	"slices"
	"time"

	"github.com/flowra/flowra/internal/domain/errs"
	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/task"
	"github.com/flowra/flowra/internal/domain/uuid"
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

	// Поля для typed чатов (Task/Bug/Epic)
	title      string
	status     string
	priority   string
	assigneeID *uuid.UUID
	dueDate    *time.Time
	severity   string // только для Bug

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

// ConvertToTask конвертирует Discussion в Task
func (c *Chat) ConvertToTask(title string, userID uuid.UUID) error {
	// Валидация
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	// Создание события
	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeTask,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	// Применение и сохранение события
	c.applyEvent(evt)
	return nil
}

// ConvertToBug конвертирует Discussion в Bug
func (c *Chat) ConvertToBug(title string, userID uuid.UUID) error {
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeBug,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// ConvertToEpic конвертирует Discussion в Epic
func (c *Chat) ConvertToEpic(title string, userID uuid.UUID) error {
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeEpic,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// ====== Entity Management Methods ======

// ChangeStatus изменяет статус typed чата
func (c *Chat) ChangeStatus(newStatus string, userID uuid.UUID) error {
	// Валидация: только для typed чатов
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Валидация статуса
	if err := c.validateStatus(newStatus); err != nil {
		return err
	}

	// Если статус не изменился
	if c.status == newStatus {
		return nil
	}

	oldStatus := c.status

	evt := NewStatusChanged(
		c.id,
		oldStatus,
		newStatus,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// AssignUser назначает исполнителя
func (c *Chat) AssignUser(assigneeID *uuid.UUID, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Снятие assignee
	if assigneeID == nil {
		if c.assigneeID == nil {
			return nil // Уже нет assignee
		}

		evt := NewAssigneeRemoved(
			c.id,
			*c.assigneeID,
			userID,
			c.version+1,
			event.Metadata{
				CorrelationID: uuid.NewUUID().String(),
				CausationID:   uuid.NewUUID().String(),
				UserID:        userID.String(),
			},
		)
		c.applyEvent(evt)
		return nil
	}

	// Проверка: не назначаем того же пользователя
	if c.assigneeID != nil && *c.assigneeID == *assigneeID {
		return nil
	}

	// Назначение assignee
	evt := NewUserAssigned(
		c.id,
		*assigneeID,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)
	c.applyEvent(evt)
	return nil
}

// SetPriority устанавливает приоритет
func (c *Chat) SetPriority(priority string, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	if err := c.validatePriority(priority); err != nil {
		return err
	}

	if c.priority == priority {
		return nil
	}

	oldPriority := c.priority

	evt := NewPrioritySet(
		c.id,
		oldPriority,
		priority,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// SetDueDate устанавливает или снимает дедлайн
func (c *Chat) SetDueDate(dueDate *time.Time, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Снятие due date
	if dueDate == nil {
		if c.dueDate == nil {
			return nil
		}

		evt := NewDueDateRemoved(
			c.id,
			*c.dueDate,
			userID,
			c.version+1,
			event.Metadata{
				CorrelationID: uuid.NewUUID().String(),
				CausationID:   uuid.NewUUID().String(),
				UserID:        userID.String(),
			},
		)
		c.applyEvent(evt)
		return nil
	}

	// Проверка: не устанавливаем ту же дату
	if c.dueDate != nil && c.dueDate.Equal(*dueDate) {
		return nil
	}

	// Установка due date
	evt := NewDueDateSet(
		c.id,
		c.dueDate,
		*dueDate,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)
	c.applyEvent(evt)
	return nil
}

// Rename изменяет название чата
func (c *Chat) Rename(newTitle string, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	if newTitle == "" {
		return errs.ErrInvalidInput
	}

	if c.title == newTitle {
		return nil
	}

	oldTitle := c.title

	evt := NewChatRenamed(
		c.id,
		oldTitle,
		newTitle,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// SetSeverity устанавливает severity для Bug
func (c *Chat) SetSeverity(severity string, userID uuid.UUID) error {
	if c.chatType != TypeBug {
		return errs.ErrInvalidState
	}

	if err := c.validateSeverity(severity); err != nil {
		return err
	}

	if c.severity == severity {
		return nil
	}

	oldSeverity := c.severity

	evt := NewSeveritySet(
		c.id,
		oldSeverity,
		severity,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
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
		c.title = evt.Title
		// Устанавливаем дефолтный статус
		c.status = c.getDefaultStatus()
		c.version = evt.Version()

	case *StatusChanged:
		c.status = evt.NewStatus
		c.version = evt.Version()

	case *UserAssigned:
		assigneeID := evt.AssigneeID
		c.assigneeID = &assigneeID
		c.version = evt.Version()

	case *AssigneeRemoved:
		c.assigneeID = nil
		c.version = evt.Version()

	case *PrioritySet:
		c.priority = evt.NewPriority
		c.version = evt.Version()

	case *DueDateSet:
		dueDate := evt.NewDueDate
		c.dueDate = &dueDate
		c.version = evt.Version()

	case *DueDateRemoved:
		c.dueDate = nil
		c.version = evt.Version()

	case *Renamed:
		c.title = evt.NewTitle
		c.version = evt.Version()

	case *SeveritySet:
		c.severity = evt.NewSeverity
		c.version = evt.Version()

	default:
		// Неизвестные события игнорируем (forward compatibility)
		return nil
	}
	return nil
}

// getDefaultStatus возвращает дефолтный статус для типа чата
func (c *Chat) getDefaultStatus() string {
	switch c.chatType {
	case TypeTask:
		return "To Do"
	case TypeBug:
		return "New"
	case TypeEpic:
		return "Planned"
	case TypeDiscussion:
		return ""
	default:
		return ""
	}
}

// applyEvent применяет событие и добавляет в uncommitted
func (c *Chat) applyEvent(evt event.DomainEvent) {
	_ = c.Apply(evt)
	c.uncommittedEvents = append(c.uncommittedEvents, evt)
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

// Title возвращает название typed чата
func (c *Chat) Title() string { return c.title }

// Status возвращает статус typed чата
func (c *Chat) Status() string { return c.status }

// Priority возвращает приоритет
func (c *Chat) Priority() string { return c.priority }

// AssigneeID возвращает ID назначенного пользователя
func (c *Chat) AssigneeID() *uuid.UUID { return c.assigneeID }

// DueDate возвращает дедлайн
func (c *Chat) DueDate() *time.Time { return c.dueDate }

// Severity возвращает severity (для Bug)
func (c *Chat) Severity() string { return c.severity }

// Validation helpers

// validateStatus проверяет валидность статуса для текущего типа чата
func (c *Chat) validateStatus(status string) error {
	var validStatuses []string

	switch c.chatType {
	case TypeTask:
		validStatuses = []string{"To Do", "In Progress", "Done"}
	case TypeBug:
		validStatuses = []string{"New", "Investigating", "Fixed", "Verified"}
	case TypeEpic:
		validStatuses = []string{"Planned", "In Progress", "Completed"}
	case TypeDiscussion:
		return errs.ErrInvalidState
	default:
		return errs.ErrInvalidState
	}

	if slices.Contains(validStatuses, status) {
		return nil
	}

	return errs.ErrInvalidInput
}

// validatePriority проверяет валидность приоритета
func (c *Chat) validatePriority(priority string) error {
	validPriorities := []string{"Low", "Medium", "High", "Critical"}

	if slices.Contains(validPriorities, priority) {
		return nil
	}

	return errs.ErrInvalidInput
}

// validateSeverity проверяет валидность severity для Bug
func (c *Chat) validateSeverity(severity string) error {
	validSeverities := []string{"Minor", "Major", "Critical", "Blocker"}

	if slices.Contains(validSeverities, severity) {
		return nil
	}

	return errs.ErrInvalidInput
}

func isValidChatType(t Type) bool {
	return t == TypeDiscussion || t == TypeTask || t == TypeBug || t == TypeEpic
}
