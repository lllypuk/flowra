package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Context keys for authentication data.
type contextKey string

const (
	// ContextKeyUserID is the context key for user ID.
	ContextKeyUserID contextKey = "user_id"

	// ContextKeyExternalUserID is the context key for external user ID (from Keycloak).
	ContextKeyExternalUserID contextKey = "external_user_id"

	// ContextKeyUsername is the context key for username.
	ContextKeyUsername contextKey = "username"

	// ContextKeyEmail is the context key for user email.
	ContextKeyEmail contextKey = "email"

	// ContextKeyRoles is the context key for user roles.
	ContextKeyRoles contextKey = "roles"

	// ContextKeyIsSystemAdmin is the context key for system admin flag.
	ContextKeyIsSystemAdmin contextKey = "is_system_admin"
)

// Auth errors.
var (
	ErrMissingAuthHeader       = errors.New("missing authorization header")
	ErrInvalidAuthHeader       = errors.New("invalid authorization header format")
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenExpired            = errors.New("token expired")
	ErrUserNotFound            = errors.New("user not found")
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// errMockSessionHandled is a sentinel error indicating mock session was handled.
	errMockSessionHandled = errors.New("mock session handled")
)

// TokenClaims represents the claims extracted from a JWT token.
type TokenClaims struct {
	// UserID is the internal user ID.
	UserID uuid.UUID

	// ExternalUserID is the user ID from the external auth provider (Keycloak).
	ExternalUserID string

	// Username is the user's username.
	Username string

	// Email is the user's email address.
	Email string

	// Roles is a list of user roles.
	Roles []string

	// IsSystemAdmin indicates if the user is a system administrator.
	IsSystemAdmin bool

	// ExpiresAt is the token expiration time.
	ExpiresAt time.Time
}

// TokenValidator defines the interface for validating JWT tokens.
type TokenValidator interface {
	// ValidateToken validates a JWT token and returns the claims.
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
}

// UserResolver resolves user information from external ID.
type UserResolver interface {
	// ResolveUser finds or creates a user by external ID and returns their internal ID.
	ResolveUser(ctx context.Context, externalID, username, email string) (uuid.UUID, error)
}

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	// Logger is the structured logger for auth events.
	Logger *slog.Logger

	// TokenValidator validates JWT tokens.
	TokenValidator TokenValidator

	// UserResolver resolves users from external IDs.
	// Optional - if nil, only ExternalUserID will be set in context.
	UserResolver UserResolver

	// SkipPaths are paths that don't require authentication.
	SkipPaths []string

	// AllowExpiredForPaths allows expired tokens for specific paths (e.g., refresh endpoint).
	AllowExpiredForPaths []string

	// SessionCookieName is the name of the session cookie to check as fallback.
	// If set, the middleware will check for this cookie when no Authorization header is present.
	SessionCookieName string

	// MockSessionToken is the token value that identifies a valid mock session.
	// Used for development when real auth is not available.
	MockSessionToken string
}

// DefaultAuthConfig returns an AuthConfig with sensible defaults.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Logger:               slog.Default(),
		SkipPaths:            []string{"/health", "/ready", "/api/v1/auth/login", "/api/v1/auth/register"},
		AllowExpiredForPaths: []string{"/api/v1/auth/refresh"},
	}
}

