package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJWTValidator is a mock implementation of keycloak.JWTValidator for testing.
type mockJWTValidator struct {
	claims *keycloak.TokenClaims
	err    error
	closed bool
}

func (m *mockJWTValidator) Validate(_ context.Context, _ string) (*keycloak.TokenClaims, error) {
	return m.claims, m.err
}

func (m *mockJWTValidator) Close() error {
	m.closed = true
	return nil
}

func TestNewKeycloakValidatorAdapter(t *testing.T) {
	t.Run("creates adapter with valid validator", func(t *testing.T) {
		validator := &mockJWTValidator{}
		adapter := middleware.NewKeycloakValidatorAdapter(validator)

		assert.NotNil(t, adapter)
	})

	t.Run("panics with nil validator", func(t *testing.T) {
		assert.Panics(t, func() {
			middleware.NewKeycloakValidatorAdapter(nil)
		})
	})

	t.Run("applies admin roles option", func(t *testing.T) {
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:     "user-123",
				Username:   "testuser",
				Email:      "test@example.com",
				RealmRoles: []string{"superadmin"},
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		}

		adapter := middleware.NewKeycloakValidatorAdapter(
			validator,
			middleware.WithAdminRoles("superadmin"),
		)

		claims, err := adapter.ValidateToken(context.Background(), "test-token")
		require.NoError(t, err)
		assert.True(t, claims.IsSystemAdmin)
	})
}

func TestKeycloakValidatorAdapter_ValidateToken(t *testing.T) {
	t.Run("successfully validates token and converts claims", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:        "keycloak-user-123",
				Email:         "user@example.com",
				EmailVerified: true,
				Username:      "testuser",
				Name:          "Test User",
				GivenName:     "Test",
				FamilyName:    "User",
				RealmRoles:    []string{"user", "editor"},
				Groups:        []string{"/team-a", "/team-b"},
				SessionState:  "session-123",
				IssuedAt:      time.Now(),
				ExpiresAt:     expiresAt,
			},
		}

		adapter := middleware.NewKeycloakValidatorAdapter(validator)
		claims, err := adapter.ValidateToken(context.Background(), "valid-token")

		require.NoError(t, err)
		require.NotNil(t, claims)

		// Verify claim mapping
		assert.Equal(t, "keycloak-user-123", claims.ExternalUserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, "user@example.com", claims.Email)
		assert.Equal(t, []string{"user", "editor"}, claims.Roles)
		assert.Equal(t, []string{"/team-a", "/team-b"}, claims.Groups)
		assert.Equal(t, expiresAt, claims.ExpiresAt)
		assert.False(t, claims.IsSystemAdmin)
		// UserID should be zero (internal ID not set by adapter)
		assert.True(t, claims.UserID.IsZero())
	})

	t.Run("identifies system admin from admin role", func(t *testing.T) {
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:     "admin-user",
				Username:   "admin",
				Email:      "admin@example.com",
				RealmRoles: []string{"user", "admin"},
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		}

		adapter := middleware.NewKeycloakValidatorAdapter(validator)
		claims, err := adapter.ValidateToken(context.Background(), "admin-token")

		require.NoError(t, err)
		assert.True(t, claims.IsSystemAdmin)
	})

	t.Run("identifies system admin from system-admin role", func(t *testing.T) {
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:     "admin-user",
				Username:   "sysadmin",
				Email:      "sysadmin@example.com",
				RealmRoles: []string{"user", "system-admin"},
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		}

		adapter := middleware.NewKeycloakValidatorAdapter(validator)
		claims, err := adapter.ValidateToken(context.Background(), "admin-token")

		require.NoError(t, err)
		assert.True(t, claims.IsSystemAdmin)
	})

	t.Run("custom admin roles override defaults", func(t *testing.T) {
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:     "user-123",
				Username:   "testuser",
				Email:      "test@example.com",
				RealmRoles: []string{"admin"}, // default admin role
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		}

		// Only "root" is admin now, not "admin"
		adapter := middleware.NewKeycloakValidatorAdapter(
			validator,
			middleware.WithAdminRoles("root"),
		)

		claims, err := adapter.ValidateToken(context.Background(), "test-token")

		require.NoError(t, err)
		assert.False(t, claims.IsSystemAdmin)
	})

	t.Run("handles empty groups and roles", func(t *testing.T) {
		validator := &mockJWTValidator{
			claims: &keycloak.TokenClaims{
				UserID:     "user-123",
				Username:   "testuser",
				Email:      "test@example.com",
				RealmRoles: nil,
				Groups:     nil,
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		}

		adapter := middleware.NewKeycloakValidatorAdapter(validator)
		claims, err := adapter.ValidateToken(context.Background(), "test-token")

		require.NoError(t, err)
		assert.Nil(t, claims.Roles)
		assert.Nil(t, claims.Groups)
		assert.False(t, claims.IsSystemAdmin)
	})
}

