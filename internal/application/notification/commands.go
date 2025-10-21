package notification

import (
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Command базовый интерфейс команд
type Command interface {
	CommandName() string
}

// CreateNotificationCommand - создание notification
type CreateNotificationCommand struct {
	UserID     uuid.UUID
	Type       notification.Type
	Title      string
	Message    string
	ResourceID string // ID задачи/чата/workspace
}

func (c CreateNotificationCommand) CommandName() string { return "CreateNotification" }

// MarkAsReadCommand - пометка как прочитанное
type MarkAsReadCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID // проверка, что notification принадлежит пользователю
}

func (c MarkAsReadCommand) CommandName() string { return "MarkAsRead" }

// MarkAllAsReadCommand - пометка всех как прочитанные
type MarkAllAsReadCommand struct {
	UserID uuid.UUID
}

func (c MarkAllAsReadCommand) CommandName() string { return "MarkAllAsRead" }

// DeleteNotificationCommand - удаление notification
type DeleteNotificationCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

func (c DeleteNotificationCommand) CommandName() string { return "DeleteNotification" }
