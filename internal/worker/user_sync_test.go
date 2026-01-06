package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKeycloakUserClient is a mock implementation of KeycloakUserClient.
type MockKeycloakUserClient struct {
	users      []keycloak.User
	countErr   error
	listErr    error
	listCalls  atomic.Int32
	countCalls atomic.Int32
}

func NewMockKeycloakUserClient(users []keycloak.User) *MockKeycloakUserClient {
	return &MockKeycloakUserClient{
		users: users,
	}
}

func (m *MockKeycloakUserClient) ListUsers(_ context.Context, first, limit int) ([]keycloak.User, error) {
	m.listCalls.Add(1)
	if m.listErr != nil {
		return nil, m.listErr
	}

	if first >= len(m.users) {
		return []keycloak.User{}, nil
	}

	end := first + limit
	if end > len(m.users) {
		end = len(m.users)
	}

	return m.users[first:end], nil
}

func (m *MockKeycloakUserClient) CountUsers(_ context.Context) (int, error) {
	m.countCalls.Add(1)
	if m.countErr != nil {
		return 0, m.countErr
	}
	return len(m.users), nil
}

func (m *MockKeycloakUserClient) SetCountError(err error) {
	m.countErr = err
}

func (m *MockKeycloakUserClient) SetListError(err error) {
	m.listErr = err
}

// MockSyncUserRepository is a mock implementation of SyncUserRepository.
type MockSyncUserRepository struct {
	mu    sync.RWMutex
	users map[string]*user.User // keyed by external ID

	findErr error
	saveErr error
	listErr error

	saveCalls atomic.Int32
	findCalls atomic.Int32
}

func NewMockSyncUserRepository() *MockSyncUserRepository {
	return &MockSyncUserRepository{
		users: make(map[string]*user.User),
	}
}

func (m *MockSyncUserRepository) FindByExternalID(_ context.Context, externalID string) (*user.User, error) {
	m.findCalls.Add(1)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.findErr != nil {
		return nil, m.findErr
	}

	u, exists := m.users[externalID]
	if !exists {
		return nil, errs.ErrNotFound
	}
	return u, nil
}

func (m *MockSyncUserRepository) Save(_ context.Context, u *user.User) error {
	m.saveCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveErr != nil {
		return m.saveErr
	}

	m.users[u.ExternalID()] = u
	return nil
}

func (m *MockSyncUserRepository) ListExternalIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listErr != nil {
		return nil, m.listErr
	}

	ids := make([]string, 0, len(m.users))
	for id := range m.users {
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *MockSyncUserRepository) AddUser(u *user.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ExternalID()] = u
}

func (m *MockSyncUserRepository) GetUser(externalID string) *user.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[externalID]
}

func (m *MockSyncUserRepository) UserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

func (m *MockSyncUserRepository) SetFindError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.findErr = err
}

func (m *MockSyncUserRepository) SetSaveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveErr = err
}

func (m *MockSyncUserRepository) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listErr = err
}

func TestNewUserSyncWorker(t *testing.T) {
	t.Run("creates worker with provided config", func(t *testing.T) {
		kcClient := NewMockKeycloakUserClient(nil)
		repo := NewMockSyncUserRepository()
		logger := slog.Default()
		config := worker.UserSyncConfig{
			Interval:  5 * time.Minute,
			BatchSize: 50,
			Enabled:   true,
		}

		w := worker.NewUserSyncWorker(kcClient, repo, logger, config)

		require.NotNil(t, w)
	})

	t.Run("creates worker with nil logger", func(t *testing.T) {
		kcClient := NewMockKeycloakUserClient(nil)
		repo := NewMockSyncUserRepository()
		config := worker.DefaultUserSyncConfig()

		w := worker.NewUserSyncWorker(kcClient, repo, nil, config)

		require.NotNil(t, w)
	})
}

func TestDefaultUserSyncConfig(t *testing.T) {
	config := worker.DefaultUserSyncConfig()

	assert.Equal(t, 15*time.Minute, config.Interval)
	assert.Equal(t, 100, config.BatchSize)
	assert.True(t, config.Enabled)
}

