package notification

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Type represents type uvedomleniya
type Type string

const (
	// TypeTaskStatusChanged notification ob izmenenii status tasks
	TypeTaskStatusChanged Type = "task.status_changed"
	// TypeTaskAssigned notification o value tasks
	TypeTaskAssigned Type = "task.assigned"
	// TypeTaskCreated notification o sozdanii tasks
	TypeTaskCreated Type = "task.created"
	// TypeChatMention notification ob upominanii in chate
	TypeChatMention Type = "chat.mention"
	// TypeChatMessage notification o novom soobschenii in chate
	TypeChatMessage Type = "chat.message"
	// TypeWorkspaceInvite notification o priglashenii in workspace
	TypeWorkspaceInvite Type = "workspace.invite"
	// TypeSystem sistemnoe notification
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

// NewNotification creates new notification
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

// Reconstruct reconstructs notification from save.
// Used by repositories for hydration obekta without validation business rules.
// all parameters dolzhny byt valid values from save.
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

// MarkAsRead pomechaet notification as prochitannoe
func (n *Notification) MarkAsRead() error {
	if n.readAt != nil {
		return errs.ErrInvalidState
	}
	now := time.Now()
	n.readAt = &now
	return nil
}

// IsRead checks, prochitano li notification
func (n *Notification) IsRead() bool {
	return n.readAt != nil
}

// ID returns ID uvedomleniya
func (n *Notification) ID() uuid.UUID { return n.id }

// UserID returns ID user
func (n *Notification) UserID() uuid.UUID { return n.userID }

// Type returns type uvedomleniya
func (n *Notification) Type() Type { return n.typ }

// Title returns zagolovok uvedomleniya
func (n *Notification) Title() string { return n.title }

// Message returns text uvedomleniya
func (n *Notification) Message() string { return n.message }

// ResourceID returns ID svyazannogo resursa
func (n *Notification) ResourceID() string { return n.resourceID }

// ReadAt returns time prochteniya
func (n *Notification) ReadAt() *time.Time { return n.readAt }

// CreatedAt returns creation time
func (n *Notification) CreatedAt() time.Time { return n.createdAt }
