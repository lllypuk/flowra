package httpserver_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRouterConfig(t *testing.T) {
	config := httpserver.DefaultRouterConfig()

	assert.NotNil(t, config.Logger)
	assert.Equal(t, "/api/v1", config.APIPrefix)
	assert.NotNil(t, config.CORSConfig.AllowOrigins)
	assert.NotNil(t, config.LoggingConfig.SkipPaths)
	assert.NotNil(t, config.RecoveryConfig.Logger)
}

func TestNewRouter(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	router := httpserver.NewRouter(e, config)

	assert.NotNil(t, router)
	assert.Equal(t, e, router.Echo())
	assert.NotNil(t, router.Public())
	assert.NotNil(t, router.Auth())
	assert.NotNil(t, router.Workspace())
}

func TestNewRouter_EmptyAPIPrefix(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.APIPrefix = ""

	router := httpserver.NewRouter(e, config)

	// Should use default prefix
	assert.NotNil(t, router.Public())
}

func TestNewRouter_NilLogger(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.Logger = nil

	router := httpserver.NewRouter(e, config)

	assert.NotNil(t, router)
}

func TestRouter_PublicRoutes(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	router.Public().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "public")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public", rec.Body.String())
}

func TestRouter_AuthRoutes_WithMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	authCalled := false
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCalled = true
			// Simulate auth check - reject if no Authorization header
			if c.Request().Header.Get("Authorization") == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	router.Auth().GET("/profile", func(c echo.Context) error {
		return c.String(http.StatusOK, "profile")
	})

	// Without auth header - should fail
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, authCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// With auth header - should succeed
	authCalled = false
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, authCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "profile", rec.Body.String())
}

func TestRouter_AuthRoutes_NoMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = nil

	router := httpserver.NewRouter(e, config)

	router.Auth().GET("/profile", func(c echo.Context) error {
		return c.String(http.StatusOK, "profile")
	})

	// Should work without auth (same as public)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_WorkspaceRoutes_WithMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	workspaceID := uuid.NewUUID()

	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", uuid.NewUUID())
			return next(c)
		}
	}

	workspaceCalled := false
	config.WorkspaceMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			workspaceCalled = true
			wsID := c.Param("workspace_id")
			if wsID != workspaceID.String() {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "no access"})
			}
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	router.Workspace().GET("/chats", func(c echo.Context) error {
		return c.String(http.StatusOK, "chats")
	})

	// Valid workspace ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID.String()+"/chats", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, workspaceCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "chats", rec.Body.String())

	// Invalid workspace ID
	workspaceCalled = false
	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/invalid-id/chats", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, workspaceCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRouter_WorkspaceRoutes_NoMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = nil
	config.WorkspaceMiddleware = nil

	router := httpserver.NewRouter(e, config)

	router.Workspace().GET("/chats", func(c echo.Context) error {
		wsID := c.Param("workspace_id")
		return c.String(http.StatusOK, "chats for "+wsID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws123/chats", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "chats for ws123", rec.Body.String())
}

func TestRouter_RegisterHealthEndpoints(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	router.RegisterHealthEndpoints(func() bool {
		return true
	})

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")

	// Test ready endpoint
	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
}

func TestRouter_RegisterHealthEndpoints_NotReady(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	router.RegisterHealthEndpoints(func() bool {
		return false
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestRouter_RegisterHealthEndpoints_NilCheck(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	router.RegisterHealthEndpoints(nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// With nil check, should be ready
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_GlobalMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	rateLimitCalled := false
	config.RateLimitMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rateLimitCalled = true
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	router.Public().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rateLimitCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_RecoveryMiddleware(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.RecoveryConfig = middleware.RecoveryConfig{
		Logger: slog.Default(),
	}

	router := httpserver.NewRouter(e, config)

	router.Public().GET("/panic", func(_ echo.Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		e.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRouteBuilder(t *testing.T) {
	e := echo.New()
	group := e.Group("/api")

	middlewareCalled := false
	testMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	}

	builder := httpserver.NewRouteBuilder(group).Use(testMiddleware)

	builder.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouteBuilder_AllMethods(t *testing.T) {
	e := echo.New()
	group := e.Group("/api")

	builder := httpserver.NewRouteBuilder(group)

	builder.GET("/get", func(c echo.Context) error {
		return c.String(http.StatusOK, "GET")
	})
	builder.POST("/post", func(c echo.Context) error {
		return c.String(http.StatusOK, "POST")
	})
	builder.PUT("/put", func(c echo.Context) error {
		return c.String(http.StatusOK, "PUT")
	})
	builder.PATCH("/patch", func(c echo.Context) error {
		return c.String(http.StatusOK, "PATCH")
	})
	builder.DELETE("/delete", func(c echo.Context) error {
		return c.String(http.StatusOK, "DELETE")
	})

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/get", "GET"},
		{http.MethodPost, "/api/post", "POST"},
		{http.MethodPut, "/api/put", "PUT"},
		{http.MethodPatch, "/api/patch", "PATCH"},
		{http.MethodDelete, "/api/delete", "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.body, rec.Body.String())
		})
	}
}

func TestRouteBuilder_Group(t *testing.T) {
	e := echo.New()
	group := e.Group("/api")

	middlewareCalled := false
	builder := httpserver.NewRouteBuilder(group).Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	})

	subGroup := builder.Group("/v1")
	subGroup.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_NewWorkspaceRouteGroup(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), uuid.NewUUID())
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	chats := router.NewWorkspaceRouteGroup("/chats")
	chats.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "list chats")
	})
	chats.POST("", func(c echo.Context) error {
		return c.String(http.StatusOK, "create chat")
	})
	chats.GET("/:chat_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "get chat "+c.Param("chat_id"))
	})

	workspaceID := uuid.NewUUID()

	// Test GET list
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID.String()+"/chats", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "list chats", rec.Body.String())

	// Test POST
	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID.String()+"/chats", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "create chat", rec.Body.String())

	// Test GET single
	chatID := uuid.NewUUID()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID.String()+"/chats/"+chatID.String(), nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "get chat "+chatID.String(), rec.Body.String())
}

