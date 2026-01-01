package httphandler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Auth handler errors.
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrSessionNotFound      = errors.New("session not found")
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	ErrRefreshTokenInvalid  = errors.New("refresh token is invalid or expired")
)

// LoginRequest represents the login request body.
type LoginRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// LoginResponse represents the login response.
type LoginResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    int     `json:"expires_in"`
	User         UserDTO `json:"user"`
}

// UserDTO represents user data in API responses.
type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
}

// RefreshRequest represents the refresh token request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents the refresh token response.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// AuthService defines the interface for authentication operations.
// Declared on the consumer side per project guidelines.
type AuthService interface {
	// Login validates OAuth code and returns tokens.
	Login(ctx echo.Context, code, redirectURI string) (*LoginResult, error)

	// Logout invalidates the current session.
	Logout(ctx echo.Context, userID uuid.UUID) error

	// RefreshToken refreshes the access token using a refresh token.
	RefreshToken(ctx echo.Context, refreshToken string) (*RefreshResult, error)
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *user.User
}

// RefreshResult contains the result of a token refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// UserRepository defines the interface for user data access.
// Declared on the consumer side per project guidelines.
type UserRepository interface {
	// FindByID finds a user by their internal ID.
	FindByID(ctx echo.Context, id uuid.UUID) (*user.User, error)

	// FindByExternalID finds a user by their external (Keycloak) ID.
	FindByExternalID(ctx echo.Context, externalID string) (*user.User, error)
}

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService AuthService
	userRepo    UserRepository
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService AuthService, userRepo UserRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

// RegisterRoutes registers auth routes with the router.
func (h *AuthHandler) RegisterRoutes(r *httpserver.Router) {
	// Public routes (no auth required)
	r.Public().POST("/auth/login", h.Login)

	// Authenticated routes
	r.Auth().POST("/auth/logout", h.Logout)
	r.Auth().GET("/auth/me", h.Me)
	r.Auth().POST("/auth/refresh", h.Refresh)
}

// Login handles POST /api/v1/auth/login.
// Validates OAuth code and returns access + refresh tokens.
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	// Validate required fields
	if req.Code == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"OAuth code is required",
		)
	}

	if req.RedirectURI == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Redirect URI is required",
		)
	}

	result, err := h.authService.Login(c, req.Code, req.RedirectURI)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"Invalid OAuth code or credentials",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"LOGIN_FAILED",
			"Failed to complete login",
		)
	}

	return httpserver.RespondOK(c, LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		User:         ToUserDTO(result.User),
	})
}

// Logout handles POST /api/v1/auth/logout.
// Invalidates the current session and refresh token.
func (h *AuthHandler) Logout(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	if err := h.authService.Logout(c, userID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// Session already invalidated, consider it a success
			return httpserver.RespondOK(c, map[string]string{
				"message": "Logged out successfully",
			})
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"LOGOUT_FAILED",
			"Failed to complete logout",
		)
	}

	return httpserver.RespondOK(c, map[string]string{
		"message": "Logged out successfully",
	})
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

// Refresh handles POST /api/v1/auth/refresh.
// Refreshes the access token using a valid refresh token.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	if req.RefreshToken == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Refresh token is required",
		)
	}

	result, err := h.authService.RefreshToken(c, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenInvalid) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusUnauthorized,
				"INVALID_REFRESH_TOKEN",
				"Refresh token is invalid or expired",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"REFRESH_FAILED",
			"Failed to refresh token",
		)
	}

	return httpserver.RespondOK(c, RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
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

// MockAuthService is a mock implementation of AuthService for testing and development.
type MockAuthService struct {
	users map[string]*user.User // code -> user
}

// NewMockAuthService creates a new mock auth service.
func NewMockAuthService() *MockAuthService {
	return &MockAuthService{
		users: make(map[string]*user.User),
	}
}

// AddUser adds a user that can be logged in with the given code.
func (m *MockAuthService) AddUser(code string, u *user.User) {
	m.users[code] = u
}

// Login implements AuthService.
func (m *MockAuthService) Login(_ echo.Context, code, _ string) (*LoginResult, error) {
	u, ok := m.users[code]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	const defaultExpiresIn = 3600

	return &LoginResult{
		AccessToken:  "mock-access-token-" + u.ID().String(),
		RefreshToken: "mock-refresh-token-" + u.ID().String(),
		ExpiresIn:    defaultExpiresIn,
		User:         u,
	}, nil
}

// Logout implements AuthService.
func (m *MockAuthService) Logout(_ echo.Context, _ uuid.UUID) error {
	return nil
}

// RefreshToken implements AuthService.
func (m *MockAuthService) RefreshToken(_ echo.Context, refreshToken string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, ErrRefreshTokenInvalid
	}

	const defaultExpiresIn = 3600

	return &RefreshResult{
		AccessToken:  "mock-access-token-refreshed-" + time.Now().Format(time.RFC3339),
		RefreshToken: "mock-refresh-token-refreshed-" + time.Now().Format(time.RFC3339),
		ExpiresIn:    defaultExpiresIn,
	}, nil
}

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	users           map[uuid.UUID]*user.User
	usersByExternal map[string]*user.User
}

// NewMockUserRepository creates a new mock user repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:           make(map[uuid.UUID]*user.User),
		usersByExternal: make(map[string]*user.User),
	}
}

// AddUser adds a user to the mock repository.
func (m *MockUserRepository) AddUser(u *user.User) {
	m.users[u.ID()] = u
	m.usersByExternal[u.ExternalID()] = u
}

// FindByID implements UserRepository.
func (m *MockUserRepository) FindByID(_ echo.Context, id uuid.UUID) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

// FindByExternalID implements UserRepository.
func (m *MockUserRepository) FindByExternalID(_ echo.Context, externalID string) (*user.User, error) {
	u, ok := m.usersByExternal[externalID]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}
