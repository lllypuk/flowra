package user

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository определяет интерфейс для команд (изменение состояния) пользователей
// Интерфейс объявлен на стороне потребителя (application layer)
type CommandRepository interface {
	// Save сохраняет пользователя (создание или обновление)
	Save(ctx context.Context, u *user.User) error

	// Delete удаляет пользователя
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryRepository определяет интерфейс для запросов (только чтение) пользователей
// Интерфейс объявлен на стороне потребителя (application layer)
type QueryRepository interface {
	// FindByID находит пользователя по ID
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)

	// FindByExternalID находит пользователя по ID из внешней системы аутентификации
	FindByExternalID(ctx context.Context, externalID string) (*user.User, error)

	// FindByEmail находит пользователя по email
	FindByEmail(ctx context.Context, email string) (*user.User, error)

	// FindByUsername находит пользователя по username
	FindByUsername(ctx context.Context, username string) (*user.User, error)

	// List возвращает список пользователей с пагинацией
	List(ctx context.Context, offset, limit int) ([]*user.User, error)

	// Count возвращает общее количество пользователей
	Count(ctx context.Context) (int, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
