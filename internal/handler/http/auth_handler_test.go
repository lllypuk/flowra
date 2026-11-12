package httphandler_test

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
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

func TestAuthHandler_Me(t *testing.T) {
	t.Run("successful get current user", func(t *testing.T) {
		e := echo.New()
		testUser := createTestUser(t)

		mockUserRepo := httphandler.NewMockUserRepository()
		mockUserRepo.AddUser(testUser)

		handler := httphandler.NewAuthHandler(mockUserRepo)

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

		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockUserRepo)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Me(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		e := echo.New()

		mockUserRepo := httphandler.NewMockUserRepository()

		handler := httphandler.NewAuthHandler(mockUserRepo)

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

func TestToUserDTO(t *testing.T) {
	testUser := createTestUser(t)

	dto := httphandler.ToUserDTO(testUser)

	assert.Equal(t, testUser.ID(), dto.ID)
	assert.Equal(t, testUser.Email(), dto.Email)
	assert.Equal(t, testUser.Username(), dto.Username)
	assert.Equal(t, testUser.DisplayName(), dto.DisplayName)
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
}

func TestNewAuthHandler(t *testing.T) {
	mockUserRepo := httphandler.NewMockUserRepository()

	handler := httphandler.NewAuthHandler(mockUserRepo)

	assert.NotNil(t, handler)
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	mockUserRepo := httphandler.NewMockUserRepository()

	handler := httphandler.NewAuthHandler(mockUserRepo)

	e := echo.New()
	router := httpserver.NewRouter(e, httpserver.DefaultRouterConfig())

	// Should not panic
	handler.RegisterRoutes(router)

	// Verify routes are registered
	routes := e.Routes()
	assert.NotEmpty(t, routes)
}
