package service_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/auth"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/service"
)

// Mock implementations

type mockOAuthClient struct {
	exchangeCodeFunc func(ctx context.Context, code, redirectURI string) (*keycloak.TokenResponse, error)
	refreshTokenFunc func(ctx context.Context, refreshToken string) (*keycloak.TokenResponse, error)
	revokeTokenFunc  func(ctx context.Context, refreshToken string) error
	getUserInfoFunc  func(ctx context.Context, accessToken string) (*keycloak.UserInfo, error)
}

func (m *mockOAuthClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*keycloak.TokenResponse, error) {
	if m.exchangeCodeFunc != nil {
		return m.exchangeCodeFunc(ctx, code, redirectURI)
	}
	return &keycloak.TokenResponse{}, nil
}

func (m *mockOAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*keycloak.TokenResponse, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, refreshToken)
	}
	return &keycloak.TokenResponse{}, nil
}

func (m *mockOAuthClient) RevokeToken(ctx context.Context, refreshToken string) error {
	if m.revokeTokenFunc != nil {
		return m.revokeTokenFunc(ctx, refreshToken)
	}
	return nil
}

func (m *mockOAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*keycloak.UserInfo, error) {
	if m.getUserInfoFunc != nil {
		return m.getUserInfoFunc(ctx, accessToken)
	}
	return &keycloak.UserInfo{}, nil
}

type mockTokenStore struct {
	storeFunc  func(ctx context.Context, userID uuid.UUID, refreshToken string, ttl time.Duration) error
	getFunc    func(ctx context.Context, userID uuid.UUID) (string, error)
	deleteFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockTokenStore) StoreRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
	ttl time.Duration,
) error {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, userID, refreshToken, ttl)
	}
	return nil
}

func (m *mockTokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, userID)
	}
	return "", nil
}

func (m *mockTokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID)
	}
	return nil
}

type mockUserRepository struct {
	findByExternalIDFunc func(ctx context.Context, externalID string) (*user.User, error)
	saveFunc             func(ctx context.Context, u *user.User) error
}

func (m *mockUserRepository) FindByExternalID(ctx context.Context, externalID string) (*user.User, error) {
	if m.findByExternalIDFunc != nil {
		return m.findByExternalIDFunc(ctx, externalID)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) Save(ctx context.Context, u *user.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, u)
	}
	return nil
}

// Helper functions

func createTestEchoContext() echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func createTestUser(t *testing.T) *user.User {
	t.Helper()
	u, err := user.NewUser("keycloak-123", "testuser", "test@example.com", "Test User")
	require.NoError(t, err)
	return u
}

func createDefaultAuthServiceConfig() service.AuthServiceConfig {
	return service.AuthServiceConfig{
		OAuthClient: &mockOAuthClient{},
		TokenStore:  &mockTokenStore{},
		UserRepo:    &mockUserRepository{},
	}
}

// Tests

