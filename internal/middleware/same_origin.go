package middleware

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

// CodeOriginMismatch is the error code of a request refused by RequireSameOriginForCookie.
const CodeOriginMismatch = "ORIGIN_MISMATCH"

// RequireSameOriginForCookie rejects a cookie-authenticated request whose Origin (or Referer) is not the request Host.
// A request with an Authorization header is not driven by a browser's ambient cookie and passes.
func RequireSameOriginForCookie(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if req.Header.Get(echo.HeaderAuthorization) != "" {
			return next(c)
		}
		origin := req.Header.Get(echo.HeaderOrigin)
		if origin == "" {
			origin = req.Header.Get("Referer")
		}
		if u, err := url.Parse(origin); origin == "" || err != nil || u.Host == "" || u.Host != req.Host {
			return c.JSON(http.StatusForbidden, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    CodeOriginMismatch,
					"message": "the Origin of the request does not match the server",
				},
			})
		}
		return next(c)
	}
}
