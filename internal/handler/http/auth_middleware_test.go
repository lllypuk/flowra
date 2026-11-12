package httphandler_test

import (
	"crypto/sha256"
	"encoding/hex"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

const sessionCookie = "flowra_session"

func ownerValidator(t *testing.T, secret string) *middleware.OwnerTokenValidator {
	t.Helper()
	sum := sha256.Sum256([]byte(secret))
	validator, err := middleware.NewOwnerTokenValidator(
		[]middleware.OwnerToken{{Name: "laptop", SHA256: hex.EncodeToString(sum[:])}},
		middleware.OwnerIdentity{Username: "owner", Email: "owner@flowra.local"},
		nil,
	)
	require.NoError(t, err)
	return validator
}

// usePageAuth swaps the package-level page auth config for one test.
func usePageAuth(t *testing.T, cfg *httphandler.PageAuthConfig) {
	t.Helper()
	httphandler.SetPageAuthConfig(cfg)
	t.Cleanup(func() { httphandler.SetPageAuthConfig(nil) })
}

func requestPage(cookieValue string) *httptest.ResponseRecorder {
	e := echo.New()
	req := httptest.NewRequest(stdhttp.MethodGet, "/workspaces", nil)
	if cookieValue != "" {
		req.AddCookie(&stdhttp.Cookie{Name: sessionCookie, Value: cookieValue})
	}
	rec := httptest.NewRecorder()

	handler := httphandler.RequireAuth(func(c echo.Context) error {
		return c.String(stdhttp.StatusOK, middleware.GetUsername(c))
	})
	_ = handler(e.NewContext(req, rec))
	return rec
}

func TestRequireAuth_OwnerToken(t *testing.T) {
	usePageAuth(t, &httphandler.PageAuthConfig{
		TokenValidator:   ownerValidator(t, "laptop-secret"),
		RequireValidator: true,
	})

	tests := []struct {
		name       string
		cookie     string
		wantStatus int
	}{
		{name: "owner token", cookie: "laptop-secret", wantStatus: stdhttp.StatusOK},
		{name: "mock session cookie", cookie: "mock-session-token", wantStatus: stdhttp.StatusFound},
		{name: "development token", cookie: "dev-token-1", wantStatus: stdhttp.StatusFound},
		{name: "no cookie", cookie: "", wantStatus: stdhttp.StatusFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := requestPage(tt.cookie)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == stdhttp.StatusOK {
				assert.Equal(t, "owner", rec.Body.String())
				return
			}
			assert.Equal(t, "/login", rec.Header().Get("Location"))
		})
	}
}

func TestRequireAuth_MissingValidator(t *testing.T) {
	t.Run("required validator rejects instead of using the mock user", func(t *testing.T) {
		usePageAuth(t, &httphandler.PageAuthConfig{RequireValidator: true})

		rec := requestPage("mock-session-token")

		assert.Equal(t, stdhttp.StatusFound, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("development default keeps the mock user", func(t *testing.T) {
		usePageAuth(t, &httphandler.PageAuthConfig{})

		rec := requestPage("mock-session-token")

		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})
}

func TestDisabledAuthService_RefusesOAuthFlows(t *testing.T) {
	svc := httphandler.NewDisabledAuthService()
	c := echo.New().NewContext(httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", nil), httptest.NewRecorder())

	login, err := svc.Login(c, "mock-code", "http://127.0.0.1:8080/auth/callback")
	require.ErrorIs(t, err, httphandler.ErrInvalidCredentials)
	assert.Nil(t, login)

	refreshed, err := svc.RefreshToken(c, "refresh")
	require.ErrorIs(t, err, httphandler.ErrRefreshTokenInvalid)
	assert.Nil(t, refreshed)
}
