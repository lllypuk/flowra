package workspace

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// Result - результат операции с workspace
type Result struct {
	appcore.Result[*workspace.Workspace]
}

// InviteResult - результат операции с invite
type InviteResult struct {
	appcore.Result[*workspace.Invite]
}

// ListResult - результат операции со списком workspace
type ListResult struct {
	Workspaces []*workspace.Workspace
	TotalCount int
	Offset     int
	Limit      int
}
