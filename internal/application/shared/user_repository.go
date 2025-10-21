package shared

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// UserRepository предоставляет доступ к информации о пользователях
type UserRepository interface {
	// Exists проверяет, существует ли пользователь с заданным ID
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)

	// GetByUsername ищет пользователя по username (для будущего парсинга @mentions)
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// User представляет минимальную информацию о пользователе
type User struct {
	ID       uuid.UUID
	Username string
	FullName string
}
