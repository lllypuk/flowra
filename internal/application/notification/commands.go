package notification

import (
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Command bazovyy interface commands
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

// MarkAsReadCommand - pometka as prochitannoe
type MarkAsReadCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID // check that notification prinadlezhit user
}

func (c MarkAsReadCommand) CommandName() string { return "MarkAsRead" }

// MarkAllAsReadCommand - pometka all as prochitannye
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
