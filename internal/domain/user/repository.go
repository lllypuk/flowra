package user

import (
	"context"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Repository определяет интерфейс репозитория пользователей
type Repository interface {
	// FindByID находит пользователя по ID
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByKeycloakID находит пользователя по Keycloak ID
	FindByKeycloakID(ctx context.Context, keycloakID string) (*User, error)

	// FindByEmail находит пользователя по email
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByUsername находит пользователя по username
	FindByUsername(ctx context.Context, username string) (*User, error)

	// Save сохраняет пользователя
	Save(ctx context.Context, user *User) error

	// Delete удаляет пользователя
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список пользователей с пагинацией
	List(ctx context.Context, offset, limit int) ([]*User, error)

	// Count возвращает общее количество пользователей
	Count(ctx context.Context) (int, error)
}
