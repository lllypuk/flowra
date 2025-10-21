package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Event types
const (
	EventTypeWorkspaceCreated = "workspace.created"
	EventTypeWorkspaceUpdated = "workspace.updated"
	EventTypeWorkspaceDeleted = "workspace.deleted"
	EventTypeInviteCreated    = "workspace.invite.created"
	EventTypeInviteUsed       = "workspace.invite.used"
	EventTypeInviteRevoked    = "workspace.invite.revoked"
)

// Created событие создания рабочего пространства
type Created struct {
	event.BaseEvent

	Name            string
	KeycloakGroupID string
	CreatedBy       uuid.UUID
}

// NewWorkspaceCreated создает новое событие WorkspaceCreated
func NewWorkspaceCreated(
	workspaceID uuid.UUID,
	name, keycloakGroupID string,
	createdBy uuid.UUID,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:       event.NewBaseEvent(EventTypeWorkspaceCreated, workspaceID.String(), "Workspace", 1, metadata),
		Name:            name,
		KeycloakGroupID: keycloakGroupID,
		CreatedBy:       createdBy,
	}
}

// Updated событие обновления рабочего пространства
type Updated struct {
	event.BaseEvent

	Name string
}

// NewWorkspaceUpdated создает новое событие WorkspaceUpdated
func NewWorkspaceUpdated(workspaceID uuid.UUID, name string, metadata event.Metadata) *Updated {
	return &Updated{
		BaseEvent: event.NewBaseEvent(EventTypeWorkspaceUpdated, workspaceID.String(), "Workspace", 1, metadata),
		Name:      name,
	}
}

// Deleted событие удаления рабочего пространства
type Deleted struct {
	event.BaseEvent
}

// NewWorkspaceDeleted создает новое событие WorkspaceDeleted
func NewWorkspaceDeleted(workspaceID uuid.UUID, metadata event.Metadata) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeWorkspaceDeleted, workspaceID.String(), "Workspace", 1, metadata),
	}
}

// InviteCreated событие создания приглашения
type InviteCreated struct {
	event.BaseEvent

	WorkspaceID uuid.UUID
	Token       string
	CreatedBy   uuid.UUID
	ExpiresAt   time.Time
	MaxUses     int
}

// NewInviteCreated создает новое событие InviteCreated
func NewInviteCreated(
	inviteID, workspaceID uuid.UUID,
	token string,
	createdBy uuid.UUID,
	expiresAt time.Time,
	maxUses int,
	metadata event.Metadata,
) *InviteCreated {
	return &InviteCreated{
		BaseEvent:   event.NewBaseEvent(EventTypeInviteCreated, inviteID.String(), "Invite", 1, metadata),
		WorkspaceID: workspaceID,
		Token:       token,
		CreatedBy:   createdBy,
		ExpiresAt:   expiresAt,
		MaxUses:     maxUses,
	}
}

// InviteUsed событие использования приглашения
type InviteUsed struct {
	event.BaseEvent

	WorkspaceID uuid.UUID
	UsedBy      uuid.UUID
	UsedCount   int
}

// NewInviteUsed создает новое событие InviteUsed
func NewInviteUsed(inviteID, workspaceID, usedBy uuid.UUID, usedCount int, metadata event.Metadata) *InviteUsed {
	return &InviteUsed{
		BaseEvent:   event.NewBaseEvent(EventTypeInviteUsed, inviteID.String(), "Invite", 1, metadata),
		WorkspaceID: workspaceID,
		UsedBy:      usedBy,
		UsedCount:   usedCount,
	}
}

// InviteRevoked событие отмены приглашения
type InviteRevoked struct {
	event.BaseEvent

	WorkspaceID uuid.UUID
	RevokedBy   uuid.UUID
}

// NewInviteRevoked создает новое событие InviteRevoked
func NewInviteRevoked(inviteID, workspaceID, revokedBy uuid.UUID, metadata event.Metadata) *InviteRevoked {
	return &InviteRevoked{
		BaseEvent:   event.NewBaseEvent(EventTypeInviteRevoked, inviteID.String(), "Invite", 1, metadata),
		WorkspaceID: workspaceID,
		RevokedBy:   revokedBy,
	}
}
