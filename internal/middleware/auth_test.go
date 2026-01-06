package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenValidator is a mock implementation of TokenValidator for testing.
type mockTokenValidator struct {
	claims *middleware.TokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateToken(_ context.Context, _ string) (*middleware.TokenClaims, error) {
	return m.claims, m.err
}

// mockUserResolver is a mock implementation of UserResolver for testing.
type mockUserResolver struct {
	userID uuid.UUID
	err    error
}

func (m *mockUserResolver) ResolveUser(_ context.Context, _, _, _ string) (uuid.UUID, error) {
	return m.userID, m.err
}

func TestDefaultAuthConfig(t *testing.T) {
	config := middleware.DefaultAuthConfig()

	assert.NotNil(t, config.Logger)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.Contains(t, config.SkipPaths, "/api/v1/auth/login")
	assert.Contains(t, config.SkipPaths, "/api/v1/auth/register")
	assert.Contains(t, config.AllowExpiredForPaths, "/api/v1/auth/refresh")
}

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	e := echo.New()

	validator := &mockTokenValidator{
		claims: &middleware.TokenClaims{},
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
	}

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "UNAUTHORIZED")
	assert.Contains(t, rec.Body.String(), "Missing authorization header")
}

func TestAuth_InvalidAuthorizationHeaderFormat(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "no bearer prefix",
			authHeader: "Basic token123",
		},
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
		},
		{
			name:       "just bearer",
			authHeader: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			validator := &mockTokenValidator{
				claims: &middleware.TokenClaims{},
			}

			config := middleware.AuthConfig{
				TokenValidator: validator,
			}

			e.Use(middleware.Auth(config))
			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(echo.HeaderAuthorization, tt.authHeader)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "Invalid authorization header")
		})
	}
}

func TestAuth_SkipPaths(t *testing.T) {
	e := echo.New()

	validator := &mockTokenValidator{
		err: middleware.ErrInvalidToken,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
		SkipPaths:      []string{"/health", "/public"},
	}

	e.Use(middleware.Auth(config))
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})
	e.GET("/public", func(c echo.Context) error {
		return c.String(http.StatusOK, "public")
	})
	e.GET("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "protected")
	})

	// Test skip path /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "healthy", rec.Body.String())

	// Test skip path /public
	req = httptest.NewRequest(http.MethodGet, "/public", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public", rec.Body.String())

	// Test protected path without auth
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	e := echo.New()

	userID := uuid.NewUUID()
	expectedClaims := &middleware.TokenClaims{
		UserID:         userID,
		ExternalUserID: "ext-123",
		Username:       "testuser",
		Email:          "test@example.com",
		Roles:          []string{"user", "admin"},
		IsSystemAdmin:  false,
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	validator := &mockTokenValidator{
		claims: expectedClaims,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
	}

	var capturedUserID uuid.UUID
	var capturedUsername string
	var capturedRoles []string

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		capturedUserID = middleware.GetUserID(c)
		capturedUsername = middleware.GetUsername(c)
		capturedRoles = middleware.GetRoles(c)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedUserID)
	assert.Equal(t, "testuser", capturedUsername)
	assert.Equal(t, []string{"user", "admin"}, capturedRoles)
}

func TestAuth_TokenValidationFailed(t *testing.T) {
	e := echo.New()

	validator := &mockTokenValidator{
		err: middleware.ErrInvalidToken,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
	}

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid token")
}

func TestAuth_TokenExpired(t *testing.T) {
	e := echo.New()

	validator := &mockTokenValidator{
		err: middleware.ErrTokenExpired,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
	}

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer expired-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_EXPIRED")
	assert.Contains(t, rec.Body.String(), "Token has expired")
}