func TestNewAuthService(t *testing.T) {
	t.Run("creates service with all dependencies", func(t *testing.T) {
		cfg := createDefaultAuthServiceConfig()
		svc := service.NewAuthService(cfg)

		assert.NotNil(t, svc)
	})

	t.Run("creates service with nil logger (uses default)", func(t *testing.T) {
		cfg := createDefaultAuthServiceConfig()
		cfg.Logger = nil
		svc := service.NewAuthService(cfg)

		assert.NotNil(t, svc)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("successfully logs in new user", func(t *testing.T) {
		var savedUser *user.User

		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, code, redirectURI string) (*keycloak.TokenResponse, error) {
				assert.Equal(t, "valid-code", code)
				assert.Equal(t, "http://localhost/callback", redirectURI)
				return &keycloak.TokenResponse{
					AccessToken:      "access-token",
					RefreshToken:     "refresh-token",
					ExpiresIn:        3600,
					RefreshExpiresIn: 7200,
				}, nil
			},
			getUserInfoFunc: func(_ context.Context, accessToken string) (*keycloak.UserInfo, error) {
				assert.Equal(t, "access-token", accessToken)
				return &keycloak.UserInfo{
					Sub:               "keycloak-new-user",
					PreferredUsername: "newuser",
					Email:             "newuser@example.com",
					Name:              "New User",
				}, nil
			},
		}

		tokenStore := &mockTokenStore{
			storeFunc: func(_ context.Context, userID uuid.UUID, refreshToken string, ttl time.Duration) error {
				assert.NotEmpty(t, userID)
				assert.Equal(t, "refresh-token", refreshToken)
				assert.Equal(t, 7200*time.Second, ttl)
				return nil
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, _ string) (*user.User, error) {
				return nil, errors.New("not found")
			},
			saveFunc: func(_ context.Context, u *user.User) error {
				savedUser = u
				return nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  tokenStore,
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.Equal(t, "access-token", result.AccessToken)
		assert.Equal(t, "refresh-token", result.RefreshToken)
		assert.Equal(t, 3600, result.ExpiresIn)
		assert.NotNil(t, result.User)
		assert.Equal(t, "newuser", result.User.Username())
		assert.Equal(t, "newuser@example.com", result.User.Email())

		// Verify user was saved
		require.NotNil(t, savedUser)
		assert.Equal(t, "keycloak-new-user", savedUser.ExternalID())
	})

	t.Run("successfully logs in existing user", func(t *testing.T) {
		existingUser := createTestUser(t)

		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{
					AccessToken:      "access-token",
					RefreshToken:     "refresh-token",
					ExpiresIn:        3600,
					RefreshExpiresIn: 7200,
				}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return &keycloak.UserInfo{
					Sub:               existingUser.ExternalID(),
					PreferredUsername: existingUser.Username(),
					Email:             existingUser.Email(),
					Name:              existingUser.DisplayName(),
				}, nil
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, externalID string) (*user.User, error) {
				if externalID == existingUser.ExternalID() {
					return existingUser, nil
				}
				return nil, errors.New("not found")
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.Equal(t, existingUser.ID(), result.User.ID())
		assert.Equal(t, existingUser.Username(), result.User.Username())
	})

	t.Run("updates existing user when data changes", func(t *testing.T) {
		existingUser := createTestUser(t)
		var userUpdated bool

		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{
					AccessToken:      "access-token",
					RefreshToken:     "refresh-token",
					ExpiresIn:        3600,
					RefreshExpiresIn: 7200,
				}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return &keycloak.UserInfo{
					Sub:               existingUser.ExternalID(),
					PreferredUsername: existingUser.Username(),
					Email:             "newemail@example.com", // Changed email
					Name:              "Updated Name",         // Changed name
				}, nil
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, _ string) (*user.User, error) {
				return existingUser, nil
			},
			saveFunc: func(_ context.Context, u *user.User) error {
				userUpdated = true
				assert.Equal(t, "newemail@example.com", u.Email())
				assert.Equal(t, "Updated Name", u.DisplayName())
				return nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.NotNil(t, result.User)
		assert.True(t, userUpdated)
	})

	t.Run("returns error on code exchange failure", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return nil, keycloak.ErrTokenExchangeFailed
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "invalid-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to exchange code")
	})

	t.Run("returns error on user info failure", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{AccessToken: "token"}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return nil, keycloak.ErrUserInfoFailed
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get user info")
	})

	t.Run("returns error on user save failure for new user", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{AccessToken: "token"}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return &keycloak.UserInfo{
					Sub:               "new-user-id",
					PreferredUsername: "newuser",
					Email:             "new@example.com",
					Name:              "New User",
				}, nil
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, _ string) (*user.User, error) {
				return nil, errors.New("not found")
			},
			saveFunc: func(_ context.Context, _ *user.User) error {
				return errors.New("database error")
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, service.ErrUserSyncFailed)
	})

	t.Run("succeeds even if token store fails", func(t *testing.T) {
		existingUser := createTestUser(t)

		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{
					AccessToken:      "access-token",
					RefreshToken:     "refresh-token",
					ExpiresIn:        3600,
					RefreshExpiresIn: 7200,
				}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return &keycloak.UserInfo{
					Sub:               existingUser.ExternalID(),
					PreferredUsername: existingUser.Username(),
					Email:             existingUser.Email(),
					Name:              existingUser.DisplayName(),
				}, nil
			},
		}

		tokenStore := &mockTokenStore{
			storeFunc: func(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
				return errors.New("redis error")
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, _ string) (*user.User, error) {
				return existingUser, nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  tokenStore,
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		// Should still succeed - token store failure is logged but not fatal
		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("does not store token when refresh token is empty", func(t *testing.T) {
		existingUser := createTestUser(t)
		var tokenStored bool

		oauthClient := &mockOAuthClient{
			exchangeCodeFunc: func(_ context.Context, _, _ string) (*keycloak.TokenResponse, error) {
				return &keycloak.TokenResponse{
					AccessToken:  "access-token",
					RefreshToken: "", // Empty refresh token
					ExpiresIn:    3600,
				}, nil
			},
			getUserInfoFunc: func(_ context.Context, _ string) (*keycloak.UserInfo, error) {
				return &keycloak.UserInfo{
					Sub:               existingUser.ExternalID(),
					PreferredUsername: existingUser.Username(),
					Email:             existingUser.Email(),
					Name:              existingUser.DisplayName(),
				}, nil
			},
		}

		tokenStore := &mockTokenStore{
			storeFunc: func(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
				tokenStored = true
				return nil
			},
		}

		userRepo := &mockUserRepository{
			findByExternalIDFunc: func(_ context.Context, _ string) (*user.User, error) {
				return existingUser, nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  tokenStore,
			UserRepo:    userRepo,
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.Login(createTestEchoContext(), "valid-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, tokenStored, "token should not be stored when refresh token is empty")
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("successfully logs out user", func(t *testing.T) {
		userID := uuid.NewUUID()
		var tokenRevoked, tokenDeleted bool

		oauthClient := &mockOAuthClient{
			revokeTokenFunc: func(_ context.Context, refreshToken string) error {
				assert.Equal(t, "stored-refresh-token", refreshToken)
				tokenRevoked = true
				return nil
			},
		}

		tokenStore := &mockTokenStore{
			getFunc: func(_ context.Context, uid uuid.UUID) (string, error) {
				assert.Equal(t, userID, uid)
				return "stored-refresh-token", nil
			},
			deleteFunc: func(_ context.Context, uid uuid.UUID) error {
				assert.Equal(t, userID, uid)
				tokenDeleted = true
				return nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  tokenStore,
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		err := svc.Logout(createTestEchoContext(), userID)

		require.NoError(t, err)
		assert.True(t, tokenRevoked)
		assert.True(t, tokenDeleted)
	})

	t.Run("succeeds when token not found (already logged out)", func(t *testing.T) {
		userID := uuid.NewUUID()

		tokenStore := &mockTokenStore{
			getFunc: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "", auth.ErrTokenNotFound
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: &mockOAuthClient{},
			TokenStore:  tokenStore,
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		err := svc.Logout(createTestEchoContext(), userID)

		require.NoError(t, err)
	})

	t.Run("succeeds even if keycloak revoke fails", func(t *testing.T) {
		userID := uuid.NewUUID()
		var tokenDeleted bool

		oauthClient := &mockOAuthClient{
			revokeTokenFunc: func(_ context.Context, _ string) error {
				return errors.New("keycloak error")
			},
		}

		tokenStore := &mockTokenStore{
			getFunc: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "refresh-token", nil
			},
			deleteFunc: func(_ context.Context, _ uuid.UUID) error {
				tokenDeleted = true
				return nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  tokenStore,
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		err := svc.Logout(createTestEchoContext(), userID)

		// Should succeed despite Keycloak failure
		require.NoError(t, err)
		assert.True(t, tokenDeleted)
	})

	t.Run("returns error when token deletion fails", func(t *testing.T) {
		userID := uuid.NewUUID()

		tokenStore := &mockTokenStore{
			getFunc: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "refresh-token", nil
			},
			deleteFunc: func(_ context.Context, _ uuid.UUID) error {
				return errors.New("redis error")
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: &mockOAuthClient{},
			TokenStore:  tokenStore,
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		err := svc.Logout(createTestEchoContext(), userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete refresh token")
	})

	t.Run("returns error when get token fails with unexpected error", func(t *testing.T) {
		userID := uuid.NewUUID()

		tokenStore := &mockTokenStore{
			getFunc: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "", errors.New("unexpected redis error")
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: &mockOAuthClient{},
			TokenStore:  tokenStore,
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		err := svc.Logout(createTestEchoContext(), userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get refresh token")
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("successfully refreshes token", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			refreshTokenFunc: func(_ context.Context, refreshToken string) (*keycloak.TokenResponse, error) {
				assert.Equal(t, "old-refresh-token", refreshToken)
				return &keycloak.TokenResponse{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    3600,
				}, nil
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.RefreshToken(createTestEchoContext(), "old-refresh-token")

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", result.AccessToken)
		assert.Equal(t, "new-refresh-token", result.RefreshToken)
		assert.Equal(t, 3600, result.ExpiresIn)
	})

	t.Run("returns error on expired refresh token", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			refreshTokenFunc: func(_ context.Context, _ string) (*keycloak.TokenResponse, error) {
				return nil, keycloak.ErrTokenRefreshFailed
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.RefreshToken(createTestEchoContext(), "expired-token")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to refresh token")
	})

	t.Run("returns error on keycloak unavailable", func(t *testing.T) {
		oauthClient := &mockOAuthClient{
			refreshTokenFunc: func(_ context.Context, _ string) (*keycloak.TokenResponse, error) {
				return nil, errors.New("connection refused")
			},
		}

		cfg := service.AuthServiceConfig{
			OAuthClient: oauthClient,
			TokenStore:  &mockTokenStore{},
			UserRepo:    &mockUserRepository{},
		}
		svc := service.NewAuthService(cfg)

		result, err := svc.RefreshToken(createTestEchoContext(), "refresh-token")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAuthService_CompileTimeAssertion(t *testing.T) {
	t.Run("AuthService implements httphandler.AuthService", func(_ *testing.T) {
		cfg := createDefaultAuthServiceConfig()
		var _ httphandler.AuthService = service.NewAuthService(cfg)
	})
}
