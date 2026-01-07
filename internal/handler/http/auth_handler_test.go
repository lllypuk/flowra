package httphandler_test

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httphandler "github.com/lllypuk/flowra/internal/handler/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test user.
func createTestUser(t *testing.T) *user.User {
	t.Helper()
	u, err := user.NewUser("external-123", "testuser", "test@example.com", "Test User")
	require.NoError(t, err)
	return u
}

// Helper function to set up auth context.
func setupAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockAuthService := httphandler.NewMockAuthService()
		mockAuthService.AddUser("valid-code", testUser)
		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"code": "valid-code", "redirect_uri": "http://localhost:3000/callback"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)
	})

	t.Run("missing code", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"redirect_uri": "http://localhost:3000/callback"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	})

	t.Run("missing redirect_uri", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"code": "valid-code"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	})

	t.Run("invalid code", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"code": "invalid-code", "redirect_uri": "http://localhost:3000/callback"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "INVALID_CREDENTIALS", resp.Error.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"code": invalid-json}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("successful logout", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		setupAuthContext(c, testUser.ID())

		err := handler.Logout(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("unauthorized - no user in context", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Logout(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
	})
}

func TestAuthHandler_Me(t *testing.T) {
	t.Run("successful get current user", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		setupAuthContext(c, testUser.ID())

		err := handler.Me(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Check that user data is in the response
		dataMap, ok := resp.Data.(map[string]any)
		require.True(t, ok, "data should be a map")
		assert.Equal(t, testUser.Email(), dataMap["email"])
		assert.Equal(t, testUser.Username(), dataMap["username"])
	})

	t.Run("unauthorized - no user in context", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Me(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context with non-existent user
		setupAuthContext(c, uuid.NewUUID())

		err := handler.Me(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "USER_NOT_FOUND", resp.Error.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"refresh_token": "valid-refresh-token"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]any)
		require.True(t, ok, "data should be a map")
		assert.NotEmpty(t, dataMap["access_token"])
		assert.NotEmpty(t, dataMap["refresh_token"])
	})

	t.Run("missing refresh token", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	})

	t.Run("empty refresh token", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"refresh_token": ""}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		e := echo.New()

		mockAuthService := httphandler.NewMockAuthService()
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `not valid json`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestToUserDTO(t *testing.T) {
	testUser := createTestUser(t)

	dto := httphandler.ToUserDTO(testUser)

	assert.Equal(t, testUser.ID(), dto.ID)
	assert.Equal(t, testUser.Email(), dto.Email)
	assert.Equal(t, testUser.Username(), dto.Username)
	assert.Equal(t, testUser.DisplayName(), dto.DisplayName)
}

func TestMockAuthService(t *testing.T) {
	t.Run("login with existing user", func(t *testing.T) {
		mockService := httphandler.NewMockAuthService()
		testUser := createTestUser(t)
		mockService.AddUser("test-code", testUser)

		result, err := mockService.Login(nil, "test-code", "http://localhost")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testUser, result.User)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
	})

	t.Run("login with invalid code", func(t *testing.T) {
		mockService := httphandler.NewMockAuthService()

		result, err := mockService.Login(nil, "invalid-code", "http://localhost")
		require.ErrorIs(t, err, httphandler.ErrInvalidCredentials)
		assert.Nil(t, result)
	})

	t.Run("logout always succeeds", func(t *testing.T) {
		mockService := httphandler.NewMockAuthService()

		err := mockService.Logout(nil, uuid.NewUUID())
		assert.NoError(t, err)
	})

	t.Run("refresh with valid token", func(t *testing.T) {
		mockService := httphandler.NewMockAuthService()

		result, err := mockService.RefreshToken(nil, "some-refresh-token")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
	})

	t.Run("refresh with empty token", func(t *testing.T) {
		mockService := httphandler.NewMockAuthService()

		result, err := mockService.RefreshToken(nil, "")
		require.ErrorIs(t, err, httphandler.ErrRefreshTokenInvalid)
		assert.Nil(t, result)
	})
}