func TestAuth_AllowExpiredTokenForRefresh(t *testing.T) {
	e := echo.New()

	expiredClaims := &middleware.TokenClaims{
		UserID:         uuid.NewUUID(),
		ExternalUserID: "ext-123",
		Username:       "testuser",
		ExpiresAt:      time.Now().Add(-time.Hour), // Expired
	}

	validator := &mockTokenValidator{
		claims: expiredClaims,
		err:    middleware.ErrTokenExpired,
	}

	config := middleware.AuthConfig{
		TokenValidator:       validator,
		AllowExpiredForPaths: []string{"/api/v1/auth/refresh"},
	}

	var capturedUsername string

	e.Use(middleware.Auth(config))
	e.POST("/api/v1/auth/refresh", func(c echo.Context) error {
		capturedUsername = middleware.GetUsername(c)
		return c.String(http.StatusOK, "refreshed")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer expired-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "refreshed", rec.Body.String())
	assert.Equal(t, "testuser", capturedUsername)
}

func TestAuth_WithUserResolver(t *testing.T) {
	e := echo.New()

	internalUserID := uuid.NewUUID()
	claims := &middleware.TokenClaims{
		ExternalUserID: "ext-123",
		Username:       "testuser",
		Email:          "test@example.com",
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	validator := &mockTokenValidator{
		claims: claims,
	}

	resolver := &mockUserResolver{
		userID: internalUserID,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
		UserResolver:   resolver,
	}

	var capturedUserID uuid.UUID

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		capturedUserID = middleware.GetUserID(c)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, internalUserID, capturedUserID)
}

func TestAuth_UserResolverFailed(t *testing.T) {
	e := echo.New()

	claims := &middleware.TokenClaims{
		ExternalUserID: "ext-123",
		Username:       "testuser",
		Email:          "test@example.com",
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	validator := &mockTokenValidator{
		claims: claims,
	}

	resolver := &mockUserResolver{
		err: errors.New("user not found"),
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
		UserResolver:   resolver,
	}

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "USER_NOT_FOUND")
}

func TestAuth_NoTokenValidator(t *testing.T) {
	e := echo.New()

	config := middleware.AuthConfig{
		TokenValidator: nil,
	}

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer some-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	userID := middleware.GetUserID(c)
	assert.True(t, userID.IsZero())

	// Test with user ID set
	expectedID := uuid.NewUUID()
	c.Set(string(middleware.ContextKeyUserID), expectedID)
	userID = middleware.GetUserID(c)
	assert.Equal(t, expectedID, userID)
}

func TestGetExternalUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	externalID := middleware.GetExternalUserID(c)
	assert.Empty(t, externalID)

	// Test with external ID set
	c.Set(string(middleware.ContextKeyExternalUserID), "ext-123")
	externalID = middleware.GetExternalUserID(c)
	assert.Equal(t, "ext-123", externalID)
}

func TestGetUsername(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	username := middleware.GetUsername(c)
	assert.Empty(t, username)

	// Test with username set
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	username = middleware.GetUsername(c)
	assert.Equal(t, "testuser", username)
}

func TestGetEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	email := middleware.GetEmail(c)
	assert.Empty(t, email)

	// Test with email set
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	email = middleware.GetEmail(c)
	assert.Equal(t, "test@example.com", email)
}

func TestGetRoles(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	roles := middleware.GetRoles(c)
	assert.Nil(t, roles)

	// Test with roles set
	expectedRoles := []string{"user", "admin"}
	c.Set(string(middleware.ContextKeyRoles), expectedRoles)
	roles = middleware.GetRoles(c)
	assert.Equal(t, expectedRoles, roles)
}

func TestIsSystemAdmin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	isAdmin := middleware.IsSystemAdmin(c)
	assert.False(t, isAdmin)

	// Test with admin flag set to false
	c.Set(string(middleware.ContextKeyIsSystemAdmin), false)
	isAdmin = middleware.IsSystemAdmin(c)
	assert.False(t, isAdmin)

	// Test with admin flag set to true
	c.Set(string(middleware.ContextKeyIsSystemAdmin), true)
	isAdmin = middleware.IsSystemAdmin(c)
	assert.True(t, isAdmin)
}

func TestHasRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	roles := []string{"user", "editor"}
	c.Set(string(middleware.ContextKeyRoles), roles)

	assert.True(t, middleware.HasRole(c, "user"))
	assert.True(t, middleware.HasRole(c, "editor"))
	assert.False(t, middleware.HasRole(c, "admin"))
}

func TestHasAnyRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	roles := []string{"user", "editor"}
	c.Set(string(middleware.ContextKeyRoles), roles)

	assert.True(t, middleware.HasAnyRole(c, "user", "admin"))
	assert.True(t, middleware.HasAnyRole(c, "admin", "editor"))
	assert.False(t, middleware.HasAnyRole(c, "admin", "superuser"))
}

func TestRequireRole(t *testing.T) {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Simulate authenticated user with roles
			c.Set(string(middleware.ContextKeyRoles), []string{"user", "editor"})
			return next(c)
		}
	})

	// Route requiring admin role
	e.GET("/admin", func(c echo.Context) error {
		return c.String(http.StatusOK, "admin")
	}, middleware.RequireRole("admin"))

	// Route requiring editor role
	e.GET("/editor", func(c echo.Context) error {
		return c.String(http.StatusOK, "editor")
	}, middleware.RequireRole("editor"))

	// Test admin route - should fail
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Test editor route - should succeed
	req = httptest.NewRequest(http.MethodGet, "/editor", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "editor", rec.Body.String())
}

func TestRequireAnyRole(t *testing.T) {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyRoles), []string{"user"})
			return next(c)
		}
	})

	e.GET("/content", func(c echo.Context) error {
		return c.String(http.StatusOK, "content")
	}, middleware.RequireAnyRole("editor", "admin"))

	e.GET("/view", func(c echo.Context) error {
		return c.String(http.StatusOK, "view")
	}, middleware.RequireAnyRole("user", "guest"))

	// Test content route - should fail
	req := httptest.NewRequest(http.MethodGet, "/content", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Test view route - should succeed
	req = httptest.NewRequest(http.MethodGet, "/view", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireSystemAdmin(t *testing.T) {
	tests := []struct {
		name         string
		isAdmin      bool
		expectedCode int
	}{
		{
			name:         "system admin allowed",
			isAdmin:      true,
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-admin forbidden",
			isAdmin:      false,
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set(string(middleware.ContextKeyIsSystemAdmin), tt.isAdmin)
					return next(c)
				}
			})

			e.GET("/admin-only", func(c echo.Context) error {
				return c.String(http.StatusOK, "admin-only")
			}, middleware.RequireSystemAdmin())

			req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestStaticTokenValidator(t *testing.T) {
	validator := middleware.NewStaticTokenValidator("secret")

	t.Run("valid dev token", func(t *testing.T) {
		claims, err := validator.ValidateToken(context.Background(), "dev-token-user123")
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user123", claims.ExternalUserID)
		assert.Contains(t, claims.Username, "user123")
		assert.Contains(t, claims.Email, "user123")
		assert.Contains(t, claims.Roles, "user")
	})

	t.Run("empty token", func(t *testing.T) {
		claims, err := validator.ValidateToken(context.Background(), "")
		require.Error(t, err)
		require.ErrorIs(t, err, middleware.ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("invalid token format", func(t *testing.T) {
		claims, err := validator.ValidateToken(context.Background(), "random-token")
		require.Error(t, err)
		require.ErrorIs(t, err, middleware.ErrInvalidToken)
		assert.Nil(t, claims)
	})
}

func TestConstantTimeCompare(t *testing.T) {
	assert.True(t, middleware.ConstantTimeCompare("test", "test"))
	assert.True(t, middleware.ConstantTimeCompare("", ""))
	assert.False(t, middleware.ConstantTimeCompare("test", "Test"))
	assert.False(t, middleware.ConstantTimeCompare("test", "test1"))
	assert.False(t, middleware.ConstantTimeCompare("test1", "test"))
}

func TestAuth_ContextEnrichment(t *testing.T) {
	e := echo.New()

	claims := &middleware.TokenClaims{
		UserID:         uuid.NewUUID(),
		ExternalUserID: "ext-456",
		Username:       "contextuser",
		Email:          "context@example.com",
		Roles:          []string{"reader", "writer"},
		IsSystemAdmin:  true,
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	validator := &mockTokenValidator{
		claims: claims,
	}

	config := middleware.AuthConfig{
		TokenValidator: validator,
	}

	var extractedUserID uuid.UUID
	var extractedExternalID string
	var extractedUsername string
	var extractedEmail string
	var extractedRoles []string
	var extractedIsAdmin bool

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		extractedUserID = middleware.GetUserID(c)
		extractedExternalID = middleware.GetExternalUserID(c)
		extractedUsername = middleware.GetUsername(c)
		extractedEmail = middleware.GetEmail(c)
		extractedRoles = middleware.GetRoles(c)
		extractedIsAdmin = middleware.IsSystemAdmin(c)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, claims.UserID, extractedUserID)
	assert.Equal(t, claims.ExternalUserID, extractedExternalID)
	assert.Equal(t, claims.Username, extractedUsername)
	assert.Equal(t, claims.Email, extractedEmail)
	assert.Equal(t, claims.Roles, extractedRoles)
	assert.Equal(t, claims.IsSystemAdmin, extractedIsAdmin)
}

func TestGetUser(t *testing.T) {
	t.Run("returns claims when set", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedClaims := &middleware.TokenClaims{
			UserID:         uuid.NewUUID(),
			ExternalUserID: "ext-123",
			Username:       "testuser",
			Email:          "test@example.com",
			Roles:          []string{"user"},
			Groups:         []string{"/team-a"},
			IsSystemAdmin:  false,
		}
		c.Set("claims", expectedClaims)

		result := middleware.GetUser(c)

		require.NotNil(t, result)
		assert.Equal(t, expectedClaims.UserID, result.UserID)
		assert.Equal(t, expectedClaims.ExternalUserID, result.ExternalUserID)
		assert.Equal(t, expectedClaims.Username, result.Username)
		assert.Equal(t, expectedClaims.Email, result.Email)
		assert.Equal(t, expectedClaims.Roles, result.Roles)
		assert.Equal(t, expectedClaims.Groups, result.Groups)
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		result := middleware.GetUser(c)

		assert.Nil(t, result)
	})

	t.Run("returns nil when wrong type", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("claims", "not a claims object")

		result := middleware.GetUser(c)

		assert.Nil(t, result)
	})
}

func TestGetGroups(t *testing.T) {
	t.Run("returns groups when set", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedGroups := []string{"/team-a", "/team-b", "/admins"}
		c.Set("groups", expectedGroups)

		result := middleware.GetGroups(c)

		assert.Equal(t, expectedGroups, result)
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		result := middleware.GetGroups(c)

		assert.Nil(t, result)
	})
}

func TestInGroup(t *testing.T) {
	tests := []struct {
		name     string
		groups   []string
		check    string
		expected bool
	}{
		{
			name:     "exact match with slash",
			groups:   []string{"/team-a", "/team-b"},
			check:    "/team-a",
			expected: true,
		},
		{
			name:     "match without slash when stored with slash",
			groups:   []string{"/team-a", "/team-b"},
			check:    "team-a",
			expected: true,
		},
		{
			name:     "match with slash when stored without slash",
			groups:   []string{"team-a", "team-b"},
			check:    "/team-a",
			expected: true,
		},
		{
			name:     "exact match without slash",
			groups:   []string{"team-a", "team-b"},
			check:    "team-a",
			expected: true,
		},
		{
			name:     "no match",
			groups:   []string{"/team-a", "/team-b"},
			check:    "team-c",
			expected: false,
		},
		{
			name:     "empty groups",
			groups:   []string{},
			check:    "team-a",
			expected: false,
		},
		{
			name:     "nil groups",
			groups:   nil,
			check:    "team-a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.groups != nil {
				c.Set("groups", tt.groups)
			}

			result := middleware.InGroup(c, tt.check)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAnyGroup(t *testing.T) {
	t.Run("returns true when user has one of the groups", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("groups", []string{"/team-a", "/team-b"})

		assert.True(t, middleware.HasAnyGroup(c, "team-c", "team-a"))
	})

	t.Run("returns false when user has none of the groups", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("groups", []string{"/team-a", "/team-b"})

		assert.False(t, middleware.HasAnyGroup(c, "team-c", "team-d"))
	})

	t.Run("returns false with no groups", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.False(t, middleware.HasAnyGroup(c, "team-a"))
	})
}

func TestRequireGroup(t *testing.T) {
	t.Run("allows access when user is in group", func(t *testing.T) {
		e := echo.New()

		claims := &middleware.TokenClaims{
			UserID:   uuid.NewUUID(),
			Username: "testuser",
			Groups:   []string{"/team-a", "/team-b"},
		}

		validator := &mockTokenValidator{claims: claims}
		config := middleware.AuthConfig{TokenValidator: validator}

		e.Use(middleware.Auth(config))

		protected := e.Group("")
		protected.Use(middleware.RequireGroup("team-a"))
		protected.GET("/protected", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("denies access when user is not in group", func(t *testing.T) {
		e := echo.New()

		claims := &middleware.TokenClaims{
			UserID:   uuid.NewUUID(),
			Username: "testuser",
			Groups:   []string{"/team-a"},
		}

		validator := &mockTokenValidator{claims: claims}
		config := middleware.AuthConfig{TokenValidator: validator}

		e.Use(middleware.Auth(config))

		protected := e.Group("")
		protected.Use(middleware.RequireGroup("team-x"))
		protected.GET("/protected", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "FORBIDDEN")
	})
}

func TestRequireAnyGroup(t *testing.T) {
	t.Run("allows access when user is in one of the groups", func(t *testing.T) {
		e := echo.New()

		claims := &middleware.TokenClaims{
			UserID:   uuid.NewUUID(),
			Username: "testuser",
			Groups:   []string{"/team-b"},
		}

		validator := &mockTokenValidator{claims: claims}
		config := middleware.AuthConfig{TokenValidator: validator}

		e.Use(middleware.Auth(config))

		protected := e.Group("")
		protected.Use(middleware.RequireAnyGroup("team-a", "team-b", "team-c"))
		protected.GET("/protected", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("denies access when user is not in any of the groups", func(t *testing.T) {
		e := echo.New()

		claims := &middleware.TokenClaims{
			UserID:   uuid.NewUUID(),
			Username: "testuser",
			Groups:   []string{"/team-x"},
		}

		validator := &mockTokenValidator{claims: claims}
		config := middleware.AuthConfig{TokenValidator: validator}

		e.Use(middleware.Auth(config))

		protected := e.Group("")
		protected.Use(middleware.RequireAnyGroup("team-a", "team-b", "team-c"))
		protected.GET("/protected", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestAuth_ContextEnrichment_WithGroups(t *testing.T) {
	e := echo.New()

	claims := &middleware.TokenClaims{
		UserID:         uuid.NewUUID(),
		ExternalUserID: "ext-789",
		Username:       "groupuser",
		Email:          "group@example.com",
		Roles:          []string{"user"},
		Groups:         []string{"/team-alpha", "/team-beta"},
		IsSystemAdmin:  false,
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	validator := &mockTokenValidator{claims: claims}
	config := middleware.AuthConfig{TokenValidator: validator}

	var extractedUser *middleware.TokenClaims
	var extractedGroups []string

	e.Use(middleware.Auth(config))
	e.GET("/test", func(c echo.Context) error {
		extractedUser = middleware.GetUser(c)
		extractedGroups = middleware.GetGroups(c)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify GetUser returns full claims
	require.NotNil(t, extractedUser)
	assert.Equal(t, claims.UserID, extractedUser.UserID)
	assert.Equal(t, claims.ExternalUserID, extractedUser.ExternalUserID)
	assert.Equal(t, claims.Username, extractedUser.Username)
	assert.Equal(t, claims.Groups, extractedUser.Groups)

	// Verify GetGroups returns groups
	assert.Equal(t, claims.Groups, extractedGroups)

	// Verify InGroup works
	e2 := echo.New()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)
	c2.Set("groups", claims.Groups)

	assert.True(t, middleware.InGroup(c2, "team-alpha"))
	assert.True(t, middleware.InGroup(c2, "/team-beta"))
	assert.False(t, middleware.InGroup(c2, "team-gamma"))
}
