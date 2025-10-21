package notification

import (
	"time"

	"github.com/flowra/flowra/internal/domain/errs"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// Type представляет тип уведомления
type Type string

const (
	// TypeTaskStatusChanged уведомление об изменении статуса задачи
	TypeTaskStatusChanged Type = "task.status_changed"
	// TypeTaskAssigned уведомление о назначении задачи
	TypeTaskAssigned Type = "task.assigned"
	// TypeTaskCreated уведомление о создании задачи
	TypeTaskCreated Type = "task.created"
	// TypeChatMention уведомление об упоминании в чате
	TypeChatMention Type = "chat.mention"
	// TypeChatMessage уведомление о новом сообщении в чате
	TypeChatMessage Type = "chat.message"
	// TypeWorkspaceInvite уведомление о приглашении в workspace
	TypeWorkspaceInvite Type = "workspace.invite"
	// TypeSystem системное уведомление
	TypeSystem Type = "system"
)

// Notification представляет уведомление для пользователя
type Notification struct {
	id         uuid.UUID
	userID     uuid.UUID
	typ        Type
	title      string
	message    string
	resourceID string
	readAt     *time.Time
	createdAt  time.Time
}

// NewNotification создает новое уведомление
func NewNotification(
	userID uuid.UUID,
	typ Type,
	title, message string,
	resourceID string,
) (*Notification, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if typ == "" {
		return nil, errs.ErrInvalidInput
	}
	if title == "" {
		return nil, errs.ErrInvalidInput
	}
	if message == "" {
		return nil, errs.ErrInvalidInput
	}

	return &Notification{
		id:         uuid.NewUUID(),
		userID:     userID,
		typ:        typ,
		title:      title,
		message:    message,
		resourceID: resourceID,
		readAt:     nil,
		createdAt:  time.Now(),
	}, nil
}

// MarkAsRead помечает уведомление как прочитанное
func (n *Notification) MarkAsRead() error {
	if n.readAt != nil {
		return errs.ErrInvalidState
	}
	now := time.Now()
	n.readAt = &now
	return nil
}

// IsRead проверяет, прочитано ли уведомление
func (n *Notification) IsRead() bool {
	return n.readAt != nil
}

// ID возвращает ID уведомления
func (n *Notification) ID() uuid.UUID { return n.id }

// UserID возвращает ID пользователя
func (n *Notification) UserID() uuid.UUID { return n.userID }

// Type возвращает тип уведомления
func (n *Notification) Type() Type { return n.typ }

// Title возвращает заголовок уведомления
func (n *Notification) Title() string { return n.title }

// Message возвращает текст уведомления
func (n *Notification) Message() string { return n.message }

// ResourceID возвращает ID связанного ресурса
func (n *Notification) ResourceID() string { return n.resourceID }

// ReadAt возвращает время прочтения
func (n *Notification) ReadAt() *time.Time { return n.readAt }

// CreatedAt возвращает время создания
func (n *Notification) CreatedAt() time.Time { return n.createdAt }
