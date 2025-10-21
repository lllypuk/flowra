package notification

import (
	"time"

	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// Event types
const (
	EventTypeNotificationCreated = "notification.created"
	EventTypeNotificationRead    = "notification.read"
	EventTypeNotificationDeleted = "notification.deleted"
)

// Created событие создания уведомления
type Created struct {
	event.BaseEvent

	UserID     uuid.UUID
	Type       Type
	Title      string
	Message    string
	ResourceID string
}

// NewNotificationCreated создает новое событие NotificationCreated
func NewNotificationCreated(
	notificationID, userID uuid.UUID,
	typ Type,
	title, message, resourceID string,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent: event.NewBaseEvent(
			EventTypeNotificationCreated,
			notificationID.String(),
			"Notification",
			1,
			metadata,
		),
		UserID:     userID,
		Type:       typ,
		Title:      title,
		Message:    message,
		ResourceID: resourceID,
	}
}

// Read событие прочтения уведомления
type Read struct {
	event.BaseEvent

	UserID uuid.UUID
	ReadAt time.Time
}

// NewNotificationRead создает новое событие NotificationRead
func NewNotificationRead(
	notificationID, userID uuid.UUID,
	readAt time.Time,
	metadata event.Metadata,
) *Read {
	return &Read{
		BaseEvent: event.NewBaseEvent(EventTypeNotificationRead, notificationID.String(), "Notification", 1, metadata),
		UserID:    userID,
		ReadAt:    readAt,
	}
}

// Deleted событие удаления уведомления
type Deleted struct {
	event.BaseEvent

	UserID uuid.UUID
}

// NewNotificationDeleted создает новое событие NotificationDeleted
func NewNotificationDeleted(
	notificationID, userID uuid.UUID,
	metadata event.Metadata,
) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(
			EventTypeNotificationDeleted,
			notificationID.String(),
			"Notification",
			1,
			metadata,
		),
		UserID: userID,
	}
}