func TestKeycloakValidatorAdapter_ErrorMapping(t *testing.T) {
	tests := []struct {
		name          string
		keycloakErr   error
		expectedErr   error
		checkContains bool
	}{
		{
			name:        "maps ErrInvalidToken",
			keycloakErr: keycloak.ErrInvalidToken,
			expectedErr: middleware.ErrInvalidToken,
		},
		{
			name:        "maps ErrTokenExpired",
			keycloakErr: keycloak.ErrTokenExpired,
			expectedErr: middleware.ErrTokenExpired,
		},
		{
			name:        "maps ErrInvalidClaims to ErrInvalidToken",
			keycloakErr: keycloak.ErrInvalidClaims,
			expectedErr: middleware.ErrInvalidToken,
		},
		{
			name:        "maps ErrMissingSubject to ErrInvalidToken",
			keycloakErr: keycloak.ErrMissingSubject,
			expectedErr: middleware.ErrInvalidToken,
		},
		{
			name:        "maps ErrInvalidIssuer to ErrInvalidToken",
			keycloakErr: keycloak.ErrInvalidIssuer,
			expectedErr: middleware.ErrInvalidToken,
		},
		{
			name:        "maps ErrInvalidAudience to ErrInvalidToken",
			keycloakErr: keycloak.ErrInvalidAudience,
			expectedErr: middleware.ErrInvalidToken,
		},
		{
			name:          "maps unknown errors to ErrInvalidToken",
			keycloakErr:   errors.New("some unknown error"),
			expectedErr:   middleware.ErrInvalidToken,
			checkContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &mockJWTValidator{
				err: tt.keycloakErr,
			}

			adapter := middleware.NewKeycloakValidatorAdapter(validator)
			_, err := adapter.ValidateToken(context.Background(), "test-token")

			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.checkContains {
				assert.ErrorContains(t, err, tt.keycloakErr.Error())
			}
		})
	}
}

func TestKeycloakValidatorAdapter_Close(t *testing.T) {
	validator := &mockJWTValidator{}
	adapter := middleware.NewKeycloakValidatorAdapter(validator)

	err := adapter.Close()

	require.NoError(t, err)
	assert.True(t, validator.closed)
}

// testContextKey is a custom type for context keys in tests.
type testContextKey string

func TestKeycloakValidatorAdapter_ContextPropagation(t *testing.T) {
	// Verify that context is properly passed to the underlying validator
	const testKey testContextKey = "test-key"
	var receivedValue any

	customValidator := &contextCapturingValidator{
		captureCtx: func(capturedCtx context.Context) {
			receivedValue = capturedCtx.Value(testKey)
		},
		claims: &keycloak.TokenClaims{
			UserID:    "user-123",
			Username:  "testuser",
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}

	adapter := middleware.NewKeycloakValidatorAdapter(customValidator)

	ctx := context.WithValue(context.Background(), testKey, "test-value")
	_, err := adapter.ValidateToken(ctx, "test-token")

	require.NoError(t, err)
	assert.Equal(t, "test-value", receivedValue)
}

// contextCapturingValidator captures the context passed to Validate.
type contextCapturingValidator struct {
	captureCtx func(ctx context.Context)
	claims     *keycloak.TokenClaims
}

func (v *contextCapturingValidator) Validate(ctx context.Context, _ string) (*keycloak.TokenClaims, error) {
	if v.captureCtx != nil {
		v.captureCtx(ctx)
	}
	return v.claims, nil
}

func (v *contextCapturingValidator) Close() error {
	return nil
}
