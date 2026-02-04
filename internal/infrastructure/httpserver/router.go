package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterConfig holds configuration for the router.
type RouterConfig struct {
	// Logger is the structured logger for router events.
	Logger *slog.Logger

	// AuthMiddleware is the authentication middleware to use for protected routes.
	AuthMiddleware echo.MiddlewareFunc

	// WorkspaceMiddleware is the workspace access middleware.
	WorkspaceMiddleware echo.MiddlewareFunc

	// RateLimitMiddleware is the rate limiting middleware.
	RateLimitMiddleware echo.MiddlewareFunc

	// CORSConfig is the CORS configuration.
	CORSConfig middleware.CORSConfig

	// LoggingConfig is the logging middleware configuration.
	LoggingConfig middleware.LoggingConfig

	// RecoveryConfig is the recovery middleware configuration.
	RecoveryConfig middleware.RecoveryConfig

	// APIPrefix is the prefix for all API routes.
	// Default is "/api/v1".
	APIPrefix string
}

// DefaultRouterConfig returns a RouterConfig with sensible defaults.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		Logger:         slog.Default(),
		CORSConfig:     middleware.DefaultCORSConfig(),
		LoggingConfig:  middleware.DefaultLoggingConfig(),
		RecoveryConfig: middleware.DefaultRecoveryConfig(),
		APIPrefix:      "/api/v1",
	}
}

// Router manages HTTP route groups and middleware chains.
type Router struct {
	echo   *echo.Echo
	config RouterConfig
	logger *slog.Logger

	// Route groups
	public    *echo.Group
	auth      *echo.Group
	workspace *echo.Group
}

// NewRouter creates a new router with the given configuration.
func NewRouter(e *echo.Echo, config RouterConfig) *Router {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.APIPrefix == "" {
		config.APIPrefix = "/api/v1"
	}

	r := &Router{
		echo:   e,
		config: config,
		logger: config.Logger,
	}

	// Apply global middleware
	r.setupGlobalMiddleware()

	// Create route groups
	r.setupRouteGroups()

	return r
}

// setupGlobalMiddleware applies global middleware to the Echo instance.
func (r *Router) setupGlobalMiddleware() {
	// Recovery middleware (must be first to catch all panics)
	r.echo.Use(middleware.RecoveryWithConfig(r.config.RecoveryConfig))

	// CORS middleware
	r.echo.Use(middleware.CORS(r.config.CORSConfig))

	// Logging middleware
	r.echo.Use(middleware.Logging(r.config.LoggingConfig))

	// Rate limiting middleware (if configured)
	if r.config.RateLimitMiddleware != nil {
		r.echo.Use(r.config.RateLimitMiddleware)
	}
}

// setupRouteGroups creates the route group hierarchy.
func (r *Router) setupRouteGroups() {
	// Public routes - no authentication required
	r.public = r.echo.Group(r.config.APIPrefix)

	// Authenticated routes - require valid JWT token
	if r.config.AuthMiddleware != nil {
		r.auth = r.public.Group("", r.config.AuthMiddleware)
	} else {
		// If no auth middleware, authenticated group is same as public
		r.auth = r.public
		r.logger.Warn("no auth middleware configured, authenticated routes are public")
	}

	// Workspace-scoped routes - require workspace membership
	if r.config.WorkspaceMiddleware != nil {
		r.workspace = r.auth.Group("/workspaces/:workspace_id", r.config.WorkspaceMiddleware)
	} else {
		// If no workspace middleware, create group without access check
		r.workspace = r.auth.Group("/workspaces/:workspace_id")
		r.logger.Warn("no workspace middleware configured, workspace routes skip membership check")
	}
}

// Echo returns the underlying Echo instance.
func (r *Router) Echo() *echo.Echo {
	return r.echo
}

// Public returns the public route group (no authentication required).
// Use for: login, registration, public info, health checks, etc.
func (r *Router) Public() *echo.Group {
	return r.public
}

