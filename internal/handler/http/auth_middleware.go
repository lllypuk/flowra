package httphandler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequireAuth is a middleware that checks if the user is authenticated.
// If not, it redirects to the login page and saves the intended destination.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for session cookie
		token := getSessionCookie(c)
		if token == "" {
			// Save intended destination
			if c.Request().Method == http.MethodGet {
				setRedirectCookie(c, c.Request().URL.Path)
			}
			return c.Redirect(http.StatusFound, "/login")
		}

		// TODO: Validate token with auth service when real auth is implemented
		// For now, any session cookie is considered valid

		// Set user info in context (mock data for now)
		c.Set("user", map[string]any{
			"id":           "mock-user-id",
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
			// Set user info in context (mock data for now)
			c.Set("user", map[string]any{
				"id":           "mock-user-id",
				"email":        "user@example.com",
				"username":     "mockuser",
				"display_name": "Mock User",
			})
		}

		return next(c)
	}
}
