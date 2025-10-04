package user

import (
	"github.com/lllypuk/teams-up/internal/domain/common"
	"github.com/lllypuk/teams-up/internal/domain/event"
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

// UserCreated событие создания пользователя
type UserCreated struct {
	event.BaseEvent
	Username    string
	Email       string
	DisplayName string
}

// NewUserCreated создает событие UserCreated
func NewUserCreated(userID common.UUID, username, email, displayName string, metadata event.EventMetadata) *UserCreated {
	return &UserCreated{
		BaseEvent:   event.NewBaseEvent(EventTypeUserCreated, userID.String(), "User", 1, metadata),
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}
}

// UserUpdated событие обновления пользователя
type UserUpdated struct {
	event.BaseEvent
	DisplayName string
}

// NewUserUpdated создает событие UserUpdated
func NewUserUpdated(userID common.UUID, displayName string, version int, metadata event.EventMetadata) *UserUpdated {
	return &UserUpdated{
		BaseEvent:   event.NewBaseEvent(EventTypeUserUpdated, userID.String(), "User", version, metadata),
		DisplayName: displayName,
	}
}

// UserDeleted событие удаления пользователя
type UserDeleted struct {
	event.BaseEvent
}

// NewUserDeleted создает событие UserDeleted
func NewUserDeleted(userID common.UUID, version int, metadata event.EventMetadata) *UserDeleted {
	return &UserDeleted{
		BaseEvent: event.NewBaseEvent(EventTypeUserDeleted, userID.String(), "User", version, metadata),
	}
}

// AdminRightsChanged событие изменения прав администратора
type AdminRightsChanged struct {
	event.BaseEvent
	IsAdmin bool
}

// NewAdminRightsChanged создает событие AdminRightsChanged
func NewAdminRightsChanged(userID common.UUID, isAdmin bool, version int, metadata event.EventMetadata) *AdminRightsChanged {
	return &AdminRightsChanged{
		BaseEvent: event.NewBaseEvent(EventTypeAdminRightsChanged, userID.String(), "User", version, metadata),
		IsAdmin:   isAdmin,
	}
}
