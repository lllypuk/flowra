package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/auth"
	"github.com/lllypuk/flowra/tests/testutil"
)

func setupTokenStore(t *testing.T) *auth.TokenStore {
	t.Helper()

	client, prefix := testutil.SetupTestRedisWithPrefix(t)

	store := auth.NewTokenStore(auth.TokenStoreConfig{
		Client:    client,
		KeyPrefix: prefix,
	})

	return store
}

func TestNewTokenStore(t *testing.T) {
	t.Run("creates store with custom prefix", func(t *testing.T) {
		client := testutil.SetupTestRedis(t)

		store := auth.NewTokenStore(auth.TokenStoreConfig{
			Client:    client,
			KeyPrefix: "custom:prefix:",
		})

		require.NotNil(t, store)
	})

	t.Run("creates store with default prefix", func(t *testing.T) {
		client := testutil.SetupTestRedis(t)

		store := auth.NewTokenStore(auth.TokenStoreConfig{
			Client: client,
		})

		require.NotNil(t, store)
	})
}

func TestTokenStore_StoreRefreshToken(t *testing.T) {
	t.Run("successfully stores token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()
		token := "test-refresh-token"
		ttl := 1 * time.Hour

		err := store.StoreRefreshToken(ctx, userID, token, ttl)

		require.NoError(t, err)

		// Verify token was stored
		retrieved, err := store.GetRefreshToken(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, token, retrieved)
	})

	t.Run("overwrites existing token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()
		ttl := 1 * time.Hour

		// Store first token
		err := store.StoreRefreshToken(ctx, userID, "first-token", ttl)
		require.NoError(t, err)

		// Store second token (should overwrite)
		err = store.StoreRefreshToken(ctx, userID, "second-token", ttl)
		require.NoError(t, err)

		// Verify second token is stored
		retrieved, err := store.GetRefreshToken(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "second-token", retrieved)
	})

	t.Run("returns error for zero userID", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		err := store.StoreRefreshToken(ctx, "", "token", time.Hour)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID is required")
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()

		err := store.StoreRefreshToken(ctx, userID, "", time.Hour)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "refreshToken is required")
	})

	t.Run("token expires after TTL", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()
		shortTTL := 100 * time.Millisecond

		err := store.StoreRefreshToken(ctx, userID, "expiring-token", shortTTL)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(200 * time.Millisecond)

		// Token should be gone
		_, err = store.GetRefreshToken(ctx, userID)
		require.Error(t, err)
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	})
}

func TestTokenStore_GetRefreshToken(t *testing.T) {
	t.Run("successfully gets token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()
		expectedToken := "my-refresh-token"

		err := store.StoreRefreshToken(ctx, userID, expectedToken, time.Hour)
		require.NoError(t, err)

		token, err := store.GetRefreshToken(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, expectedToken, token)
	})

	t.Run("returns error for non-existent token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID() // Never stored

		_, err := store.GetRefreshToken(ctx, userID)

		require.Error(t, err)
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	})

	t.Run("returns error for zero userID", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		_, err := store.GetRefreshToken(ctx, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID is required")
	})
}

func TestTokenStore_DeleteRefreshToken(t *testing.T) {
	t.Run("successfully deletes token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()

		// Store token first
		err := store.StoreRefreshToken(ctx, userID, "token-to-delete", time.Hour)
		require.NoError(t, err)

		// Delete token
		err = store.DeleteRefreshToken(ctx, userID)
		require.NoError(t, err)

		// Verify token is gone
		_, err = store.GetRefreshToken(ctx, userID)
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	})

	t.Run("succeeds for non-existent token (idempotent)", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID() // Never stored

		err := store.DeleteRefreshToken(ctx, userID)

		// Should not error even if token didn't exist
		require.NoError(t, err)
	})

	t.Run("returns error for zero userID", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		err := store.DeleteRefreshToken(ctx, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID is required")
	})
}

