package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	userdomain "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestUserRepository создает тестовый репозиторий пользователей
func setupTestUserRepository(t *testing.T) *mongodb.MongoUserRepository {
	t.Helper()

	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("users")

	return mongodb.NewMongoUserRepository(coll)
}

// createTestUser создает тестового пользователя с уникальными данными
func createTestUser(t *testing.T, suffix string) *userdomain.User {
	t.Helper()

	user, err := userdomain.NewUser(
		"ext-id-"+suffix,
		"testuser_"+suffix,
		"test_"+suffix+"@example.com",
		"Test User "+suffix,
	)
	require.NoError(t, err)
	return user
}

// TestMongoUserRepository_Save_And_FindByID проверяет сохранение и поиск пользователя по ID
func TestMongoUserRepository_Save_And_FindByID(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "1")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Find by ID
	loaded, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all fields
	assert.Equal(t, user.ID(), loaded.ID())
	assert.Equal(t, user.ExternalID(), loaded.ExternalID())
	assert.Equal(t, user.Username(), loaded.Username())
	assert.Equal(t, user.Email(), loaded.Email())
	assert.Equal(t, user.DisplayName(), loaded.DisplayName())
	assert.Equal(t, user.IsSystemAdmin(), loaded.IsSystemAdmin())
}

// TestMongoUserRepository_FindByID_NotFound проверяет поиск несуществующего пользователя
func TestMongoUserRepository_FindByID_NotFound(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Try to find non-existent user
	_, err := repo.FindByID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoUserRepository_FindByEmail проверяет поиск пользователя по email
func TestMongoUserRepository_FindByEmail(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "email")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Find by email
	loaded, err := repo.FindByEmail(ctx, user.Email())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, user.ID(), loaded.ID())
	assert.Equal(t, user.Email(), loaded.Email())

	// Find by non-existent email
	_, err = repo.FindByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoUserRepository_FindByUsername проверяет поиск пользователя по username
func TestMongoUserRepository_FindByUsername(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "username")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Find by username
	loaded, err := repo.FindByUsername(ctx, user.Username())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, user.ID(), loaded.ID())
	assert.Equal(t, user.Username(), loaded.Username())

	// Find by non-existent username
	_, err = repo.FindByUsername(ctx, "nonexistent")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoUserRepository_FindByExternalID проверяет поиск пользователя по Keycloak ID
func TestMongoUserRepository_FindByExternalID(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "extid")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Find by external ID
	loaded, err := repo.FindByExternalID(ctx, user.ExternalID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, user.ID(), loaded.ID())
	assert.Equal(t, user.ExternalID(), loaded.ExternalID())

	// Find by non-existent external ID
	_, err = repo.FindByExternalID(ctx, "nonexistent-keycloak-id")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoUserRepository_List проверяет получение списка пользователей
func TestMongoUserRepository_List(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save multiple users
	for i := range 5 {
		user := createTestUser(t, string(rune('a'+i)))
		err := repo.Save(ctx, user)
		require.NoError(t, err)
	}

	// List all users
	users, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Len(t, users, 5)

	// List with pagination
	users, err = repo.List(ctx, 0, 2)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	// List with offset
	users, err = repo.List(ctx, 2, 10)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// List with offset beyond count
	users, err = repo.List(ctx, 10, 10)
	require.NoError(t, err)
	assert.Empty(t, users)
}

