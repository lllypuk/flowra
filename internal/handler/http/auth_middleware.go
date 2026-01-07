package httphandler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// HTMX header constants.
const (
	htmxHeaderValue = "true"
)

// PageAuthConfig holds configuration for page authentication middleware.
type PageAuthConfig struct {
	// TokenValidator validates JWT tokens from session cookies.
	TokenValidator middleware.TokenValidator

	// UserResolver resolves users from external IDs.
	UserResolver middleware.UserResolver

	// Logger for auth events.
	Logger *slog.Logger
}

// globalPageAuthConfig holds the global configuration for page auth middleware.
// This is set during application startup via SetPageAuthConfig.
//
//nolint:gochecknoglobals // Required for middleware configuration injection
var globalPageAuthConfig *PageAuthConfig

// SetPageAuthConfig sets the global page auth configuration.
// This must be called during application startup before any requests are processed.
func SetPageAuthConfig(config *PageAuthConfig) {
	globalPageAuthConfig = config
}

// RequireAuth is a middleware that checks if the user is authenticated.
// For regular requests, it redirects to login. For HTMX requests, it returns 401.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for session cookie
		token := getSessionCookie(c)
		if token == "" {
			return handleUnauthenticated(c)
		}

		// Validate token and set user context
		if err := validateAndSetUserContext(c, token); err != nil {
			logger().Warn("token validation failed",
				slog.String("error", err.Error()),
				slog.String("path", c.Request().URL.Path),
			)
			// Clear invalid session cookie
			clearSessionCookie(c)
			return handleUnauthenticated(c)
		}

		return next(c)
	}
}

// OptionalAuth is a middleware that checks for authentication but doesn't require it.
// It sets user info in context if authenticated, but allows the request to proceed either way.
func OptionalAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for session cookie
		token := getSessionCookie(c)
		if token != "" {
			// Try to validate, but don't fail if invalid
			if err := validateAndSetUserContext(c, token); err != nil {
				logger().Debug("optional auth: token validation failed",
					slog.String("error", err.Error()),
				)
				// Clear invalid session cookie
				clearSessionCookie(c)
			}
		}

		return next(c)
	}
}

// validateAndSetUserContext validates the JWT token and sets user info in context.
func validateAndSetUserContext(c echo.Context, token string) error {
	config := globalPageAuthConfig

	// If no config is set, use mock mode for development
	if config == nil || config.TokenValidator == nil {
		return setMockUserContext(c, token)
	}

	ctx := c.Request().Context()

	// Validate token
	claims, err := config.TokenValidator.ValidateToken(ctx, token)
	if err != nil {
		return err
	}

	// Resolve internal user ID if resolver is configured
	userID := claims.UserID
	if config.UserResolver != nil && userID.IsZero() && claims.ExternalUserID != "" {
		resolvedID, resolveErr := config.UserResolver.ResolveUser(
			ctx,
			claims.ExternalUserID,
			claims.Username,
			claims.Email,
		)
		if resolveErr != nil {
			return resolveErr
		}
		userID = resolvedID
	}

	// Set user info in context
	setUserContext(c, userID, claims)

	return nil
}

// setUserContext sets user information in echo context.
func setUserContext(c echo.Context, userID uuid.UUID, claims *middleware.TokenClaims) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyExternalUserID), claims.ExternalUserID)
	c.Set(string(middleware.ContextKeyUsername), claims.Username)
	c.Set(string(middleware.ContextKeyEmail), claims.Email)
	c.Set(string(middleware.ContextKeyRoles), claims.Roles)
	c.Set(string(middleware.ContextKeyGroups), claims.Groups)
	c.Set(string(middleware.ContextKeyIsSystemAdmin), claims.IsSystemAdmin)
	c.Set(string(middleware.ContextKeyClaims), claims)

	// Also set legacy "user" key for backwards compatibility
	displayName := claims.Username
	if displayName == "" {
		displayName = claims.Email
	}
	c.Set("user", map[string]any{
		"id":           userID.String(),
		"email":        claims.Email,
		"username":     claims.Username,
		"display_name": displayName,
	})
}

// setMockUserContext sets mock user context for development mode.
func setMockUserContext(c echo.Context, token string) error {
	// Use a stable mock user ID for development (based on token value for consistency)
	mockUserID := uuid.DeterministicUUID("mock-user-" + token)

	claims := &middleware.TokenClaims{
		UserID:         mockUserID,
		ExternalUserID: "mock-external-id",
		Username:       "mockuser",
		Email:          "user@example.com",
		Roles:          []string{"user"},
		Groups:         []string{},
		IsSystemAdmin:  false,
	}

	setUserContext(c, mockUserID, claims)

	logger().Debug("using mock user context (development mode)",
		slog.String("user_id", mockUserID.String()),
	)

	return nil
}

// handleUnauthenticated handles unauthenticated requests.
func handleUnauthenticated(c echo.Context) error {
	// For HTMX requests, return 401 with HX-Redirect header
	if c.Request().Header.Get("Hx-Request") == htmxHeaderValue {
		c.Response().Header().Set("Hx-Redirect", "/login")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// For regular requests, save destination and redirect
	if c.Request().Method == http.MethodGet {
		setRedirectCookie(c, c.Request().URL.Path)
	}
	return c.Redirect(http.StatusFound, "/login")
}

// logger returns the configured logger or default.
func logger() *slog.Logger {
	if globalPageAuthConfig != nil && globalPageAuthConfig.Logger != nil {
		return globalPageAuthConfig.Logger
	}
	return slog.Default()
}

// GetCurrentUser returns user information from context for templates.
// Returns nil if user is not authenticated.
func GetCurrentUser(c echo.Context) map[string]any {
	user, ok := c.Get("user").(map[string]any)
	if !ok {
		return nil
	}
	return user
}

// IsAuthenticated checks if the current request has a valid authenticated user.
func IsAuthenticated(c echo.Context) bool {
	userID := middleware.GetUserID(c)
	return !userID.IsZero()
}

// GetUserIDFromContext returns the user ID from context.
// Returns zero UUID if not authenticated.
//
// Deprecated: Use middleware.GetUserID(c) for echo.Context instead.
func GetUserIDFromContext(_ context.Context) uuid.UUID {
	// This function is kept for backwards compatibility
	// Use middleware.GetUserID(c) for echo.Context
	return uuid.UUID("")
}
