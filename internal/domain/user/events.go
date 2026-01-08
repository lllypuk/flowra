package user

import (
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	// EventTypeUserCreated type event creating user
	EventTypeUserCreated = "user.created"
	// EventTypeUserUpdated type event updating user
	EventTypeUserUpdated = "user.updated"
	// EventTypeUserDeleted type event removing user
	EventTypeUserDeleted = "user.deleted"
	// EventTypeAdminRightsChanged type event changing прав administratorа
	EventTypeAdminRightsChanged = "user.admin_rights_changed"
)

// Created event creating user
type Created struct {
	event.BaseEvent

	Username    string
	Email       string
	DisplayName string
}

// NewUserCreated creates event UserCreated
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

// Updated event updating user
type Updated struct {
	event.BaseEvent

	DisplayName string
}

// NewUserUpdated creates event UserUpdated
func NewUserUpdated(userID uuid.UUID, displayName string, version int, metadata event.Metadata) *Updated {
	return &Updated{
		BaseEvent:   event.NewBaseEvent(EventTypeUserUpdated, userID.String(), "User", version, metadata),
		DisplayName: displayName,
	}
}

// Deleted event removing user
type Deleted struct {
	event.BaseEvent
}

// NewUserDeleted creates event UserDeleted
func NewUserDeleted(userID uuid.UUID, version int, metadata event.Metadata) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeUserDeleted, userID.String(), "User", version, metadata),
	}
}

// AdminRightsChanged event changing прав administratorа
type AdminRightsChanged struct {
	event.BaseEvent

	IsAdmin bool
}

// NewAdminRightsChanged creates event AdminRightsChanged
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
