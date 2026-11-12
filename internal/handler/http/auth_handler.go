package httphandler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// UserDTO represents user data in API responses.
type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
}

// UserRepository defines the interface for user data access.
// Declared on the consumer side per project guidelines.
type UserRepository interface {
	// FindByID finds a user by their internal ID.
	FindByID(ctx echo.Context, id uuid.UUID) (*user.User, error)
}

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	userRepo UserRepository
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userRepo UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

// RegisterRoutes registers auth routes with the router.
func (h *AuthHandler) RegisterRoutes(r *httpserver.Router) {
	r.Auth().GET("/auth/me", h.Me)
}

// Me handles GET /api/v1/auth/me.
// Returns the current authenticated user's information.
func (h *AuthHandler) Me(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	usr, err := h.userRepo.FindByID(c, userID)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusNotFound,
			"USER_NOT_FOUND",
			"User not found",
		)
	}

	return httpserver.RespondOK(c, ToUserDTO(usr))
}

// ToUserDTO converts a domain User to a UserDTO.
func ToUserDTO(u *user.User) UserDTO {
	return UserDTO{
		ID:          u.ID(),
		Email:       u.Email(),
		Username:    u.Username(),
		DisplayName: u.DisplayName(),
		// AvatarURL is left empty as it's not in the domain model yet
	}
}

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	users map[uuid.UUID]*user.User
}

// NewMockUserRepository creates a new mock user repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[uuid.UUID]*user.User),
	}
}

// AddUser adds a user to the mock repository.
func (m *MockUserRepository) AddUser(u *user.User) {
	m.users[u.ID()] = u
}

// FindByID implements UserRepository.
func (m *MockUserRepository) FindByID(_ echo.Context, id uuid.UUID) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}