func TestMockUserRepository(t *testing.T) {
	t.Run("find by ID", func(t *testing.T) {
		mockRepo := httphandler.NewMockUserRepository()
		testUser := createTestUser(t)
		mockRepo.AddUser(testUser)

		found, err := mockRepo.FindByID(nil, testUser.ID())
		require.NoError(t, err)
		assert.Equal(t, testUser, found)
	})

	t.Run("find by ID not found", func(t *testing.T) {
		mockRepo := httphandler.NewMockUserRepository()

		found, err := mockRepo.FindByID(nil, uuid.NewUUID())
		require.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("find by external ID", func(t *testing.T) {
		mockRepo := httphandler.NewMockUserRepository()
		testUser := createTestUser(t)
		mockRepo.AddUser(testUser)

		found, err := mockRepo.FindByExternalID(nil, testUser.ExternalID())
		require.NoError(t, err)
		assert.Equal(t, testUser, found)
	})

	t.Run("find by external ID not found", func(t *testing.T) {
		mockRepo := httphandler.NewMockUserRepository()

		found, err := mockRepo.FindByExternalID(nil, "non-existent")
		require.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestNewAuthHandler(t *testing.T) {
	mockAuthService := httphandler.NewMockAuthService()
	mockUserRepo := httphandler.NewMockUserRepository()

	handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

	assert.NotNil(t, handler)
}

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	t.Run("internal service error", func(t *testing.T) {
		e := echo.New()

		// Create a custom mock that returns a non-credential error
		mockAuthService := &errorAuthService{}
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"code": "valid-code", "redirect_uri": "http://localhost:3000/callback"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusInternalServerError, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "LOGIN_FAILED", resp.Error.Code)
	})
}

// errorAuthService is a mock that returns generic errors
type errorAuthService struct{}

func (e *errorAuthService) Login(_ echo.Context, _, _ string) (*httphandler.LoginResult, error) {
	return nil, errors.New("internal error")
}

func (e *errorAuthService) Logout(_ echo.Context, _ uuid.UUID) error {
	return errors.New("logout failed")
}

func (e *errorAuthService) RefreshToken(_ echo.Context, _ string) (*httphandler.RefreshResult, error) {
	return nil, errors.New("refresh failed")
}

func TestAuthHandler_Logout_ServiceError(t *testing.T) {
	t.Run("service error during logout", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockAuthService := &errorAuthService{}
		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupAuthContext(c, testUser.ID())

		err := handler.Logout(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusInternalServerError, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "LOGOUT_FAILED", resp.Error.Code)
	})
}

func TestAuthHandler_Refresh_ServiceError(t *testing.T) {
	t.Run("service error during refresh", func(t *testing.T) {
		e := echo.New()

		mockAuthService := &errorAuthService{}
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"refresh_token": "valid-token"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusInternalServerError, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "REFRESH_FAILED", resp.Error.Code)
	})
}

// sessionNotFoundAuthService is a mock that returns session not found error
type sessionNotFoundAuthService struct{}

func (s *sessionNotFoundAuthService) Login(_ echo.Context, _, _ string) (*httphandler.LoginResult, error) {
	return nil, httphandler.ErrInvalidCredentials
}

func (s *sessionNotFoundAuthService) Logout(_ echo.Context, _ uuid.UUID) error {
	return httphandler.ErrSessionNotFound
}

func (s *sessionNotFoundAuthService) RefreshToken(_ echo.Context, _ string) (*httphandler.RefreshResult, error) {
	return nil, httphandler.ErrRefreshTokenInvalid
}

func TestAuthHandler_Logout_SessionNotFound(t *testing.T) {
	t.Run("session not found still succeeds", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockAuthService := &sessionNotFoundAuthService{}
		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupAuthContext(c, testUser.ID())

		err := handler.Logout(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	t.Run("invalid refresh token error", func(t *testing.T) {
		e := echo.New()

		mockAuthService := &sessionNotFoundAuthService{}
		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

		reqBody := `{"refresh_token": "invalid-token"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/refresh", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Refresh(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REFRESH_TOKEN", resp.Error.Code)
	})
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	mockAuthService := httphandler.NewMockAuthService()
	mockUserRepo := httphandler.NewMockUserRepository()

	handler := httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

	e := echo.New()
	router := httpserver.NewRouter(e, httpserver.DefaultRouterConfig())

	// Should not panic
	handler.RegisterRoutes(router)

	// Verify routes are registered
	routes := e.Routes()
	assert.NotEmpty(t, routes)
}

func TestErrors(t *testing.T) {
	t.Run("error variables are defined", func(t *testing.T) {
		require.Error(t, httphandler.ErrInvalidCredentials)
		require.Error(t, httphandler.ErrSessionNotFound)
		require.Error(t, httphandler.ErrRefreshTokenRequired)
		require.Error(t, httphandler.ErrRefreshTokenInvalid)
	})
}