func TestTokenStore_IsTokenValid(t *testing.T) {
	t.Run("returns true for non-blacklisted token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		valid, err := store.IsTokenValid(ctx, "some-token-id")

		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("returns false for blacklisted token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		tokenID := "blacklisted-token"

		// Blacklist the token
		err := store.BlacklistToken(ctx, tokenID, time.Hour)
		require.NoError(t, err)

		// Check validity
		valid, err := store.IsTokenValid(ctx, tokenID)

		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("returns error for empty tokenID", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		_, err := store.IsTokenValid(ctx, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tokenID is required")
	})
}

func TestTokenStore_BlacklistToken(t *testing.T) {
	t.Run("successfully blacklists token", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		tokenID := "token-to-blacklist"

		err := store.BlacklistToken(ctx, tokenID, time.Hour)

		require.NoError(t, err)

		// Verify token is blacklisted
		valid, err := store.IsTokenValid(ctx, tokenID)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("blacklist expires after TTL", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		tokenID := "short-lived-blacklist"
		shortTTL := 100 * time.Millisecond

		err := store.BlacklistToken(ctx, tokenID, shortTTL)
		require.NoError(t, err)

		// Verify initially blacklisted
		valid, err := store.IsTokenValid(ctx, tokenID)
		require.NoError(t, err)
		assert.False(t, valid)

		// Wait for blacklist to expire
		time.Sleep(200 * time.Millisecond)

		// Token should be valid again (blacklist expired)
		valid, err = store.IsTokenValid(ctx, tokenID)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("returns error for empty tokenID", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()

		err := store.BlacklistToken(ctx, "", time.Hour)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tokenID is required")
	})
}

func TestTokenStore_Integration(t *testing.T) {
	t.Run("full token lifecycle", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		userID := uuid.NewUUID()
		refreshToken := "lifecycle-refresh-token"

		// 1. Store token
		err := store.StoreRefreshToken(ctx, userID, refreshToken, time.Hour)
		require.NoError(t, err)

		// 2. Retrieve token
		retrieved, err := store.GetRefreshToken(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, refreshToken, retrieved)

		// 3. Update token (e.g., after refresh)
		newToken := "new-lifecycle-refresh-token"
		err = store.StoreRefreshToken(ctx, userID, newToken, time.Hour)
		require.NoError(t, err)

		// 4. Verify new token
		retrieved, err = store.GetRefreshToken(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, newToken, retrieved)

		// 5. Delete token (logout)
		err = store.DeleteRefreshToken(ctx, userID)
		require.NoError(t, err)

		// 6. Verify token is gone
		_, err = store.GetRefreshToken(ctx, userID)
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	})

	t.Run("multiple users with separate tokens", func(t *testing.T) {
		store := setupTokenStore(t)
		ctx := context.Background()
		user1 := uuid.NewUUID()
		user2 := uuid.NewUUID()

		// Store tokens for both users
		err := store.StoreRefreshToken(ctx, user1, "user1-token", time.Hour)
		require.NoError(t, err)

		err = store.StoreRefreshToken(ctx, user2, "user2-token", time.Hour)
		require.NoError(t, err)

		// Verify each user has their own token
		token1, err := store.GetRefreshToken(ctx, user1)
		require.NoError(t, err)
		assert.Equal(t, "user1-token", token1)

		token2, err := store.GetRefreshToken(ctx, user2)
		require.NoError(t, err)
		assert.Equal(t, "user2-token", token2)

		// Delete user1's token
		err = store.DeleteRefreshToken(ctx, user1)
		require.NoError(t, err)

		// User1's token should be gone
		_, err = store.GetRefreshToken(ctx, user1)
		require.ErrorIs(t, err, auth.ErrTokenNotFound)

		// User2's token should still exist
		token2, err = store.GetRefreshToken(ctx, user2)
		require.NoError(t, err)
		assert.Equal(t, "user2-token", token2)
	})
}