// Auth returns the authenticated route group (requires valid JWT).
// Use for: user profile, global settings, workspace list, etc.
func (r *Router) Auth() *echo.Group {
	return r.auth
}

// Workspace returns the workspace-scoped route group.
// Requires both authentication and workspace membership.
// Use for: chats, messages, tasks, workspace settings, etc.
func (r *Router) Workspace() *echo.Group {
	return r.workspace
}

// RegisterHealthEndpoints registers health and readiness endpoints.
//
// Deprecated: Use RegisterHealthEndpointsWithChecker or RegisterHealthEndpointsSimple instead.
// This function uses a callback without context, which can lead to incorrect behavior
// under load or when the request is cancelled.
//
// The readinessCheck callback here does NOT receive a context, which means:
//   - It cannot respect request cancellation/deadlines
//   - It may use stale or incorrect context if called improperly
//
// Migration: Replace with RegisterHealthEndpointsSimple(func(ctx context.Context) bool { ... })
// or implement HealthChecker interface and use RegisterHealthEndpointsWithChecker.
func (r *Router) RegisterHealthEndpoints(readinessCheck func() bool) {
	r.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{Status: StatusHealthy})
	})

	r.echo.GET("/ready", func(c echo.Context) error {
		if readinessCheck == nil || readinessCheck() {
			return c.JSON(http.StatusOK, HealthResponse{Status: StatusReady})
		}
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: StatusNotReady})
	})
}

// RouteBuilder provides a fluent API for building routes.
type RouteBuilder struct {
	group      *echo.Group
	middleware []echo.MiddlewareFunc
}

// NewRouteBuilder creates a new route builder for the given group.
func NewRouteBuilder(group *echo.Group) *RouteBuilder {
	return &RouteBuilder{
		group:      group,
		middleware: make([]echo.MiddlewareFunc, 0),
	}
}

// Use adds middleware to the route builder.
func (rb *RouteBuilder) Use(middleware ...echo.MiddlewareFunc) *RouteBuilder {
	rb.middleware = append(rb.middleware, middleware...)
	return rb
}

// Group creates a sub-group with the builder's middleware.
func (rb *RouteBuilder) Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.Group(prefix, allMiddleware...)
}

// GET registers a GET route with the builder's middleware.
func (rb *RouteBuilder) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.GET(path, h, allMiddleware...)
}

// POST registers a POST route with the builder's middleware.
func (rb *RouteBuilder) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.POST(path, h, allMiddleware...)
}

// PUT registers a PUT route with the builder's middleware.
func (rb *RouteBuilder) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.PUT(path, h, allMiddleware...)
}

// PATCH registers a PATCH route with the builder's middleware.
func (rb *RouteBuilder) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.PATCH(path, h, allMiddleware...)
}

// DELETE registers a DELETE route with the builder's middleware.
func (rb *RouteBuilder) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	allMiddleware := make([]echo.MiddlewareFunc, 0, len(rb.middleware)+len(m))
	allMiddleware = append(allMiddleware, rb.middleware...)
	allMiddleware = append(allMiddleware, m...)
	return rb.group.DELETE(path, h, allMiddleware...)
}

// RouteRegistrar defines the interface for registering routes.
type RouteRegistrar interface {
	RegisterRoutes(r *Router)
}

// RegisterAll registers all route registrars with the router.
func (r *Router) RegisterAll(registrars ...RouteRegistrar) {
	for _, registrar := range registrars {
		registrar.RegisterRoutes(r)
	}
}

// WorkspaceRouteGroup provides a convenient way to register workspace-scoped routes.
type WorkspaceRouteGroup struct {
	group  *echo.Group
	router *Router
}

// NewWorkspaceRouteGroup creates a new workspace route group with additional path prefix.
func (r *Router) NewWorkspaceRouteGroup(prefix string, m ...echo.MiddlewareFunc) *WorkspaceRouteGroup {
	return &WorkspaceRouteGroup{
		group:  r.workspace.Group(prefix, m...),
		router: r,
	}
}

