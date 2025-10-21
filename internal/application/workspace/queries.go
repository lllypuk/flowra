package workspace

import "github.com/flowra/flowra/internal/domain/uuid"

// Query базовый интерфейс запросов
type Query interface {
	QueryName() string
}

// GetWorkspaceQuery - получение workspace по ID
type GetWorkspaceQuery struct {
	WorkspaceID uuid.UUID
}

func (q GetWorkspaceQuery) QueryName() string { return "GetWorkspace" }

// ListUserWorkspacesQuery - список workspace пользователя
type ListUserWorkspacesQuery struct {
	UserID uuid.UUID
	Offset int
	Limit  int
}

func (q ListUserWorkspacesQuery) QueryName() string { return "ListUserWorkspaces" }
