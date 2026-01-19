package mocks

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MockUserRepository implements appcore.UserRepository for testing
type MockUserRepository struct {
	users map[uuid.UUID]*appcore.User
}

// NewMockUserRepository creates a new mock repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[uuid.UUID]*appcore.User),
	}
}

// AddUser adds a user to the mock repository
func (m *MockUserRepository) AddUser(id uuid.UUID, username, fullName string) {
	m.users[id] = &appcore.User{
		ID:       id,
		Username: username,
		FullName: fullName,
	}
}

// Exists checks if a user exists
func (m *MockUserRepository) Exists(_ context.Context, userID uuid.UUID) (bool, error) {
	_, exists := m.users[userID]
	return exists, nil
}

// GetByID finds a user by ID
func (m *MockUserRepository) GetByID(_ context.Context, userID uuid.UUID) (*appcore.User, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, errs.ErrNotFound
	}
	return user, nil
}

// GetByUsername finds a user by username
func (m *MockUserRepository) GetByUsername(_ context.Context, username string) (*appcore.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errs.ErrNotFound
}

// Reset clears all users (for tests)
func (m *MockUserRepository) Reset() {
	m.users = make(map[uuid.UUID]*appcore.User)
}

// GetAllUsers returns all users (for tests)
func (m *MockUserRepository) GetAllUsers() []*appcore.User {
	users := make([]*appcore.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users
}