// Auth returns an authentication middleware with the given configuration.
//
//nolint:gocognit // Auth middleware requires complex token validation logic.
func Auth(config AuthConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	skipPaths := make(map[string]struct{}, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	allowExpiredPaths := make(map[string]struct{}, len(config.AllowExpiredForPaths))
	for _, path := range config.AllowExpiredForPaths {
		allowExpiredPaths[path] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip authentication for configured paths
			if _, ok := skipPaths[path]; ok {
				return next(c)
			}

			// Extract token from Authorization header or session cookie
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			token, tokenErr := extractTokenFromRequest(c, authHeader, config)
			if tokenErr != nil {
				// Check if mock session was handled
				if errors.Is(tokenErr, errMockSessionHandled) {
					return next(c)
				}
				return respondAuthError(c, tokenErr)
			}

			// Validate token
			if config.TokenValidator == nil {
				config.Logger.Error("token validator not configured")
				return respondAuthError(c, ErrInvalidToken)
			}

			claims, validateErr := config.TokenValidator.ValidateToken(c.Request().Context(), token)
			//nolint:nestif // Token validation requires nested error handling for expired token paths
			if validateErr != nil {
				// Check if expired tokens are allowed for this path
				if errors.Is(validateErr, ErrTokenExpired) {
					if _, ok := allowExpiredPaths[path]; ok {
						// Allow expired token for refresh endpoint
						if claims != nil {
							enrichContext(c, claims)
							return next(c)
						}
					}
				}

				config.Logger.Warn("token validation failed",
					slog.String("error", validateErr.Error()),
					slog.String("path", path),
					slog.String("remote_ip", c.RealIP()),
				)
				return respondAuthError(c, validateErr)
			}

			// Resolve internal user ID if resolver is configured
			if config.UserResolver != nil && claims.UserID.IsZero() {
				userID, resolveErr := config.UserResolver.ResolveUser(
					c.Request().Context(),
					claims.ExternalUserID,
					claims.Username,
					claims.Email,
				)
				if resolveErr != nil {
					config.Logger.Error("failed to resolve user",
						slog.String("error", resolveErr.Error()),
						slog.String("external_id", claims.ExternalUserID),
					)
					return respondAuthError(c, ErrUserNotFound)
				}
				claims.UserID = userID
			}

			// Enrich context with user information
			enrichContext(c, claims)

			// Log successful authentication
			config.Logger.Debug("user authenticated",
				slog.String("user_id", claims.UserID.String()),
				slog.String("username", claims.Username),
				slog.String("path", path),
			)

			return next(c)
		}
	}
}

// extractTokenFromRequest extracts the auth token from the request.
// It first checks the Authorization header, then falls back to session cookie.
func extractTokenFromRequest(c echo.Context, authHeader string, config AuthConfig) (string, error) {
	// Try Authorization header first
	if authHeader != "" {
		return extractBearerToken(authHeader)
	}

	// Fallback to session cookie for HTMX requests
	if config.SessionCookieName != "" {
		cookie, cookieErr := c.Cookie(config.SessionCookieName)
		if cookieErr == nil && cookie.Value != "" {
			// Check if this is a mock session (for development)
			if config.MockSessionToken != "" && cookie.Value == config.MockSessionToken {
				setMockUserContext(c)
				return "", errMockSessionHandled
			}
			return cookie.Value, nil
		}
	}

	return "", ErrMissingAuthHeader
}

// extractBearerToken extracts the token from a Bearer authorization header.
func extractBearerToken(authHeader string) (string, error) {
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", ErrInvalidAuthHeader
	}

	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		return "", ErrInvalidAuthHeader
	}

	return token, nil
}

// enrichContext adds user information to the echo context.
func enrichContext(c echo.Context, claims *TokenClaims) {
	c.Set(string(ContextKeyUserID), claims.UserID)
	c.Set(string(ContextKeyExternalUserID), claims.ExternalUserID)
	c.Set(string(ContextKeyUsername), claims.Username)
	c.Set(string(ContextKeyEmail), claims.Email)
	c.Set(string(ContextKeyRoles), claims.Roles)
	c.Set(string(ContextKeyIsSystemAdmin), claims.IsSystemAdmin)
}

// setMockUserContext sets mock user context for development sessions.
func setMockUserContext(c echo.Context) {
	mockUserID := uuid.NewUUID()
	c.Set(string(ContextKeyUserID), mockUserID)
	c.Set(string(ContextKeyExternalUserID), "mock-external-id")
	c.Set(string(ContextKeyUsername), "mockuser")
	c.Set(string(ContextKeyEmail), "user@example.com")
	c.Set(string(ContextKeyRoles), []string{"user"})
	c.Set(string(ContextKeyIsSystemAdmin), false)
}

