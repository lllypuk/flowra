package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCORSConfig(t *testing.T) {
	config := middleware.DefaultCORSConfig()

	assert.Equal(t, []string{"*"}, config.AllowOrigins)
	assert.Contains(t, config.AllowMethods, echo.GET)
	assert.Contains(t, config.AllowMethods, echo.POST)
	assert.Contains(t, config.AllowMethods, echo.PUT)
	assert.Contains(t, config.AllowMethods, echo.PATCH)
	assert.Contains(t, config.AllowMethods, echo.DELETE)
	assert.Contains(t, config.AllowMethods, echo.OPTIONS)
	assert.Contains(t, config.AllowHeaders, echo.HeaderOrigin)
	assert.Contains(t, config.AllowHeaders, echo.HeaderContentType)
	assert.Contains(t, config.AllowHeaders, echo.HeaderAccept)
	assert.Contains(t, config.AllowHeaders, echo.HeaderAuthorization)
	assert.Contains(t, config.AllowHeaders, echo.HeaderXRequestID)
	assert.False(t, config.AllowCredentials)
	assert.Empty(t, config.ExposeHeaders)
	assert.Equal(t, middleware.DefaultCORSMaxAge, config.MaxAge)
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name                   string
		config                 middleware.CORSConfig
		requestOrigin          string
		expectedAllowOrigin    string
		expectedAllowMethods   string
		expectedAllowHeaders   string
		expectCredentialsAllow bool
	}{
		{
			name:                   "default config allows all origins",
			config:                 middleware.DefaultCORSConfig(),
			requestOrigin:          "http://example.com",
			expectedAllowOrigin:    "*",
			expectedAllowMethods:   "",
			expectedAllowHeaders:   "",
			expectCredentialsAllow: false,
		},
		{
			name: "specific origin allowed",
			config: middleware.CORSConfig{
				AllowOrigins:     []string{"http://localhost:3000"},
				AllowMethods:     []string{echo.GET, echo.POST},
				AllowHeaders:     []string{echo.HeaderContentType},
				AllowCredentials: true,
			},
			requestOrigin:          "http://localhost:3000",
			expectedAllowOrigin:    "http://localhost:3000",
			expectedAllowMethods:   "",
			expectedAllowHeaders:   "",
			expectCredentialsAllow: true,
		},
		{
			name: "multiple origins configured",
			config: middleware.CORSConfig{
				AllowOrigins: []string{
					"http://localhost:3000",
					"http://localhost:8080",
				},
				AllowMethods: []string{echo.GET},
				AllowHeaders: []string{echo.HeaderContentType},
			},
			requestOrigin:          "http://localhost:8080",
			expectedAllowOrigin:    "http://localhost:8080",
			expectedAllowMethods:   "",
			expectedAllowHeaders:   "",
			expectCredentialsAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(middleware.CORS(tt.config))

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(echo.HeaderOrigin, tt.requestOrigin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expectedAllowOrigin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

			if tt.expectCredentialsAllow {
				assert.Equal(t, "true", rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	config := middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	e := echo.New()
	e.Use(middleware.CORS(config))

	e.POST("/api/users", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	// Send OPTIONS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/users", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:3000")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, "POST")
	req.Header.Set(echo.HeaderAccessControlRequestHeaders, "Content-Type,Authorization")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "http://localhost:3000", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Equal(t, "true", rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
	assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlAllowMethods), "POST")
	assert.Equal(t, "3600", rec.Header().Get(echo.HeaderAccessControlMaxAge))
}

func TestCORSWithOrigins(t *testing.T) {
	tests := []struct {
		name                string
		allowedOrigins      []string
		requestOrigin       string
		expectedAllowOrigin string
	}{
		{
			name:                "single origin allowed",
			allowedOrigins:      []string{"http://example.com"},
			requestOrigin:       "http://example.com",
			expectedAllowOrigin: "http://example.com",
		},
		{
			name:                "multiple origins - first match",
			allowedOrigins:      []string{"http://one.com", "http://two.com", "http://three.com"},
			requestOrigin:       "http://two.com",
			expectedAllowOrigin: "http://two.com",
		},
		{
			name:                "origin not in allowed list",
			allowedOrigins:      []string{"http://allowed.com"},
			requestOrigin:       "http://notallowed.com",
			expectedAllowOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(middleware.CORSWithOrigins(tt.allowedOrigins...))

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(echo.HeaderOrigin, tt.requestOrigin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expectedAllowOrigin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
		})
	}
}

func TestCORSWithOriginsEnablesCredentials(t *testing.T) {
	e := echo.New()
	e.Use(middleware.CORSWithOrigins("http://localhost:3000"))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:3000")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
}

func TestCORSExposeHeaders(t *testing.T) {
	config := middleware.CORSConfig{
		AllowOrigins:  []string{"http://localhost:3000"},
		AllowMethods:  []string{echo.GET},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	}

	e := echo.New()
	e.Use(middleware.CORS(config))

	e.GET("/test", func(c echo.Context) error {
		c.Response().Header().Set("X-Custom-Header", "value")
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:3000")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	exposeHeaders := rec.Header().Get(echo.HeaderAccessControlExposeHeaders)
	assert.Contains(t, exposeHeaders, "X-Custom-Header")
	assert.Contains(t, exposeHeaders, "X-Another-Header")
}

func TestCORSNoOriginHeader(t *testing.T) {
	e := echo.New()
	e.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Request without Origin header (same-origin request)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestCORSPreflightWithCustomHeaders(t *testing.T) {
	config := middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST},
		AllowHeaders: []string{
			echo.HeaderContentType,
			echo.HeaderAuthorization,
			"X-Custom-Header",
		},
	}

	e := echo.New()
	e.Use(middleware.CORS(config))

	e.POST("/api/data", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set(echo.HeaderOrigin, "http://example.com")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, "POST")
	req.Header.Set(echo.HeaderAccessControlRequestHeaders, "Content-Type,X-Custom-Header")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	allowHeaders := rec.Header().Get(echo.HeaderAccessControlAllowHeaders)
	assert.Contains(t, allowHeaders, echo.HeaderContentType)
	assert.Contains(t, allowHeaders, "X-Custom-Header")
}
