package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// CommandRepository defines interface for commands (change state) workspace prostranstv
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves workspace space (creation or update)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete udalyaet workspace space
	Delete(ctx context.Context, id uuid.UUID) error

	// AddMember adds chlena in workspace
	AddMember(ctx context.Context, member *workspace.Member) error

	// RemoveMember udalyaet chlena from workspace
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMember obnovlyaet data chlena workspace
	UpdateMember(ctx context.Context, member *workspace.Member) error
}

// QueryRepository defines interface for zaprosov (only reading) workspace prostranstv
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds workspace space po ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// FindByKeycloakGroup finds workspace space po ID groups Keycloak
	FindByKeycloakGroup(ctx context.Context, keycloakGroupID string) (*workspace.Workspace, error)

	// List returns list workspace prostranstv s paginatsiey
	List(ctx context.Context, offset, limit int) ([]*workspace.Workspace, error)

	// Count returns obschee count workspace prostranstv
	Count(ctx context.Context) (int, error)

	// FindInviteByToken finds priglashenie po tokenu
	FindInviteByToken(ctx context.Context, token string) (*workspace.Invite, error)

	// GetMember returns chlena workspace po userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// IsMember checks, is li user chlenom workspace
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)

	// ListWorkspacesByUser returns workspaces, in kotoryh user is chlenom
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

	// CountWorkspacesByUser returns count workspaces user
	CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)

	// ListMembers returns all chlenov workspace
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)

	// CountMembers returns count chlenov workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of operatsiy
type Repository interface {
	CommandRepository
	QueryRepository
}
