package httphandler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Cookie names.
const (
	sessionCookieName  = "flowra_session"
	stateCookieName    = "flowra_state"
	redirectCookieName = "flowra_redirect"
)

// Cookie durations.
const (
	stateCookieMaxAge    = 300 // 5 minutes
	redirectCookieMaxAge = 300 // 5 minutes
	stateRandomBytes     = 16  // Random bytes for state generation
)

// setSessionCookie sets the session cookie with the access token.
func setSessionCookie(c echo.Context, token string, expiresIn int) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   expiresIn,
		HttpOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}

// getSessionCookie retrieves the session cookie value.
func getSessionCookie(c echo.Context) string {
	cookie, err := c.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// clearSessionCookie clears the session cookie.
func clearSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	c.SetCookie(cookie)
}

// setStateCookie sets the state cookie for CSRF protection in OAuth flow.
func setStateCookie(c echo.Context, state string) {
	cookie := &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   stateCookieMaxAge,
		HttpOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}

// getStateCookie retrieves the state cookie value.
func getStateCookie(c echo.Context) string {
	cookie, err := c.Cookie(stateCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// clearStateCookie clears the state cookie.
func clearStateCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:   stateCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	c.SetCookie(cookie)
}

// generateState generates a random state string for OAuth flow.
func generateState() string {
	b := make([]byte, stateRandomBytes)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// setRedirectCookie stores the intended destination URL before login.
func setRedirectCookie(c echo.Context, url string) {
	cookie := &http.Cookie{
		Name:     redirectCookieName,
		Value:    url,
		Path:     "/",
		MaxAge:   redirectCookieMaxAge,
		HttpOnly: true,
	}
	c.SetCookie(cookie)
}

// getRedirectCookie retrieves the redirect URL cookie value.
func getRedirectCookie(c echo.Context) string {
	cookie, err := c.Cookie(redirectCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// clearRedirectCookie clears the redirect cookie.
func clearRedirectCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:   redirectCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	c.SetCookie(cookie)
}

// GetRedirectURI builds the OAuth redirect URI based on the request.
// This will be used when integrating with real OAuth provider (Keycloak).
func GetRedirectURI(c echo.Context) string {
	scheme := c.Scheme()
	host := c.Request().Host
	return scheme + "://" + host + "/auth/callback"
}
