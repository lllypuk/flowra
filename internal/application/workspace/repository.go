package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// CommandRepository defines interface for commands (change state) workspaceих пространств
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves workspaceее пространство (creation or update)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete удаляет workspaceее пространство
	Delete(ctx context.Context, id uuid.UUID) error

	// AddMember добавляет члена in workspace
	AddMember(ctx context.Context, member *workspace.Member) error

	// RemoveMember удаляет члена from workspace
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMember обновляет data члена workspace
	UpdateMember(ctx context.Context, member *workspace.Member) error
}

// QueryRepository defines interface for запросов (only reading) workspaceих пространств
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds workspaceее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// FindByKeycloakGroup finds workspaceее пространство по ID groupsы Keycloak
	FindByKeycloakGroup(ctx context.Context, keycloakGroupID string) (*workspace.Workspace, error)

	// List returns list workspaceих пространств с пагинацией
	List(ctx context.Context, offset, limit int) ([]*workspace.Workspace, error)

	// Count returns общее count workspaceих пространств
	Count(ctx context.Context) (int, error)

	// FindInviteByToken finds приглашение по токену
	FindInviteByToken(ctx context.Context, token string) (*workspace.Invite, error)

	// GetMember returns члена workspace по userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// IsMember checks, is ли userель членом workspace
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)

	// ListWorkspacesByUser returns workspaces, in которых userель is членом
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

	// CountWorkspacesByUser returns count workspaces user
	CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)

	// ListMembers returns all членов workspace
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)

	// CountMembers returns count членов workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of операций
type Repository interface {
	CommandRepository
	QueryRepository
}
