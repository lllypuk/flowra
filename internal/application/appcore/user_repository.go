package appcore

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// UserRepository provides access to user information
type UserRepository interface {
	// Exists checks if a user with the given ID exists
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)

	// GetByID finds a user by ID
	GetByID(ctx context.Context, userID uuid.UUID) (*User, error)

	// GetByUsername finds a user by username (for future @mentions parsing)
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// User represents minimal user information
type User struct {
	ID       uuid.UUID
	Username string
	FullName string
}
