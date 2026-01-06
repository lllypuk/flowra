package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Token store errors.
var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)

// TokenStore manages refresh tokens in Redis.
type TokenStore struct {
	client    *redis.Client
	keyPrefix string
}

// TokenStoreConfig contains configuration for TokenStore.
type TokenStoreConfig struct {
	Client    *redis.Client
	KeyPrefix string
}

const (
	defaultKeyPrefix = "auth:refresh_token:"
)

// NewTokenStore creates a new Redis-based token store.
func NewTokenStore(cfg TokenStoreConfig) *TokenStore {
	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = defaultKeyPrefix
	}

	return &TokenStore{
		client:    cfg.Client,
		keyPrefix: keyPrefix,
	}
}

// tokenKey generates the Redis key for a user's refresh token.
func (s *TokenStore) tokenKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", s.keyPrefix, userID.String())
}

// StoreRefreshToken stores a refresh token for a user with the given TTL.
func (s *TokenStore) StoreRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
	ttl time.Duration,
) error {
	if userID.IsZero() {
		return errors.New("userID is required")
	}
	if refreshToken == "" {
		return errors.New("refreshToken is required")
	}

	key := s.tokenKey(userID)
	err := s.client.Set(ctx, key, refreshToken, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a stored refresh token for a user.
func (s *TokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if userID.IsZero() {
		return "", errors.New("userID is required")
	}

	key := s.tokenKey(userID)
	token, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

// DeleteRefreshToken removes a user's refresh token (logout).
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	if userID.IsZero() {
		return errors.New("userID is required")
	}

	key := s.tokenKey(userID)
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

// IsTokenValid checks if a token ID is not in the blacklist.
// This can be used for additional security to track revoked access tokens.
func (s *TokenStore) IsTokenValid(ctx context.Context, tokenID string) (bool, error) {
	if tokenID == "" {
		return false, errors.New("tokenID is required")
	}

	blacklistKey := fmt.Sprintf("%sblacklist:%s", s.keyPrefix, tokenID)
	exists, err := s.client.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token validity: %w", err)
	}

	// If exists in blacklist, token is invalid
	return exists == 0, nil
}

// BlacklistToken adds a token ID to the blacklist with the given TTL.
// This can be used to revoke access tokens before they expire.
func (s *TokenStore) BlacklistToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	if tokenID == "" {
		return errors.New("tokenID is required")
	}

	blacklistKey := fmt.Sprintf("%sblacklist:%s", s.keyPrefix, tokenID)
	err := s.client.Set(ctx, blacklistKey, "revoked", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}
