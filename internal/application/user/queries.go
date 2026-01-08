package user

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Query базовый interface запросов
type Query interface {
	QueryName() string
}

// GetUserQuery - retrieval user по ID
type GetUserQuery struct {
	UserID uuid.UUID
}

func (q GetUserQuery) QueryName() string { return "GetUser" }

// GetUserByUsernameQuery - search по username
type GetUserByUsernameQuery struct {
	Username string
}

func (q GetUserByUsernameQuery) QueryName() string { return "GetUserByUsername" }

// ListUsersQuery - list users с пагинацией
type ListUsersQuery struct {
	Offset int
	Limit  int
}

func (q ListUsersQuery) QueryName() string { return "ListUsers" }
