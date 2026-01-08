package notification

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Type represents type уведомления
type Type string

const (
	// TypeTaskStatusChanged notification об изменении status tasks
	TypeTaskStatusChanged Type = "task.status_changed"
	// TypeTaskAssigned notification о наvalueении tasks
	TypeTaskAssigned Type = "task.assigned"
	// TypeTaskCreated notification о создании tasks
	TypeTaskCreated Type = "task.created"
	// TypeChatMention notification об упоминании in чате
	TypeChatMention Type = "chat.mention"
	// TypeChatMessage notification о новом сообщении in чате
	TypeChatMessage Type = "chat.message"
	// TypeWorkspaceInvite notification о приглашении in workspace
	TypeWorkspaceInvite Type = "workspace.invite"
	// TypeSystem системное notification
	TypeSystem Type = "system"
)

// Notification represents notification for user
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

// NewNotification creates новое notification
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

// Reconstruct восстанавливает notification from storage.
// Used by repositories for hydration объекта without validation business rules.
// all parameters должны быть valid values from storage.
func Reconstruct(
	id uuid.UUID,
	userID uuid.UUID,
	typ Type,
	title, message string,
	resourceID string,
	readAt *time.Time,
	createdAt time.Time,
) *Notification {
	return &Notification{
		id:         id,
		userID:     userID,
		typ:        typ,
		title:      title,
		message:    message,
		resourceID: resourceID,
		readAt:     readAt,
		createdAt:  createdAt,
	}
}

// MarkAsRead помечает notification as прочитанное
func (n *Notification) MarkAsRead() error {
	if n.readAt != nil {
		return errs.ErrInvalidState
	}
	now := time.Now()
	n.readAt = &now
	return nil
}

// IsRead checks, прочитано ли notification
func (n *Notification) IsRead() bool {
	return n.readAt != nil
}

// ID returns ID уведомления
func (n *Notification) ID() uuid.UUID { return n.id }

// UserID returns ID user
func (n *Notification) UserID() uuid.UUID { return n.userID }

// Type returns type уведомления
func (n *Notification) Type() Type { return n.typ }

// Title returns заголовок уведомления
func (n *Notification) Title() string { return n.title }

// Message returns text уведомления
func (n *Notification) Message() string { return n.message }

// ResourceID returns ID связанного ресурса
func (n *Notification) ResourceID() string { return n.resourceID }

// ReadAt returns time прочтения
func (n *Notification) ReadAt() *time.Time { return n.readAt }

// CreatedAt returns creation time
func (n *Notification) CreatedAt() time.Time { return n.createdAt }
