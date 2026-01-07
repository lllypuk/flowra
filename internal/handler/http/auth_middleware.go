package httphandler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// HTMX header constants.
const (
	htmxHeaderValue = "true"
)

// RequireAuth is a middleware that checks if the user is authenticated.
// For regular requests, it redirects to login. For HTMX requests, it returns 401.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for session cookie
		token := getSessionCookie(c)
		if token == "" {
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

		// TODO: Validate token with auth service when real auth is implemented
		// For now, any session cookie is considered valid

		// Use a stable mock user ID for development (based on token value for consistency)
		// This ensures the same session always gets the same user ID
		mockUserID := uuid.DeterministicUUID("mock-user-" + token)

		// Set user info in context compatible with middleware.GetUserID, etc.
		c.Set(string(middleware.ContextKeyUserID), mockUserID)
		c.Set(string(middleware.ContextKeyUsername), "mockuser")
		c.Set(string(middleware.ContextKeyEmail), "user@example.com")

		// Also set legacy "user" key for backwards compatibility
		c.Set("user", map[string]any{
			"id":           mockUserID.String(),
			"email":        "user@example.com",
			"username":     "mockuser",
			"display_name": "Mock User",
		})

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
			// TODO: Validate token with auth service when real auth is implemented
			// Use a stable mock user ID for development (based on token value for consistency)
			mockUserID := uuid.DeterministicUUID("mock-user-" + token)

			// Set user info in context compatible with middleware.GetUserID, etc.
			c.Set(string(middleware.ContextKeyUserID), mockUserID)
			c.Set(string(middleware.ContextKeyUsername), "mockuser")
			c.Set(string(middleware.ContextKeyEmail), "user@example.com")

			// Also set legacy "user" key for backwards compatibility
			c.Set("user", map[string]any{
				"id":           mockUserID.String(),
				"email":        "user@example.com",
				"username":     "mockuser",
				"display_name": "Mock User",
			})
		}

		return next(c)
	}
}
