package user

import "github.com/flowra/flowra/internal/domain/uuid"

// Query базовый интерфейс запросов
type Query interface {
	QueryName() string
}

// GetUserQuery - получение пользователя по ID
type GetUserQuery struct {
	UserID uuid.UUID
}

func (q GetUserQuery) QueryName() string { return "GetUser" }

// GetUserByUsernameQuery - поиск по username
type GetUserByUsernameQuery struct {
	Username string
}

func (q GetUserByUsernameQuery) QueryName() string { return "GetUserByUsername" }

// ListUsersQuery - список пользователей с пагинацией
type ListUsersQuery struct {
	Offset int
	Limit  int
}

func (q ListUsersQuery) QueryName() string { return "ListUsers" }
