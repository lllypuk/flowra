package notification

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Event types
const (
	EventTypeNotificationCreated = "notification.created"
	EventTypeNotificationRead    = "notification.read"
	EventTypeNotificationDeleted = "notification.deleted"
)

// Created event creating uvedomleniya
type Created struct {
	event.BaseEvent

	UserID     uuid.UUID
	Type       Type
	Title      string
	Message    string
	ResourceID string
}

// NewNotificationCreated creates new event NotificationCreated
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

// Read event prochteniya uvedomleniya
type Read struct {
	event.BaseEvent

	UserID uuid.UUID
	ReadAt time.Time
}

// NewNotificationRead creates new event NotificationRead
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

// Deleted event removing uvedomleniya
type Deleted struct {
	event.BaseEvent

	UserID uuid.UUID
}

// NewNotificationDeleted creates new event NotificationDeleted
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
