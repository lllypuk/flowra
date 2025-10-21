package workspace

import (
	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// Result - результат операции с workspace
type Result struct {
	shared.Result[*workspace.Workspace]
}

// InviteResult - результат операции с invite
type InviteResult struct {
	shared.Result[*workspace.Invite]
}

// ListResult - результат операции со списком workspace
type ListResult struct {
	Workspaces []*workspace.Workspace
	TotalCount int
	Offset     int
	Limit      int
}
