package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CORS configuration constants.
const (
	// DefaultCORSMaxAge is the default max age for CORS preflight cache (24 hours in seconds).
	DefaultCORSMaxAge = 86400
)

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	// AllowOrigins defines a list of origins that may access the resource.
	// Use "*" to allow all origins.
	AllowOrigins []string

	// AllowMethods defines a list of methods allowed when accessing the resource.
	AllowMethods []string

	// AllowHeaders defines a list of request headers that can be used when
	// making the actual request.
	AllowHeaders []string

	// AllowCredentials indicates whether the request can include user credentials.
	AllowCredentials bool

	// ExposeHeaders defines a list of headers that browsers are allowed to access.
	ExposeHeaders []string

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	MaxAge int
}

// DefaultCORSConfig returns a CORSConfig with sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET,
			echo.HEAD,
			echo.PUT,
			echo.PATCH,
			echo.POST,
			echo.DELETE,
			echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderXRequestID,
		},
		AllowCredentials: false,
		ExposeHeaders:    []string{},
		MaxAge:           DefaultCORSMaxAge,
	}
}

// CORS returns a CORS middleware with the given configuration.
func CORS(config CORSConfig) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     config.AllowOrigins,
		AllowMethods:     config.AllowMethods,
		AllowHeaders:     config.AllowHeaders,
		AllowCredentials: config.AllowCredentials,
		ExposeHeaders:    config.ExposeHeaders,
		MaxAge:           config.MaxAge,
	})
}

// CORSWithOrigins returns a CORS middleware configured for specific origins.
func CORSWithOrigins(origins ...string) echo.MiddlewareFunc {
	config := DefaultCORSConfig()
	config.AllowOrigins = origins
	config.AllowCredentials = true
	return CORS(config)
}
