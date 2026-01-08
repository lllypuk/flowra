package workspace

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Query базовый interface запросов
type Query interface {
	QueryName() string
}

// GetWorkspaceQuery - retrieval workspace по ID
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
