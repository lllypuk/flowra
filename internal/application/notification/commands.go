package notification

import (
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Command базовый interface commands
type Command interface {
	CommandName() string
}

// CreateNotificationCommand - notification creation
type CreateNotificationCommand struct {
	UserID     uuid.UUID
	Type       notification.Type
	Title      string
	Message    string
	ResourceID string // ID tasks/chat/workspace
}

func (c CreateNotificationCommand) CommandName() string { return "CreateNotification" }

// MarkAsReadCommand - пометка as прочитанное
type MarkAsReadCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID // check, that notification принадлежит user
}

func (c MarkAsReadCommand) CommandName() string { return "MarkAsRead" }

// MarkAllAsReadCommand - пометка all as прочитанные
type MarkAllAsReadCommand struct {
	UserID uuid.UUID
}

func (c MarkAllAsReadCommand) CommandName() string { return "MarkAllAsRead" }

// DeleteNotificationCommand - deletion notification
type DeleteNotificationCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

func (c DeleteNotificationCommand) CommandName() string { return "DeleteNotification" }
