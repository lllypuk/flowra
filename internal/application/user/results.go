package user

import (
	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/user"
)

// Result - результат операции с одним пользователем
type Result struct {
	shared.Result[*user.User]
}

// UsersListResult - результат операции со списком пользователей
type UsersListResult struct {
	Users      []*user.User
	TotalCount int
	Offset     int
	Limit      int
}