func TestWorkspaceRouteGroup_RequireAdmin(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}

	router := httpserver.NewRouter(e, config)

	settings := router.NewWorkspaceRouteGroup("/settings")

	// Add admin context
	settings.Group().Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Request().Header.Get("X-Role")
			if role != "" {
				c.Set(string(middleware.ContextKeyWorkspaceRole), role)
			}
			return next(c)
		}
	})

	adminSettings := settings.RequireAdmin()
	adminSettings.PUT("", func(c echo.Context) error {
		return c.String(http.StatusOK, "settings updated")
	})

	workspaceID := uuid.NewUUID()

	// As member - should fail
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/"+workspaceID.String()+"/settings", nil)
	req.Header.Set("X-Role", middleware.WorkspaceRoleMember)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// As admin - should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/"+workspaceID.String()+"/settings", nil)
	req.Header.Set("X-Role", middleware.WorkspaceRoleAdmin)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWorkspaceRouteGroup_RequireOwner(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}

	router := httpserver.NewRouter(e, config)

	danger := router.NewWorkspaceRouteGroup("/danger")

	danger.Group().Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Request().Header.Get("X-Role")
			if role != "" {
				c.Set(string(middleware.ContextKeyWorkspaceRole), role)
			}
			return next(c)
		}
	})

	ownerOnly := danger.RequireOwner()
	ownerOnly.DELETE("", func(c echo.Context) error {
		return c.String(http.StatusOK, "deleted")
	})

	workspaceID := uuid.NewUUID()

	// As admin - should fail
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+workspaceID.String()+"/danger", nil)
	req.Header.Set("X-Role", middleware.WorkspaceRoleAdmin)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// As owner - should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+workspaceID.String()+"/danger", nil)
	req.Header.Set("X-Role", middleware.WorkspaceRoleOwner)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_NewAuthRouteGroup(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	authCalled := false
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCalled = true
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	users := router.NewAuthRouteGroup("/users")
	users.GET("/me", func(c echo.Context) error {
		return c.String(http.StatusOK, "current user")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, authCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "current user", rec.Body.String())
}

func TestAuthRouteGroup_RequireRole(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Request().Header.Get("X-Role")
			if role != "" {
				c.Set(string(middleware.ContextKeyRoles), []string{role})
			}
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	admin := router.NewAuthRouteGroup("/admin").RequireRole("admin")
	admin.GET("/dashboard", func(c echo.Context) error {
		return c.String(http.StatusOK, "admin dashboard")
	})

	// Without admin role - should fail
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	req.Header.Set("X-Role", "user")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// With admin role - should succeed
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	req.Header.Set("X-Role", "admin")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthRouteGroup_RequireSystemAdmin(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAdmin := c.Request().Header.Get("X-System-Admin") == "true"
			c.Set(string(middleware.ContextKeyIsSystemAdmin), isAdmin)
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	system := router.NewAuthRouteGroup("/system").RequireSystemAdmin()
	system.GET("/config", func(c echo.Context) error {
		return c.String(http.StatusOK, "system config")
	})

	// Without system admin - should fail
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// With system admin - should succeed
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	req.Header.Set("X-System-Admin", "true")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthRouteGroup_AllMethods(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}

	router := httpserver.NewRouter(e, config)

	users := router.NewAuthRouteGroup("/users")
	users.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "GET")
	})
	users.POST("", func(c echo.Context) error {
		return c.String(http.StatusOK, "POST")
	})
	users.PUT("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "PUT")
	})
	users.PATCH("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "PATCH")
	})
	users.DELETE("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "DELETE")
	})

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/users", "GET"},
		{http.MethodPost, "/api/v1/users", "POST"},
		{http.MethodPut, "/api/v1/users/123", "PUT"},
		{http.MethodPatch, "/api/v1/users/123", "PATCH"},
		{http.MethodDelete, "/api/v1/users/123", "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.body, rec.Body.String())
		})
	}
}

