package workspace

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Query bazovyy interface zaprosov
type Query interface {
	QueryName() string
}

// GetWorkspaceQuery - retrieval workspace po ID
type GetWorkspaceQuery struct {
	WorkspaceID uuid.UUID
}

func (q GetWorkspaceQuery) QueryName() string { return "GetWorkspace" }

// ListUserWorkspacesQuery - list workspace user
type ListUserWorkspacesQuery struct {
	UserID uuid.UUID
	Offset int
	Limit  int
}

func (q ListUserWorkspacesQuery) QueryName() string { return "ListUserWorkspaces" }
