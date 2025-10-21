package user_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowra/flowra/internal/domain/errs"
	userDomain "github.com/flowra/flowra/internal/domain/user"
	"github.com/flowra/flowra/internal/domain/uuid"
)

func TestNewUser_Success(t *testing.T) {
	// Arrange
	keycloakID := "external-123"
	username := "john_doe"
	email := "john@example.com"
	displayName := "John Doe"

	// Act
	user, err := userDomain.NewUser(keycloakID, username, email, displayName)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.False(t, user.ID().IsZero())
	assert.Equal(t, keycloakID, user.ExternalID())
	assert.Equal(t, username, user.Username())
	assert.Equal(t, email, user.Email())
	assert.Equal(t, displayName, user.DisplayName())
	assert.False(t, user.IsSystemAdmin())
	assert.False(t, user.CreatedAt().IsZero())
	assert.False(t, user.UpdatedAt().IsZero())
	assert.WithinDuration(t, time.Now(), user.CreatedAt(), time.Second)
	assert.WithinDuration(t, time.Now(), user.UpdatedAt(), time.Second)
}

func TestNewUser_EmptyExternalID(t *testing.T) {
	// Act
	user, err := userDomain.NewUser("", "username", "email@example.com", "Display")

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, errs.ErrInvalidInput)
	assert.Nil(t, user)
}

func TestNewUser_EmptyUsername(t *testing.T) {
	// Act
	user, err := userDomain.NewUser("external-123", "", "email@example.com", "Display")

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, errs.ErrInvalidInput)
	assert.Nil(t, user)
}

func TestNewUser_EmptyEmail(t *testing.T) {
	// Act
	user, err := userDomain.NewUser("external-123", "username", "", "Display")

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, errs.ErrInvalidInput)
	assert.Nil(t, user)
}

func TestNewUser_EmptyDisplayName_Allowed(t *testing.T) {
	// Arrange - display name может быть пустым
	keycloakID := "external-123"
	username := "john_doe"
	email := "john@example.com"

	// Act
	user, err := userDomain.NewUser(keycloakID, username, email, "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Empty(t, user.DisplayName())
}

func TestReconstruct(t *testing.T) {
	// Arrange
	id := uuid.NewUUID()
	keycloakID := "external-jane"
	username := "jane_doe"
	email := "jane@example.com"
	displayName := "Jane Doe"
	isSystemAdmin := true
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now().Add(-1 * time.Hour)

	// Act
	user := userDomain.Reconstruct(id, keycloakID, username, email, displayName, isSystemAdmin, createdAt, updatedAt)

	// Assert
	assert.NotNil(t, user)
	assert.Equal(t, id, user.ID())
	assert.Equal(t, keycloakID, user.ExternalID())
	assert.Equal(t, username, user.Username())
	assert.Equal(t, email, user.Email())
	assert.Equal(t, displayName, user.DisplayName())
	assert.True(t, user.IsSystemAdmin())
	assert.Equal(t, createdAt, user.CreatedAt())
	assert.Equal(t, updatedAt, user.UpdatedAt())
}

func TestUser_UpdateProfile_Success(t *testing.T) {
	// Arrange
	user, _ := userDomain.NewUser("external-john", "john", "john@example.com", "John")
	oldUpdatedAt := user.UpdatedAt()
	newDisplayName := "John Smith"

	// Небольшая задержка чтобы UpdatedAt изменился
	time.Sleep(10 * time.Millisecond)

	// Act
	err := user.UpdateProfile(&newDisplayName, nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, user.DisplayName())
	assert.True(t, user.UpdatedAt().After(oldUpdatedAt))
}

func TestUser_UpdateProfile_NothingToUpdate(t *testing.T) {
	// Arrange
	user, _ := userDomain.NewUser("external-john", "john", "john@example.com", "John")
	oldDisplayName := user.DisplayName()
	oldUpdatedAt := user.UpdatedAt()

	// Act - ничего не передаем
	err := user.UpdateProfile(nil, nil)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, errs.ErrInvalidInput)
	assert.Equal(t, oldDisplayName, user.DisplayName(), "DisplayName should not change")
	assert.Equal(t, oldUpdatedAt, user.UpdatedAt(), "UpdatedAt should not change")
}

func TestUser_SetAdmin_GrantRights(t *testing.T) {
	// Arrange
	user, _ := userDomain.NewUser("external-john", "john", "john@example.com", "John")
	assert.False(t, user.IsSystemAdmin())
	oldUpdatedAt := user.UpdatedAt()

	// Небольшая задержка
	time.Sleep(10 * time.Millisecond)

	// Act
	user.SetAdmin(true)

	// Assert
	assert.True(t, user.IsSystemAdmin())
	assert.True(t, user.UpdatedAt().After(oldUpdatedAt))
}

func TestUser_SetAdmin_RevokeRights(t *testing.T) {
	// Arrange
	id := uuid.NewUUID()
	user := userDomain.Reconstruct(
		id,
		"external-admin",
		"admin",
		"admin@example.com",
		"Admin",
		true,
		time.Now(),
		time.Now(),
	)
	assert.True(t, user.IsSystemAdmin())
	oldUpdatedAt := user.UpdatedAt()

	// Небольшая задержка
	time.Sleep(10 * time.Millisecond)

	// Act
	user.SetAdmin(false)

	// Assert
	assert.False(t, user.IsSystemAdmin())
	assert.True(t, user.UpdatedAt().After(oldUpdatedAt))
}

func TestUser_AllGetters(t *testing.T) {
	// Arrange
	id := uuid.NewUUID()
	keycloakID := "external-test"
	username := "testuser"
	email := "test@example.com"
	displayName := "Test User"
	isAdmin := true
	createdAt := time.Now().Add(-48 * time.Hour)
	updatedAt := time.Now().Add(-24 * time.Hour)

	user := userDomain.Reconstruct(id, keycloakID, username, email, displayName, isAdmin, createdAt, updatedAt)

	// Act & Assert
	t.Run("ID", func(t *testing.T) {
		assert.Equal(t, id, user.ID())
	})

	t.Run("ExternalID", func(t *testing.T) {
		assert.Equal(t, keycloakID, user.ExternalID())
	})

	t.Run("Username", func(t *testing.T) {
		assert.Equal(t, username, user.Username())
	})

	t.Run("Email", func(t *testing.T) {
		assert.Equal(t, email, user.Email())
	})

	t.Run("DisplayName", func(t *testing.T) {
		assert.Equal(t, displayName, user.DisplayName())
	})

	t.Run("IsSystemAdmin", func(t *testing.T) {
		assert.Equal(t, isAdmin, user.IsSystemAdmin())
	})

	t.Run("CreatedAt", func(t *testing.T) {
		assert.Equal(t, createdAt, user.CreatedAt())
	})

	t.Run("UpdatedAt", func(t *testing.T) {
		assert.Equal(t, updatedAt, user.UpdatedAt())
	})
}