// TestMongoUserRepository_Count проверяет подсчет пользователей
func TestMongoUserRepository_Count(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Initial count should be 0
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add users and verify count
	for i := range 3 {
		user := createTestUser(t, string(rune('x'+i)))
		saveErr := repo.Save(ctx, user)
		require.NoError(t, saveErr)
	}

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestMongoUserRepository_Delete проверяет удаление пользователя
func TestMongoUserRepository_Delete(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "delete")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify user exists
	_, err = repo.FindByID(ctx, user.ID())
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(ctx, user.ID())
	require.NoError(t, err)

	// Verify user no longer exists
	_, err = repo.FindByID(ctx, user.ID())
	require.ErrorIs(t, err, errs.ErrNotFound)

	// Delete non-existent user should return error
	err = repo.Delete(ctx, uuid.NewUUID())
	require.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoUserRepository_Exists проверяет наличие пользователя по ID
func TestMongoUserRepository_Exists(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "exists")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Test Exists - should return true
	exists, err := repo.Exists(ctx, user.ID())
	require.NoError(t, err)
	assert.True(t, exists)

	// Test Exists - should return false for non-existent user
	exists, err = repo.Exists(ctx, uuid.NewUUID())
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestMongoUserRepository_ExistsByUsername проверяет наличие пользователя по username
func TestMongoUserRepository_ExistsByUsername(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "existsuser")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Test ExistsByUsername - should return true
	exists, err := repo.ExistsByUsername(ctx, user.Username())
	require.NoError(t, err)
	assert.True(t, exists)

	// Test ExistsByUsername - should return false
	exists, err = repo.ExistsByUsername(ctx, "nonexistent_user")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestMongoUserRepository_ExistsByEmail проверяет наличие пользователя по email
func TestMongoUserRepository_ExistsByEmail(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "existsemail")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Test ExistsByEmail - should return true
	exists, err := repo.ExistsByEmail(ctx, user.Email())
	require.NoError(t, err)
	assert.True(t, exists)

	// Test ExistsByEmail - should return false
	exists, err = repo.ExistsByEmail(ctx, "nonexistent@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestMongoUserRepository_InputValidation проверяет валидацию входных данных
func TestMongoUserRepository_InputValidation(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	t.Run("FindByID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByEmail with empty email", func(t *testing.T) {
		_, err := repo.FindByEmail(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByUsername with empty username", func(t *testing.T) {
		_, err := repo.FindByUsername(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByExternalID with empty externalID", func(t *testing.T) {
		_, err := repo.FindByExternalID(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Save with nil user", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Delete with zero UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Exists with zero UUID", func(t *testing.T) {
		_, err := repo.Exists(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("ExistsByUsername with empty username", func(t *testing.T) {
		_, err := repo.ExistsByUsername(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("ExistsByEmail with empty email", func(t *testing.T) {
		_, err := repo.ExistsByEmail(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

// TestMongoUserRepository_Update проверяет обновление пользователя
func TestMongoUserRepository_Update(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "update")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Update user profile
	newDisplayName := "Updated Display Name"
	err = user.UpdateProfile(&newDisplayName, nil)
	require.NoError(t, err)

	// Save updated user
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Load and verify
	loaded, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Display Name", loaded.DisplayName())
}

// TestMongoUserRepository_SetAdmin проверяет установку прав администратора
func TestMongoUserRepository_SetAdmin(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "admin")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify not admin initially
	loaded, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.False(t, loaded.IsSystemAdmin())

	// Set admin
	user.SetAdmin(true)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify admin flag
	loaded, err = repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.True(t, loaded.IsSystemAdmin())

	// Remove admin
	user.SetAdmin(false)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify admin flag removed
	loaded, err = repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.False(t, loaded.IsSystemAdmin())
}

// TestMongoUserRepository_DocumentToUser проверяет преобразование документа в User
func TestMongoUserRepository_DocumentToUser(t *testing.T) {
	repo := setupTestUserRepository(t)
	ctx := context.Background()

	// Create and save user
	user := createTestUser(t, "doctouser")
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Load user
	loaded, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)

	// Verify all fields are correctly restored
	assert.Equal(t, user.ID(), loaded.ID())
	assert.Equal(t, user.ExternalID(), loaded.ExternalID())
	assert.Equal(t, user.Username(), loaded.Username())
	assert.Equal(t, user.Email(), loaded.Email())
	assert.Equal(t, user.DisplayName(), loaded.DisplayName())
	assert.Equal(t, user.IsSystemAdmin(), loaded.IsSystemAdmin())
	// Times should be close (allow for millisecond precision loss due to MongoDB serialization)
	assert.WithinDuration(t, user.CreatedAt(), loaded.CreatedAt(), time.Millisecond)
	assert.WithinDuration(t, user.UpdatedAt(), loaded.UpdatedAt(), time.Millisecond)
}