func TestUserSyncWorker_Sync_CreatesNewUsers(t *testing.T) {
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "alice",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Smith",
			Enabled:   true,
		},
		{
			ID:        "kc-user-2",
			Username:  "bob",
			Email:     "bob@example.com",
			FirstName: "Bob",
			LastName:  "Jones",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, repo.UserCount())

	alice := repo.GetUser("kc-user-1")
	require.NotNil(t, alice)
	assert.Equal(t, "alice", alice.Username())
	assert.Equal(t, "alice@example.com", alice.Email())
	assert.Equal(t, "Alice Smith", alice.DisplayName())
	assert.True(t, alice.IsActive())

	bob := repo.GetUser("kc-user-2")
	require.NotNil(t, bob)
	assert.Equal(t, "bob", bob.Username())
}

func TestUserSyncWorker_Sync_UpdatesExistingUsers(t *testing.T) {
	// Create existing user with old data
	existingUser, err := user.NewUser("kc-user-1", "old_alice", "old@example.com", "Old Name")
	require.NoError(t, err)

	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "new_alice",
			Email:     "new@example.com",
			FirstName: "New",
			LastName:  "Name",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	repo.AddUser(existingUser)

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err = w.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, repo.UserCount())

	updated := repo.GetUser("kc-user-1")
	require.NotNil(t, updated)
	assert.Equal(t, "new_alice", updated.Username())
	assert.Equal(t, "new@example.com", updated.Email())
	assert.Equal(t, "New Name", updated.DisplayName())
}

func TestUserSyncWorker_Sync_DeactivatesMissingUsers(t *testing.T) {
	// Create user that exists locally but not in Keycloak
	localUser, err := user.NewUser("kc-user-deleted", "deleted_user", "deleted@example.com", "Deleted User")
	require.NoError(t, err)
	assert.True(t, localUser.IsActive())

	// Keycloak returns different user
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "alice",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Smith",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	repo.AddUser(localUser)

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err = w.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, repo.UserCount()) // Original + new user

	// Check deleted user is deactivated
	deactivated := repo.GetUser("kc-user-deleted")
	require.NotNil(t, deactivated)
	assert.False(t, deactivated.IsActive())

	// Check new user is created
	alice := repo.GetUser("kc-user-1")
	require.NotNil(t, alice)
	assert.True(t, alice.IsActive())
}

func TestUserSyncWorker_Sync_HandlesDisabledKeycloakUsers(t *testing.T) {
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "disabled_user",
			Email:     "disabled@example.com",
			FirstName: "Disabled",
			LastName:  "User",
			Enabled:   false, // User is disabled in Keycloak
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.NoError(t, err)

	created := repo.GetUser("kc-user-1")
	require.NotNil(t, created)
	assert.False(t, created.IsActive()) // Should reflect Keycloak enabled status
}

func TestUserSyncWorker_Sync_NoChangesNeeded(t *testing.T) {
	// Create user with same data as Keycloak
	existingUser, err := user.NewUser("kc-user-1", "alice", "alice@example.com", "Alice Smith")
	require.NoError(t, err)

	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "alice",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Smith",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	repo.AddUser(existingUser)

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	savesBeforeSync := repo.saveCalls.Load()

	err = w.Sync(context.Background())

	require.NoError(t, err)
	// Save should not be called when no changes
	assert.Equal(t, savesBeforeSync, repo.saveCalls.Load())
}

func TestUserSyncWorker_Sync_BatchProcessing(t *testing.T) {
	// Create 25 users to test batching
	var kcUsers []keycloak.User
	for i := range 25 {
		kcUsers = append(kcUsers, keycloak.User{
			ID:        "kc-user-" + string(rune('a'+i)),
			Username:  "user" + string(rune('a'+i)),
			Email:     "user" + string(rune('a'+i)) + "@example.com",
			FirstName: "User",
			LastName:  string(rune('A' + i)),
			Enabled:   true,
		})
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 10, // Small batch size to test multiple batches
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 25, repo.UserCount())
	// Should have made 3 list calls (10 + 10 + 5)
	assert.Equal(t, int32(3), kcClient.listCalls.Load())
}

func TestUserSyncWorker_Sync_CountUsersError(t *testing.T) {
	kcClient := NewMockKeycloakUserClient(nil)
	kcClient.SetCountError(errors.New("keycloak unavailable"))

	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count keycloak users")
}

func TestUserSyncWorker_Sync_ListUsersError(t *testing.T) {
	kcClient := NewMockKeycloakUserClient([]keycloak.User{{ID: "test"}})
	kcClient.SetListError(errors.New("list failed"))

	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list keycloak users")
}

