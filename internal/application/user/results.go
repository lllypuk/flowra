package user

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// Result - result операции с одним userелем
type Result struct {
	appcore.Result[*user.User]
}

// UsersListResult - result операции with списком users
type UsersListResult struct {
	Users      []*user.User
	TotalCount int
	Offset     int
	Limit      int
}
