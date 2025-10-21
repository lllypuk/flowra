package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Command базовый интерфейс команд
type Command interface {
	CommandName() string
}

// CreateWorkspaceCommand - создание workspace
type CreateWorkspaceCommand struct {
	Name      string
	CreatedBy uuid.UUID
}

func (c CreateWorkspaceCommand) CommandName() string { return "CreateWorkspace" }

// UpdateWorkspaceCommand - обновление workspace
type UpdateWorkspaceCommand struct {
	WorkspaceID uuid.UUID
	Name        string
	UpdatedBy   uuid.UUID
}

func (c UpdateWorkspaceCommand) CommandName() string { return "UpdateWorkspace" }

// CreateInviteCommand - создание инвайта
type CreateInviteCommand struct {
	WorkspaceID uuid.UUID
	ExpiresAt   *time.Time // опционально, default: 7 дней
	MaxUses     *int       // опционально, default: 0 (unlimited)
	CreatedBy   uuid.UUID
}

func (c CreateInviteCommand) CommandName() string { return "CreateInvite" }

// AcceptInviteCommand - принятие инвайта
type AcceptInviteCommand struct {
	Token  string
	UserID uuid.UUID
}

func (c AcceptInviteCommand) CommandName() string { return "AcceptInvite" }

// RevokeInviteCommand - отзыв инвайта
type RevokeInviteCommand struct {
	InviteID  uuid.UUID
	RevokedBy uuid.UUID
}

func (c RevokeInviteCommand) CommandName() string { return "RevokeInvite" }
