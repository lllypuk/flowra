package user

import (
	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/user"
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