// Group returns the underlying echo group.
func (wrg *WorkspaceRouteGroup) Group() *echo.Group {
	return wrg.group
}

// GET registers a GET route.
func (wrg *WorkspaceRouteGroup) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return wrg.group.GET(path, h, m...)
}

// POST registers a POST route.
func (wrg *WorkspaceRouteGroup) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return wrg.group.POST(path, h, m...)
}

// PUT registers a PUT route.
func (wrg *WorkspaceRouteGroup) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return wrg.group.PUT(path, h, m...)
}

// PATCH registers a PATCH route.
func (wrg *WorkspaceRouteGroup) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return wrg.group.PATCH(path, h, m...)
}

// DELETE registers a DELETE route.
func (wrg *WorkspaceRouteGroup) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return wrg.group.DELETE(path, h, m...)
}

// RequireAdmin adds the workspace admin requirement to the route.
func (wrg *WorkspaceRouteGroup) RequireAdmin() *WorkspaceRouteGroup {
	return &WorkspaceRouteGroup{
		group:  wrg.group.Group("", middleware.RequireWorkspaceAdmin()),
		router: wrg.router,
	}
}

// RequireOwner adds the workspace owner requirement to the route.
func (wrg *WorkspaceRouteGroup) RequireOwner() *WorkspaceRouteGroup {
	return &WorkspaceRouteGroup{
		group:  wrg.group.Group("", middleware.RequireWorkspaceOwner()),
		router: wrg.router,
	}
}

// AuthRouteGroup provides a convenient way to register authenticated routes.
type AuthRouteGroup struct {
	group  *echo.Group
	router *Router
}

// NewAuthRouteGroup creates a new authenticated route group with additional path prefix.
func (r *Router) NewAuthRouteGroup(prefix string, m ...echo.MiddlewareFunc) *AuthRouteGroup {
	return &AuthRouteGroup{
		group:  r.auth.Group(prefix, m...),
		router: r,
	}
}

// Group returns the underlying echo group.
func (arg *AuthRouteGroup) Group() *echo.Group {
	return arg.group
}

// GET registers a GET route.
func (arg *AuthRouteGroup) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return arg.group.GET(path, h, m...)
}

// POST registers a POST route.
func (arg *AuthRouteGroup) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return arg.group.POST(path, h, m...)
}

// PUT registers a PUT route.
func (arg *AuthRouteGroup) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return arg.group.PUT(path, h, m...)
}

// PATCH registers a PATCH route.
func (arg *AuthRouteGroup) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return arg.group.PATCH(path, h, m...)
}

// DELETE registers a DELETE route.
func (arg *AuthRouteGroup) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return arg.group.DELETE(path, h, m...)
}

// RequireRole adds a role requirement to the route.
func (arg *AuthRouteGroup) RequireRole(role string) *AuthRouteGroup {
	return &AuthRouteGroup{
		group:  arg.group.Group("", middleware.RequireRole(role)),
		router: arg.router,
	}
}

// RequireSystemAdmin adds the system admin requirement to the route.
func (arg *AuthRouteGroup) RequireSystemAdmin() *AuthRouteGroup {
	return &AuthRouteGroup{
		group:  arg.group.Group("", middleware.RequireSystemAdmin()),
		router: arg.router,
	}
}

// PrintRoutes logs all registered routes (for debugging).
func (r *Router) PrintRoutes() {
	for _, route := range r.echo.Routes() {
		r.logger.Debug("registered route",
			slog.String("method", route.Method),
			slog.String("path", route.Path),
			slog.String("name", route.Name),
		)
	}
}

// RegisterMetricsEndpoint registers the Prometheus metrics endpoint.
func (r *Router) RegisterMetricsEndpoint() {
	// Import is at package level to avoid unused import when metrics aren't needed
	// We use echo.WrapHandler to convert http.Handler to echo.HandlerFunc
	r.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
