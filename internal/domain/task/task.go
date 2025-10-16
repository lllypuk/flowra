package task

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Entity представляет типизированную сущность (Task/Bug/Epic)
// ID Entity всегда равен ID соответствующего Chat
type Entity struct {
	id           uuid.UUID
	chatID       uuid.UUID
	title        string
	state        *EntityState
	assignedTo   *uuid.UUID
	dueDate      *time.Time
	customFields map[string]string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewTask создает новую задачу
func NewTask(chatID uuid.UUID, title string, entityType EntityType) (*Entity, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if title == "" {
		return nil, errs.ErrInvalidInput
	}

	state, err := NewEntityState(entityType)
	if err != nil {
		return nil, err
	}

	return &Entity{
		id:           chatID, // TaskEntity.ID == Chat.ID
		chatID:       chatID,
		title:        title,
		state:        state,
		assignedTo:   nil,
		dueDate:      nil,
		customFields: make(map[string]string),
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}, nil
}

// ChangeStatus изменяет статус задачи
func (t *Entity) ChangeStatus(newStatus Status) error {
	err := t.state.ChangeStatus(newStatus)
	if err != nil {
		return err
	}
	t.updatedAt = time.Now()
	return nil
}

// Assign назначает задачу на пользователя
func (t *Entity) Assign(userID uuid.UUID) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}
	t.assignedTo = &userID
	t.updatedAt = time.Now()
	return nil
}

// Unassign снимает назначение задачи
func (t *Entity) Unassign() {
	t.assignedTo = nil
	t.updatedAt = time.Now()
}

// SetPriority устанавливает приоритет задачи
func (t *Entity) SetPriority(priority Priority) error {
	err := t.state.SetPriority(priority)
	if err != nil {
		return err
	}
	t.updatedAt = time.Now()
	return nil
}

// SetDueDate устанавливает срок выполнения
func (t *Entity) SetDueDate(dueDate time.Time) error {
	if dueDate.Before(time.Now()) {
		return errs.ErrInvalidInput
	}
	t.dueDate = &dueDate
	t.updatedAt = time.Now()
	return nil
}

// ClearDueDate очищает срок выполнения
func (t *Entity) ClearDueDate() {
	t.dueDate = nil
	t.updatedAt = time.Now()
}

// SetCustomField устанавливает кастомное поле
func (t *Entity) SetCustomField(key, value string) error {
	if key == "" {
		return errs.ErrInvalidInput
	}
	if value == "" {
		// Пустое значение = удаление поля
		delete(t.customFields, key)
	} else {
		t.customFields[key] = value
	}
	t.updatedAt = time.Now()
	return nil
}

// UpdateTitle обновляет заголовок задачи
func (t *Entity) UpdateTitle(title string) error {
	if title == "" {
		return errs.ErrInvalidInput
	}
	t.title = title
	t.updatedAt = time.Now()
	return nil
}

// IsOverdue проверяет, просрочена ли задача
func (t *Entity) IsOverdue() bool {
	if t.dueDate == nil {
		return false
	}
	if t.state.Status() == StatusDone || t.state.Status() == StatusCancelled {
		return false
	}
	return time.Now().After(*t.dueDate)
}

// Getters

// ID возвращает ID задачи (равен ChatID)
func (t *Entity) ID() uuid.UUID { return t.id }

// ChatID возвращает ID связанного чата
func (t *Entity) ChatID() uuid.UUID { return t.chatID }

// Title возвращает заголовок задачи
func (t *Entity) Title() string { return t.title }

// State возвращает состояние задачи
func (t *Entity) State() *EntityState { return t.state }

// Type возвращает тип задачи
func (t *Entity) Type() EntityType { return t.state.Type() }

// Status возвращает статус задачи
func (t *Entity) Status() Status { return t.state.Status() }

// Priority возвращает приоритет задачи
func (t *Entity) Priority() Priority { return t.state.Priority() }

// AssignedTo возвращает ID назначенного пользователя
func (t *Entity) AssignedTo() *uuid.UUID { return t.assignedTo }

// DueDate возвращает срок выполнения
func (t *Entity) DueDate() *time.Time { return t.dueDate }

// CustomFields возвращает кастомные поля
func (t *Entity) CustomFields() map[string]string {
	// Возвращаем копию чтобы избежать внешних изменений
	fields := make(map[string]string, len(t.customFields))
	for k, v := range t.customFields {
		fields[k] = v
	}
	return fields
}

// CreatedAt возвращает время создания
func (t *Entity) CreatedAt() time.Time { return t.createdAt }

// UpdatedAt возвращает время последнего обновления
func (t *Entity) UpdatedAt() time.Time { return t.updatedAt }
