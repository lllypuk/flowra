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

	// Delete udalyaet user
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryRepository defines interface for zaprosov (only reading) users
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds user po ID
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)

	// FindByExternalID finds user po ID from vneshney sistemy autentifikatsii
	FindByExternalID(ctx context.Context, externalID string) (*user.User, error)

	// FindByEmail finds user po email
	FindByEmail(ctx context.Context, email string) (*user.User, error)

	// FindByUsername finds user po username
	FindByUsername(ctx context.Context, username string) (*user.User, error)

	// List returns list users s paginatsiey
	List(ctx context.Context, offset, limit int) ([]*user.User, error)

	// Count returns obschee count users
	Count(ctx context.Context) (int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of operatsiy
type Repository interface {
	CommandRepository
	QueryRepository
}
