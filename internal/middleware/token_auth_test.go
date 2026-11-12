package middleware_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/middleware"
)

func digestOf(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func newOwnerValidator(t *testing.T, tokens ...middleware.OwnerToken) *middleware.OwnerTokenValidator {
	t.Helper()
	validator, err := middleware.NewOwnerTokenValidator(
		tokens,
		middleware.OwnerIdentity{Username: "owner", Email: "owner@flowra.local"},
		nil,
	)
	require.NoError(t, err)
	return validator
}

func TestNewOwnerTokenValidator_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		tokens []middleware.OwnerToken
	}{
		{name: "no tokens", tokens: nil},
		{name: "digest is not hex", tokens: []middleware.OwnerToken{{Name: "laptop", SHA256: "not-hex"}}},
		{name: "digest is too short", tokens: []middleware.OwnerToken{{Name: "laptop", SHA256: "abcd"}}},
		{name: "raw secret instead of digest", tokens: []middleware.OwnerToken{{Name: "laptop", SHA256: "secret"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := middleware.NewOwnerTokenValidator(tt.tokens, middleware.OwnerIdentity{}, nil)
			require.Error(t, err)
			assert.Nil(t, validator)
		})
	}
}

func TestOwnerTokenValidator_ValidateToken(t *testing.T) {
	validator := newOwnerValidator(t,
		middleware.OwnerToken{Name: "laptop", SHA256: digestOf("laptop-secret")},
		middleware.OwnerToken{Name: "phone", SHA256: digestOf("phone-secret")},
	)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{name: "first token", token: "laptop-secret"},
		{name: "second token", token: "phone-secret"},
		{name: "unknown token", token: "other-secret", wantErr: middleware.ErrInvalidToken},
		{name: "empty token", token: "", wantErr: middleware.ErrInvalidToken},
		{name: "digest presented as token", token: digestOf("laptop-secret"), wantErr: middleware.ErrInvalidToken},
		{name: "development token", token: "dev-token-123", wantErr: middleware.ErrInvalidToken},
		{name: "mock session cookie value", token: "mock-session-token", wantErr: middleware.ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(context.Background(), tt.token)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, middleware.OwnerExternalID, claims.ExternalUserID)
			assert.Equal(t, "owner", claims.Username)
			assert.Equal(t, "owner@flowra.local", claims.Email)
			assert.False(t, claims.IsSystemAdmin)
		})
	}
}

func TestOwnerTokenValidator_RotationKeepsIdentity(t *testing.T) {
	before := newOwnerValidator(t, middleware.OwnerToken{Name: "laptop", SHA256: digestOf("old-secret")})
	after := newOwnerValidator(t, middleware.OwnerToken{Name: "laptop", SHA256: digestOf("new-secret")})

	oldClaims, err := before.ValidateToken(context.Background(), "old-secret")
	require.NoError(t, err)
	newClaims, err := after.ValidateToken(context.Background(), "new-secret")
	require.NoError(t, err)
	assert.Equal(t, oldClaims.ExternalUserID, newClaims.ExternalUserID)

	_, err = after.ValidateToken(context.Background(), "old-secret")
	require.ErrorIs(t, err, middleware.ErrInvalidToken)
}

func TestAuth_OwnerToken_BearerAndCookie(t *testing.T) {
	validator := newOwnerValidator(t, middleware.OwnerToken{Name: "laptop", SHA256: digestOf("laptop-secret")})
	auth := middleware.Auth(middleware.AuthConfig{
		TokenValidator:    validator,
		SessionCookieName: "flowra_session",
	})
	handler := auth(func(c echo.Context) error {
		return c.String(http.StatusOK, middleware.GetUsername(c))
	})

	tests := []struct {
		name       string
		prepare    func(req *http.Request)
		wantStatus int
	}{
		{
			name:       "bearer token",
			prepare:    func(req *http.Request) { req.Header.Set("Authorization", "Bearer laptop-secret") },
			wantStatus: http.StatusOK,
		},
		{
			name: "session cookie",
			prepare: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "flowra_session", Value: "laptop-secret"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong bearer token",
			prepare:    func(req *http.Request) { req.Header.Set("Authorization", "Bearer dev-token-1") },
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "mock session cookie",
			prepare: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "flowra_session", Value: "mock-session-token"})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no credentials",
			prepare:    func(_ *http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
			tt.prepare(req)
			rec := httptest.NewRecorder()

			_ = handler(e.NewContext(req, rec))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, "owner", rec.Body.String())
			}
		})
	}
}