func TestWorkspaceRouteGroup_AllMethods(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}

	router := httpserver.NewRouter(e, config)

	messages := router.NewWorkspaceRouteGroup("/messages")
	messages.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "GET")
	})
	messages.POST("", func(c echo.Context) error {
		return c.String(http.StatusOK, "POST")
	})
	messages.PUT("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "PUT")
	})
	messages.PATCH("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "PATCH")
	})
	messages.DELETE("/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "DELETE")
	})

	workspaceID := uuid.NewUUID()
	basePath := "/api/v1/workspaces/" + workspaceID.String() + "/messages"

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, basePath, "GET"},
		{http.MethodPost, basePath, "POST"},
		{http.MethodPut, basePath + "/123", "PUT"},
		{http.MethodPatch, basePath + "/123", "PATCH"},
		{http.MethodDelete, basePath + "/123", "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.body, rec.Body.String())
		})
	}
}

// RouteRegistrar interface test
type testRegistrar struct {
	called bool
}

func (r *testRegistrar) RegisterRoutes(router *httpserver.Router) {
	r.called = true
	router.Public().GET("/registered", func(c echo.Context) error {
		return c.String(http.StatusOK, "registered")
	})
}

func TestRouter_RegisterAll(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	registrar := &testRegistrar{}
	router.RegisterAll(registrar)

	assert.True(t, registrar.called)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/registered", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "registered", rec.Body.String())
}

func TestRouter_RegisterAll_Multiple(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	registrar1 := &testRegistrar{}
	registrar2 := &testRegistrar{}

	router.RegisterAll(registrar1, registrar2)

	assert.True(t, registrar1.called)
	assert.True(t, registrar2.called)
}

func TestRouter_PrintRoutes(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()
	router := httpserver.NewRouter(e, config)

	router.Public().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Should not panic
	require.NotPanics(t, func() {
		router.PrintRoutes()
	})
}

func TestRouter_MiddlewareChain(t *testing.T) {
	e := echo.New()
	config := httpserver.DefaultRouterConfig()

	order := make([]string, 0)

	config.AuthMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			order = append(order, "auth")
			return next(c)
		}
	}

	config.WorkspaceMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			order = append(order, "workspace")
			return next(c)
		}
	}

	router := httpserver.NewRouter(e, config)

	router.Workspace().GET("/test", func(c echo.Context) error {
		order = append(order, "handler")
		return c.String(http.StatusOK, "ok")
	})

	workspaceID := uuid.NewUUID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID.String()+"/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check middleware order: logging (from global) -> auth -> workspace -> handler
	// Note: recovery, CORS, logging are applied globally first
	assert.Contains(t, order, "auth")
	assert.Contains(t, order, "workspace")
	assert.Contains(t, order, "handler")

	// Auth should come before workspace
	authIdx := -1
	workspaceIdx := -1
	handlerIdx := -1
	for i, v := range order {
		if v == "auth" {
			authIdx = i
		}
		if v == "workspace" {
			workspaceIdx = i
		}
		if v == "handler" {
			handlerIdx = i
		}
	}

	assert.Less(t, authIdx, workspaceIdx)
	assert.Less(t, workspaceIdx, handlerIdx)
}