func TestUserSyncWorker_Sync_ContextCancellation(t *testing.T) {
	var kcUsers []keycloak.User
	for i := range 100 {
		kcUsers = append(kcUsers, keycloak.User{
			ID:       "kc-user-" + string(rune(i)),
			Username: "user" + string(rune(i)),
			Email:    "user@example.com",
			Enabled:  true,
		})
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 10,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := w.Sync(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestUserSyncWorker_Sync_SaveError(t *testing.T) {
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "alice",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Smith",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	repo.SetSaveError(errors.New("database error"))

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	// Should not return error, but log it
	err := w.Sync(context.Background())

	require.NoError(t, err) // Sync continues despite individual errors
	assert.Equal(t, 0, repo.UserCount())
}

func TestUserSyncWorker_Sync_EmptyKeycloakUsers(t *testing.T) {
	kcClient := NewMockKeycloakUserClient(nil)
	repo := NewMockSyncUserRepository()

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, repo.UserCount())
}

func TestUserSyncWorker_Run_DisabledWorker(t *testing.T) {
	kcClient := NewMockKeycloakUserClient(nil)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Millisecond,
		BatchSize: 100,
		Enabled:   false, // Disabled
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)

	// Should return nil immediately when disabled
	require.NoError(t, err)
	// CountUsers should not be called
	assert.Equal(t, int32(0), kcClient.countCalls.Load())
}

func TestUserSyncWorker_Run_StopsOnContextCancel(t *testing.T) {
	kcClient := NewMockKeycloakUserClient(nil)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour, // Long interval so we can cancel
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx)
	}()

	// Let initial sync complete
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}

	// Should have run at least one sync
	assert.GreaterOrEqual(t, kcClient.countCalls.Load(), int32(1))
}

func TestUserSyncWorker_SyncSingleUser(t *testing.T) {
	t.Run("creates new user", func(t *testing.T) {
		kcClient := NewMockKeycloakUserClient(nil)
		repo := NewMockSyncUserRepository()
		config := worker.DefaultUserSyncConfig()

		w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

		kcUser := keycloak.User{
			ID:        "kc-single-user",
			Username:  "single",
			Email:     "single@example.com",
			FirstName: "Single",
			LastName:  "User",
			Enabled:   true,
		}

		err := w.SyncSingleUser(context.Background(), kcUser)

		require.NoError(t, err)
		assert.Equal(t, 1, repo.UserCount())

		created := repo.GetUser("kc-single-user")
		require.NotNil(t, created)
		assert.Equal(t, "single", created.Username())
	})

	t.Run("updates existing user", func(t *testing.T) {
		kcClient := NewMockKeycloakUserClient(nil)
		repo := NewMockSyncUserRepository()

		existing, _ := user.NewUser("kc-single-user", "old_username", "old@example.com", "Old Name")
		repo.AddUser(existing)

		config := worker.DefaultUserSyncConfig()
		w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

		kcUser := keycloak.User{
			ID:        "kc-single-user",
			Username:  "new_username",
			Email:     "new@example.com",
			FirstName: "New",
			LastName:  "Name",
			Enabled:   true,
		}

		err := w.SyncSingleUser(context.Background(), kcUser)

		require.NoError(t, err)

		updated := repo.GetUser("kc-single-user")
		require.NotNil(t, updated)
		assert.Equal(t, "new_username", updated.Username())
		assert.Equal(t, "new@example.com", updated.Email())
	})
}

func TestUserSyncWorker_Sync_DisplayNameFallback(t *testing.T) {
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-no-name",
			Username:  "noname",
			Email:     "noname@example.com",
			FirstName: "",
			LastName:  "",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	err := w.Sync(context.Background())

	require.NoError(t, err)

	created := repo.GetUser("kc-user-no-name")
	require.NotNil(t, created)
	// When FirstName and LastName are empty, should fall back to username
	assert.Equal(t, "noname", created.DisplayName())
}

func TestUserSyncWorker_Sync_ListExternalIDsError(t *testing.T) {
	kcUsers := []keycloak.User{
		{
			ID:        "kc-user-1",
			Username:  "alice",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Smith",
			Enabled:   true,
		},
	}

	kcClient := NewMockKeycloakUserClient(kcUsers)
	repo := NewMockSyncUserRepository()
	repo.SetListError(errors.New("list failed"))

	config := worker.UserSyncConfig{
		Interval:  time.Hour,
		BatchSize: 100,
		Enabled:   true,
	}

	w := worker.NewUserSyncWorker(kcClient, repo, slog.Default(), config)

	// Should not fail, but log the error
	err := w.Sync(context.Background())

	require.NoError(t, err)
	// User should still be created
	assert.Equal(t, 1, repo.UserCount())
}
