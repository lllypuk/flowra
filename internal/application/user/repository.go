package user

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository defines interface for commands (change state) users
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves user (creation or update)
	Save(ctx context.Context, u *user.User) error

	// Delete удаляет user
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryRepository defines interface for запросов (only reading) users
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds user по ID
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)

	// FindByExternalID finds user по ID from внешней системы аутентификации
	FindByExternalID(ctx context.Context, externalID string) (*user.User, error)

	// FindByEmail finds user по email
	FindByEmail(ctx context.Context, email string) (*user.User, error)

	// FindByUsername finds user по username
	FindByUsername(ctx context.Context, username string) (*user.User, error)

	// List returns list users с пагинацией
	List(ctx context.Context, offset, limit int) ([]*user.User, error)

	// Count returns общее count users
	Count(ctx context.Context) (int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of операций
type Repository interface {
	CommandRepository
	QueryRepository
}
