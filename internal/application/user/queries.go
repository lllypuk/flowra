package user

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Query bazovyy interface zaprosov
type Query interface {
	QueryName() string
}

// GetUserQuery - retrieval user po ID
type GetUserQuery struct {
	UserID uuid.UUID
}

func (q GetUserQuery) QueryName() string { return "GetUser" }

// GetUserByUsernameQuery - search po username
type GetUserByUsernameQuery struct {
	Username string
}

func (q GetUserByUsernameQuery) QueryName() string { return "GetUserByUsername" }

// ListUsersQuery - list users s paginatsiey
type ListUsersQuery struct {
	Offset int
	Limit  int
}

func (q ListUsersQuery) QueryName() string { return "ListUsers" }
