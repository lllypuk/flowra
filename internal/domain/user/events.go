package user

import (
	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

const (
	// EventTypeUserCreated тип события создания пользователя
	EventTypeUserCreated = "user.created"
	// EventTypeUserUpdated тип события обновления пользователя
	EventTypeUserUpdated = "user.updated"
	// EventTypeUserDeleted тип события удаления пользователя
	EventTypeUserDeleted = "user.deleted"
	// EventTypeAdminRightsChanged тип события изменения прав администратора
	EventTypeAdminRightsChanged = "user.admin_rights_changed"
)

// Created событие создания пользователя
type Created struct {
	event.BaseEvent

	Username    string
	Email       string
	DisplayName string
}

// NewUserCreated создает событие UserCreated
func NewUserCreated(
	userID uuid.UUID,
	username, email, displayName string,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:   event.NewBaseEvent(EventTypeUserCreated, userID.String(), "User", 1, metadata),
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}
}

// Updated событие обновления пользователя
type Updated struct {
	event.BaseEvent

	DisplayName string
}

// NewUserUpdated создает событие UserUpdated
func NewUserUpdated(userID uuid.UUID, displayName string, version int, metadata event.Metadata) *Updated {
	return &Updated{
		BaseEvent:   event.NewBaseEvent(EventTypeUserUpdated, userID.String(), "User", version, metadata),
		DisplayName: displayName,
	}
}

// Deleted событие удаления пользователя
type Deleted struct {
	event.BaseEvent
}

// NewUserDeleted создает событие UserDeleted
func NewUserDeleted(userID uuid.UUID, version int, metadata event.Metadata) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeUserDeleted, userID.String(), "User", version, metadata),
	}
}

// AdminRightsChanged событие изменения прав администратора
type AdminRightsChanged struct {
	event.BaseEvent

	IsAdmin bool
}

// NewAdminRightsChanged создает событие AdminRightsChanged
func NewAdminRightsChanged(
	userID uuid.UUID,
	isAdmin bool,
	version int,
	metadata event.Metadata,
) *AdminRightsChanged {
	return &AdminRightsChanged{
		BaseEvent: event.NewBaseEvent(EventTypeAdminRightsChanged, userID.String(), "User", version, metadata),
		IsAdmin:   isAdmin,
	}
}
