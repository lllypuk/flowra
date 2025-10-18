package mocks

import (
	"context"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/lllypuk/teams-up/internal/usecase/shared"
)

// MockUserRepository реализует shared.UserRepository для тестирования
type MockUserRepository struct {
	users map[uuid.UUID]*shared.User
}

// NewMockUserRepository создает новый mock репозиторий
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[uuid.UUID]*shared.User),
	}
}

// AddUser добавляет пользователя в mock репозиторий
func (m *MockUserRepository) AddUser(id uuid.UUID, username, fullName string) {
	m.users[id] = &shared.User{
		ID:       id,
		Username: username,
		FullName: fullName,
	}
}

// Exists проверяет, существует ли пользователь
func (m *MockUserRepository) Exists(_ context.Context, userID uuid.UUID) (bool, error) {
	_, exists := m.users[userID]
	return exists, nil
}

// GetByUsername ищет пользователя по username
func (m *MockUserRepository) GetByUsername(_ context.Context, username string) (*shared.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

// Reset очищает все пользователей (для тестов)
func (m *MockUserRepository) Reset() {
	m.users = make(map[uuid.UUID]*shared.User)
}

// GetAllUsers возвращает всех пользователей (для тестов)
func (m *MockUserRepository) GetAllUsers() []*shared.User {
	users := make([]*shared.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users
}
