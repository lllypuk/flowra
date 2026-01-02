package httphandler_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/application/appcore"
	userapp "github.com/lllypuk/flowra/internal/application/user"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up user auth context.
func setupUserAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

// Helper function to create a test user for user handler tests.
func createTestUserForUserHandler(t *testing.T) *user.User {
	t.Helper()
	u, err := user.NewUser(
		"ext-"+uuid.NewUUID().String(),
		"testuser",
		"test@example.com",
		"Test User",
	)
	require.NoError(t, err)
	return u
}

func TestUserHandler_GetMe(t *testing.T) {
	t.Run("successful get me", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.GetMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		mockService := httphandler.NewMockUserService()
		handler := httphandler.NewUserHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockUserService()
		handler := httphandler.NewUserHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, userID)

		err := handler.GetMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})
}

func TestUserHandler_UpdateMe(t *testing.T) {
	t.Run("successful update display name", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		reqBody := `{"display_name": "New Display Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("successful update email", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		reqBody := `{"email": "newemail@example.com"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		mockService := httphandler.NewMockUserService()
		handler := httphandler.NewUserHandler(mockService)

		reqBody := `{"display_name": "New Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("empty request body", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		reqBody := `{}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("empty display name", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		reqBody := `{"display_name": ""}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("display name too long", func(t *testing.T) {
		e := echo.New()

		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		handler := httphandler.NewUserHandler(mockService)

		longName := strings.Repeat("a", 150)
		reqBody := `{"display_name": "` + longName + `"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/users/me", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupUserAuthContext(c, testUser.ID())

		err := handler.UpdateMe(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestUserHandler_Get(t *testing.T) {
	t.Run("successful get user by ID", func(t *testing.T) {
		e := echo.New()

		currentUser := createTestUserForUserHandler(t)
		targetUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(currentUser)
		mockService.AddUser(targetUser)
		handler := httphandler.NewUserHandler(mockService)

		url := "/api/v1/users/" + targetUser.ID().String()
		req := httptest.NewRequest(stdhttp.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(targetUser.ID().String())

		setupUserAuthContext(c, currentUser.ID())

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		e := echo.New()

		currentUser := createTestUserForUserHandler(t)
		nonExistentID := uuid.NewUUID()
		mockService := NewMockUserServiceWithUser(currentUser)
		handler := httphandler.NewUserHandler(mockService)

		url := "/api/v1/users/" + nonExistentID.String()
		req := httptest.NewRequest(stdhttp.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentID.String())

		setupUserAuthContext(c, currentUser.ID())

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("invalid user ID format", func(t *testing.T) {
		e := echo.New()

		currentUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(currentUser)
		handler := httphandler.NewUserHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/users/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")

		setupUserAuthContext(c, currentUser.ID())

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		userID := uuid.NewUUID()
		mockService := httphandler.NewMockUserService()
		handler := httphandler.NewUserHandler(mockService)

		url := "/api/v1/users/" + userID.String()
		req := httptest.NewRequest(stdhttp.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(userID.String())

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNewUserHandler(t *testing.T) {
	mockService := httphandler.NewMockUserService()
	handler := httphandler.NewUserHandler(mockService)
	assert.NotNil(t, handler)
}

func TestToUserResponse(t *testing.T) {
	u := user.Reconstruct(
		uuid.NewUUID(),
		"ext-123",
		"testuser",
		"test@example.com",
		"Test User Display",
		true,
		time.Now().Add(-24*time.Hour),
		time.Now(),
	)

	resp := httphandler.ToUserResponse(u)

	assert.Equal(t, u.ID().String(), resp.ID)
	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User Display", resp.DisplayName)
	assert.True(t, resp.IsAdmin)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

func TestMockUserService(t *testing.T) {
	t.Run("add and get user", func(t *testing.T) {
		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)
		mockService.AddUser(testUser)

		query := userapp.GetUserQuery{
			UserID: testUser.ID(),
		}

		_, err := mockService.GetUser(context.Background(), query)
		require.NoError(t, err)
	})

	t.Run("get non-existent user", func(t *testing.T) {
		mockService := httphandler.NewMockUserService()

		query := userapp.GetUserQuery{
			UserID: uuid.NewUUID(),
		}

		_, err := mockService.GetUser(context.Background(), query)
		assert.ErrorIs(t, err, userapp.ErrUserNotFound)
	})

	t.Run("update profile", func(t *testing.T) {
		testUser := createTestUserForUserHandler(t)
		mockService := NewMockUserServiceWithUser(testUser)

		newName := "Updated Name"
		cmd := userapp.UpdateProfileCommand{
			UserID:      testUser.ID(),
			DisplayName: &newName,
		}

		_, err := mockService.UpdateProfile(context.Background(), cmd)
		require.NoError(t, err)
	})

	t.Run("update profile - user not found", func(t *testing.T) {
		mockService := httphandler.NewMockUserService()

		newName := "Updated Name"
		cmd := userapp.UpdateProfileCommand{
			UserID:      uuid.NewUUID(),
			DisplayName: &newName,
		}

		_, err := mockService.UpdateProfile(context.Background(), cmd)
		assert.ErrorIs(t, err, userapp.ErrUserNotFound)
	})
}

// MockUserServiceWithUser is a wrapper for testing that stores users.
type MockUserServiceWithUser struct {
	users        map[uuid.UUID]*user.User
	usersByEmail map[string]*user.User
}

// NewMockUserServiceWithUser creates a mock user service with the given user.
func NewMockUserServiceWithUser(u *user.User) *MockUserServiceWithUser {
	mock := &MockUserServiceWithUser{
		users:        make(map[uuid.UUID]*user.User),
		usersByEmail: make(map[string]*user.User),
	}
	mock.AddUser(u)
	return mock
}

// AddUser adds a user to the mock service.
func (m *MockUserServiceWithUser) AddUser(u *user.User) {
	m.users[u.ID()] = u
	m.usersByEmail[u.Email()] = u
}

// GetUser gets a user from the mock service.
func (m *MockUserServiceWithUser) GetUser(
	_ context.Context,
	query userapp.GetUserQuery,
) (userapp.Result, error) {
	u, ok := m.users[query.UserID]
	if !ok {
		return userapp.Result{}, userapp.ErrUserNotFound
	}

	return userapp.Result{
		Result: appcore.Result[*user.User]{Value: u},
	}, nil
}

// GetUserByUsername gets a user by username from the mock service.
func (m *MockUserServiceWithUser) GetUserByUsername(
	_ context.Context,
	query userapp.GetUserByUsernameQuery,
) (userapp.Result, error) {
	for _, u := range m.users {
		if u.Username() == query.Username {
			return userapp.Result{
				Result: appcore.Result[*user.User]{Value: u},
			}, nil
		}
	}
	return userapp.Result{}, userapp.ErrUserNotFound
}

// UpdateProfile updates a user's profile in the mock service.
func (m *MockUserServiceWithUser) UpdateProfile(
	_ context.Context,
	cmd userapp.UpdateProfileCommand,
) (userapp.Result, error) {
	u, ok := m.users[cmd.UserID]
	if !ok {
		return userapp.Result{}, userapp.ErrUserNotFound
	}

	// Check email uniqueness if email is being updated
	if cmd.Email != nil {
		for _, existing := range m.usersByEmail {
			if existing.ID() != u.ID() && existing.Email() == *cmd.Email {
				return userapp.Result{}, userapp.ErrEmailAlreadyExists
			}
		}
	}

	// Update profile
	if err := u.UpdateProfile(cmd.DisplayName, cmd.Email); err != nil {
		return userapp.Result{}, err
	}

	return userapp.Result{
		Result: appcore.Result[*user.User]{Value: u},
	}, nil
}