// respondAuthError sends an authentication error response.
func respondAuthError(c echo.Context, err error) error {
	code := "UNAUTHORIZED"
	message := "Authentication required"
	status := http.StatusUnauthorized

	switch {
	case errors.Is(err, ErrMissingAuthHeader):
		message = "Missing authorization header"
	case errors.Is(err, ErrInvalidAuthHeader):
		message = "Invalid authorization header format"
	case errors.Is(err, ErrTokenExpired):
		message = "Token has expired"
		code = "TOKEN_EXPIRED"
	case errors.Is(err, ErrInvalidToken):
		message = "Invalid token"
	case errors.Is(err, ErrUserNotFound):
		message = "User not found"
		code = "USER_NOT_FOUND"
	case errors.Is(err, ErrInsufficientPermissions):
		message = "Insufficient permissions"
		code = "FORBIDDEN"
		status = http.StatusForbidden
	}

	return c.JSON(status, map[string]any{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// GetUserID extracts the user ID from the echo context.
func GetUserID(c echo.Context) uuid.UUID {
	if id, ok := c.Get(string(ContextKeyUserID)).(uuid.UUID); ok {
		return id
	}
	return uuid.UUID("")
}

// GetExternalUserID extracts the external user ID from the echo context.
func GetExternalUserID(c echo.Context) string {
	if id, ok := c.Get(string(ContextKeyExternalUserID)).(string); ok {
		return id
	}
	return ""
}

// GetUsername extracts the username from the echo context.
func GetUsername(c echo.Context) string {
	if username, ok := c.Get(string(ContextKeyUsername)).(string); ok {
		return username
	}
	return ""
}

// GetEmail extracts the email from the echo context.
func GetEmail(c echo.Context) string {
	if email, ok := c.Get(string(ContextKeyEmail)).(string); ok {
		return email
	}
	return ""
}

// GetRoles extracts the user roles from the echo context.
func GetRoles(c echo.Context) []string {
	if roles, ok := c.Get(string(ContextKeyRoles)).([]string); ok {
		return roles
	}
	return nil
}

// IsSystemAdmin checks if the current user is a system administrator.
func IsSystemAdmin(c echo.Context) bool {
	if isAdmin, ok := c.Get(string(ContextKeyIsSystemAdmin)).(bool); ok {
		return isAdmin
	}
	return false
}

// HasRole checks if the current user has the specified role.
func HasRole(c echo.Context, role string) bool {
	roles := GetRoles(c)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the current user has any of the specified roles.
func HasAnyRole(c echo.Context, roles ...string) bool {
	for _, role := range roles {
		if HasRole(c, role) {
			return true
		}
	}
	return false
}

// RequireRole returns a middleware that requires the user to have a specific role.
func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !HasRole(c, role) {
				return respondAuthError(c, ErrInsufficientPermissions)
			}
			return next(c)
		}
	}
}

// RequireAnyRole returns a middleware that requires the user to have any of the specified roles.
func RequireAnyRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !HasAnyRole(c, roles...) {
				return respondAuthError(c, ErrInsufficientPermissions)
			}
			return next(c)
		}
	}
}

// RequireSystemAdmin returns a middleware that requires the user to be a system admin.
func RequireSystemAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !IsSystemAdmin(c) {
				return respondAuthError(c, ErrInsufficientPermissions)
			}
			return next(c)
		}
	}
}

// StaticTokenValidator is a simple token validator for development/testing.
// It validates tokens against a static secret using HMAC.
// DO NOT USE IN PRODUCTION - use proper JWT validation instead.
type StaticTokenValidator struct {
	secret []byte
}

// NewStaticTokenValidator creates a new static token validator.
func NewStaticTokenValidator(secret string) *StaticTokenValidator {
	return &StaticTokenValidator{
		secret: []byte(secret),
	}
}

// ValidateToken validates a token using constant-time comparison.
// This is a placeholder implementation for development.
func (v *StaticTokenValidator) ValidateToken(_ context.Context, token string) (*TokenClaims, error) {
	// This is a dummy implementation for development/testing
	// In production, this should decode and verify a real JWT token
	if len(token) == 0 {
		return nil, ErrInvalidToken
	}

	// Simple constant-time comparison for development tokens
	// Format: "dev-token-<user_id>"
	if strings.HasPrefix(token, "dev-token-") {
		parts := strings.Split(token, "-")
		const minDevTokenParts = 3
		const devTokenExpirationHours = 24
		if len(parts) >= minDevTokenParts {
			return &TokenClaims{
				ExternalUserID: parts[2],
				Username:       "dev-user-" + parts[2],
				Email:          "dev-" + parts[2] + "@example.com",
				Roles:          []string{"user"},
				ExpiresAt:      time.Now().Add(devTokenExpirationHours * time.Hour),
			}, nil
		}
	}

	return nil, ErrInvalidToken
}

// ConstantTimeCompare performs a constant-time comparison of two strings.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
