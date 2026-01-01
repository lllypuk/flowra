package httphandler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/application/appcore"
	userapp "github.com/lllypuk/flowra/internal/application/user"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants for user handler.
const (
	maxDisplayNameLength = 100
	maxAvatarURLLength   = 500
)

// User handler errors.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDisplayNameEmpty   = errors.New("display name cannot be empty")
	ErrDisplayNameTooLong = errors.New("display name is too long")
	ErrInvalidAvatarURL   = errors.New("invalid avatar URL")
	ErrEmailInvalid       = errors.New("invalid email format")
)

// UpdateProfileRequest represents the request to update user profile.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	AvatarURL   *string `json:"avatar_url"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	IsAdmin     bool   `json:"is_admin"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// UserService defines the interface for user operations.
// Declared on the consumer side per project guidelines.
type UserService interface {
	// GetUser gets a user by ID.
	GetUser(ctx context.Context, query userapp.GetUserQuery) (userapp.Result, error)

	// GetUserByUsername gets a user by username.
	GetUserByUsername(ctx context.Context, query userapp.GetUserByUsernameQuery) (userapp.Result, error)

	// UpdateProfile updates a user's profile.
	UpdateProfile(ctx context.Context, cmd userapp.UpdateProfileCommand) (userapp.Result, error)
}

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userService UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterRoutes registers user routes with the router.
func (h *UserHandler) RegisterRoutes(r *httpserver.Router) {
	// Current user operations
	r.Auth().GET("/users/me", h.GetMe)
	r.Auth().PUT("/users/me", h.UpdateMe)

	// Get other users (authenticated)
	r.Auth().GET("/users/:id", h.Get)
}

// GetMe handles GET /api/v1/users/me.
// Gets the current authenticated user.
func (h *UserHandler) GetMe(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	query := userapp.GetUserQuery{
		UserID: userID,
	}

	result, err := h.userService.GetUser(c.Request().Context(), query)
	if err != nil {
		return handleUserError(c, err)
	}

	resp := ToUserResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// UpdateMe handles PUT /api/v1/users/me.
// Updates the current authenticated user's profile.
func (h *UserHandler) UpdateMe(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	var req UpdateProfileRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate request
	if valErr := validateUpdateProfileRequest(&req); valErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error())
	}

	cmd := userapp.UpdateProfileCommand{
		UserID:      userID,
		DisplayName: req.DisplayName,
		Email:       req.Email,
	}

	result, err := h.userService.UpdateProfile(c.Request().Context(), cmd)
	if err != nil {
		return handleUserError(c, err)
	}

	resp := ToUserResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// Get handles GET /api/v1/users/:id.
// Gets a user by ID.
func (h *UserHandler) Get(c echo.Context) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	userIDStr := c.Param("id")
	userID, parseErr := uuid.ParseUUID(userIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_USER_ID", "invalid user ID format")
	}

	query := userapp.GetUserQuery{
		UserID: userID,
	}

	result, err := h.userService.GetUser(c.Request().Context(), query)
	if err != nil {
		return handleUserError(c, err)
	}

	resp := ToUserResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// Helper functions

func validateUpdateProfileRequest(req *UpdateProfileRequest) error {
	// At least one field must be provided
	if req.DisplayName == nil && req.Email == nil && req.AvatarURL == nil {
		return errors.New("at least one field must be provided")
	}

	if req.DisplayName != nil {
		if *req.DisplayName == "" {
			return ErrDisplayNameEmpty
		}
		if len(*req.DisplayName) > maxDisplayNameLength {
			return ErrDisplayNameTooLong
		}
	}

	if req.AvatarURL != nil && len(*req.AvatarURL) > maxAvatarURLLength {
		return ErrInvalidAvatarURL
	}

	return nil
}

func handleUserError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, userapp.ErrUserNotFound):
		return httpserver.RespondErrorWithCode(c, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	case errors.Is(err, userapp.ErrEmailAlreadyExists):
		return httpserver.RespondErrorWithCode(
			c, http.StatusConflict, "EMAIL_EXISTS", "email is already in use")
	case errors.Is(err, userapp.ErrUsernameAlreadyExists):
		return httpserver.RespondErrorWithCode(
			c, http.StatusConflict, "USERNAME_EXISTS", "username is already in use")
	case errors.Is(err, userapp.ErrInvalidEmail):
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_EMAIL", "invalid email format")
	case errors.Is(err, userapp.ErrInvalidUsername):
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_USERNAME", "invalid username format")
	default:
		return httpserver.RespondError(c, err)
	}
}

// ToUserResponse converts a domain User to UserResponse.
func ToUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:          u.ID().String(),
		Username:    u.Username(),
		Email:       u.Email(),
		DisplayName: u.DisplayName(),
		IsAdmin:     u.IsSystemAdmin(),
		CreatedAt:   u.CreatedAt().Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt().Format(time.RFC3339),
	}
}

// MockUserService is a mock implementation of UserService for testing.
type MockUserService struct {
	users        map[uuid.UUID]*user.User
	usersByName  map[string]*user.User
	usersByEmail map[string]*user.User
}

// NewMockUserService creates a new mock user service.
func NewMockUserService() *MockUserService {
	return &MockUserService{
		users:        make(map[uuid.UUID]*user.User),
		usersByName:  make(map[string]*user.User),
		usersByEmail: make(map[string]*user.User),
	}
}

// AddUser adds a user to the mock service.
func (m *MockUserService) AddUser(u *user.User) {
	m.users[u.ID()] = u
	m.usersByName[u.Username()] = u
	m.usersByEmail[u.Email()] = u
}

// GetUser gets a user from the mock service.
func (m *MockUserService) GetUser(
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
func (m *MockUserService) GetUserByUsername(
	_ context.Context,
	query userapp.GetUserByUsernameQuery,
) (userapp.Result, error) {
	u, ok := m.usersByName[query.Username]
	if !ok {
		return userapp.Result{}, userapp.ErrUserNotFound
	}

	return userapp.Result{
		Result: appcore.Result[*user.User]{Value: u},
	}, nil
}

// UpdateProfile updates a user's profile in the mock service.
func (m *MockUserService) UpdateProfile(
	_ context.Context,
	cmd userapp.UpdateProfileCommand,
) (userapp.Result, error) {
	u, ok := m.users[cmd.UserID]
	if !ok {
		return userapp.Result{}, userapp.ErrUserNotFound
	}

	// Check email uniqueness if email is being updated
	if cmd.Email != nil {
		existing, emailExists := m.usersByEmail[*cmd.Email]
		if emailExists && existing.ID() != u.ID() {
			return userapp.Result{}, userapp.ErrEmailAlreadyExists
		}
	}

	// Update profile
	if err := u.UpdateProfile(cmd.DisplayName, cmd.Email); err != nil {
		return userapp.Result{}, err
	}

	// Update email index if changed
	if cmd.Email != nil {
		delete(m.usersByEmail, u.Email())
		m.usersByEmail[*cmd.Email] = u
	}

	return userapp.Result{
		Result: appcore.Result[*user.User]{Value: u},
	}, nil
}
