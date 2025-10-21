package mocks

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockUserRepository_Exists(t *testing.T) {
	repo := NewMockUserRepository()
	userID := uuid.NewUUID()

	// User doesn't exist initially
	exists, err := repo.Exists(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Add user
	repo.AddUser(userID, "testuser", "Test User")

	// User now exists
	exists, err = repo.Exists(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMockUserRepository_GetByUsername(t *testing.T) {
	repo := NewMockUserRepository()
	userID := uuid.NewUUID()

	// User doesn't exist initially
	user, err := repo.GetByUsername(context.Background(), "testuser")
	require.ErrorIs(t, err, errs.ErrNotFound)
	assert.Nil(t, user)

	// Add user
	repo.AddUser(userID, "testuser", "Test User")

	// User can be found by username
	user, err = repo.GetByUsername(context.Background(), "testuser")
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "Test User", user.FullName)
}

func TestMockUserRepository_Reset(t *testing.T) {
	repo := NewMockUserRepository()
	userID := uuid.NewUUID()

	// Add user
	repo.AddUser(userID, "testuser", "Test User")
	exists, _ := repo.Exists(context.Background(), userID)
	assert.True(t, exists)

	// Reset
	repo.Reset()

	// User no longer exists
	exists, err := repo.Exists(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMockUserRepository_GetAllUsers(t *testing.T) {
	repo := NewMockUserRepository()

	// Initially empty
	users := repo.GetAllUsers()
	assert.Empty(t, users)

	// Add multiple users
	userID1 := uuid.NewUUID()
	userID2 := uuid.NewUUID()
	repo.AddUser(userID1, "user1", "User One")
	repo.AddUser(userID2, "user2", "User Two")

	// Get all users
	users = repo.GetAllUsers()
	assert.Len(t, users, 2)
}
