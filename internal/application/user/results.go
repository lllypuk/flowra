package user

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// Result - result operatsii s odnim user
type Result struct {
	appcore.Result[*user.User]
}

// UsersListResult - result operatsii with spiskom users
type UsersListResult struct {
	Users      []*user.User
	TotalCount int
	Offset     int
	Limit      int
}
