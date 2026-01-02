// Package main provides the API server entry point.
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Health status constants.
const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
	statusReady     = "ready"
	statusNotReady  = "not ready"
)

// SetupRoutes configures all API routes and middleware chains.
func SetupRoutes(c *Container) *httpserver.Router {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Create router configuration
	routerConfig := httpserver.RouterConfig{
		Logger: c.Logger,
		AuthMiddleware: middleware.Auth(middleware.AuthConfig{
			Logger:         c.Logger,
			TokenValidator: c.TokenValidator,
			SkipPaths: []string{
				"/health",
				"/ready",
				"/api/v1/auth/login",
				"/api/v1/auth/register",
			},
			AllowExpiredForPaths: []string{
				"/api/v1/auth/refresh",
			},
		}),
		WorkspaceMiddleware: middleware.WorkspaceAccess(middleware.WorkspaceConfig{
			Logger:           c.Logger,
			AccessChecker:    c.AccessChecker,
			WorkspaceIDParam: "workspace_id",
			AllowSystemAdmin: true,
		}),
		CORSConfig:     middleware.DefaultCORSConfig(),
		LoggingConfig:  middleware.DefaultLoggingConfig(),
		RecoveryConfig: middleware.DefaultRecoveryConfig(),
		APIPrefix:      "/api/v1",
	}

	// Create router with configuration
	router := httpserver.NewRouter(e, routerConfig)

	// Register health check endpoints
	router.RegisterHealthEndpoints(func() bool {
		return c.IsReady(e.AcquireContext().Request().Context())
	})

	// Register API routes
	registerAuthRoutes(router, c)
	registerWorkspaceRoutes(router, c)
	registerChatRoutes(router, c)
	registerMessageRoutes(router, c)
	registerTaskRoutes(router, c)
	registerNotificationRoutes(router, c)
	registerUserRoutes(router, c)
	registerWebSocketRoutes(router, c)

	// Log all registered routes in debug mode
	if c.Config.IsDevelopment() {
		router.PrintRoutes()
	}

	return router
}

// registerAuthRoutes registers authentication-related routes.
func registerAuthRoutes(r *httpserver.Router, c *Container) {
	// Public auth routes
	r.Public().POST("/auth/login", c.AuthHandler.Login)

	// Authenticated auth routes
	r.Auth().POST("/auth/logout", c.AuthHandler.Logout)
	r.Auth().POST("/auth/refresh", c.AuthHandler.Refresh)
	r.Auth().GET("/auth/me", c.AuthHandler.Me)
}

// registerWorkspaceRoutes registers workspace-related routes.
func registerWorkspaceRoutes(r *httpserver.Router, c *Container) {
	// Workspace list and create (authenticated but not workspace-scoped)
	r.Auth().POST("/workspaces", c.WorkspaceHandler.Create)
	r.Auth().GET("/workspaces", c.WorkspaceHandler.List)

	// Workspace-scoped routes
	ws := r.Workspace()
	ws.GET("", c.WorkspaceHandler.Get)
	ws.PUT("", c.WorkspaceHandler.Update)
	ws.DELETE("", c.WorkspaceHandler.Delete, middleware.RequireWorkspaceOwner())

	// Workspace member management
	ws.POST("/members", c.WorkspaceHandler.AddMember, middleware.RequireWorkspaceAdmin())
	ws.DELETE("/members/:user_id", c.WorkspaceHandler.RemoveMember, middleware.RequireWorkspaceAdmin())
	ws.PUT("/members/:user_id/role", c.WorkspaceHandler.UpdateMemberRole, middleware.RequireWorkspaceAdmin())
}

// registerChatRoutes registers chat-related routes.
func registerChatRoutes(r *httpserver.Router, c *Container) {
	chats := r.NewWorkspaceRouteGroup("/chats")

	// Chat CRUD
	chats.POST("", c.ChatHandler.Create)
	chats.GET("", c.ChatHandler.List)
	chats.GET("/:chat_id", c.ChatHandler.Get)
	chats.PUT("/:chat_id", c.ChatHandler.Update)
	chats.DELETE("/:chat_id", c.ChatHandler.Delete)

	// Chat participants
	chats.POST("/:chat_id/participants", c.ChatHandler.AddParticipant)
	chats.DELETE("/:chat_id/participants/:user_id", c.ChatHandler.RemoveParticipant)
}

// registerMessageRoutes registers message-related routes.
func registerMessageRoutes(r *httpserver.Router, c *Container) {
	// Messages are workspace-scoped through chat_id
	messages := r.NewWorkspaceRouteGroup("/chats/:chat_id/messages")

	if c.MessageHandler != nil {
		messages.POST("", c.MessageHandler.Send)
		messages.GET("", c.MessageHandler.List)
		messages.PUT("/:message_id", c.MessageHandler.Edit)
		messages.DELETE("/:message_id", c.MessageHandler.Delete)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := func(ctx echo.Context) error {
			return ctx.JSON(http.StatusNotImplemented, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "NOT_IMPLEMENTED",
					"message": "Message service not available",
				},
			})
		}
		messages.POST("", placeholder)
		messages.GET("", placeholder)
		messages.PUT("/:message_id", placeholder)
		messages.DELETE("/:message_id", placeholder)
	}
}

// registerTaskRoutes registers task-related routes.
func registerTaskRoutes(r *httpserver.Router, c *Container) {
	tasks := r.NewWorkspaceRouteGroup("/tasks")

	if c.TaskHandler != nil {
		tasks.POST("", c.TaskHandler.Create)
		tasks.GET("", c.TaskHandler.List)
		tasks.GET("/:task_id", c.TaskHandler.Get)
		tasks.PUT("/:task_id/status", c.TaskHandler.ChangeStatus)
		tasks.PUT("/:task_id/assignee", c.TaskHandler.Assign)
		tasks.PUT("/:task_id/priority", c.TaskHandler.ChangePriority)
		tasks.PUT("/:task_id/due-date", c.TaskHandler.SetDueDate)
		tasks.DELETE("/:task_id", c.TaskHandler.Delete)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := func(ctx echo.Context) error {
			return ctx.JSON(http.StatusNotImplemented, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "NOT_IMPLEMENTED",
					"message": "Task service not available",
				},
			})
		}
		tasks.POST("", placeholder)
		tasks.GET("", placeholder)
		tasks.GET("/:task_id", placeholder)
		tasks.PUT("/:task_id/status", placeholder)
		tasks.PUT("/:task_id/assignee", placeholder)
		tasks.PUT("/:task_id/priority", placeholder)
		tasks.PUT("/:task_id/due-date", placeholder)
		tasks.DELETE("/:task_id", placeholder)
	}
}

// registerNotificationRoutes registers notification-related routes.
func registerNotificationRoutes(r *httpserver.Router, c *Container) {
	if c.NotificationHandler != nil {
		// Notifications are user-scoped, not workspace-scoped
		r.Auth().GET("/notifications", c.NotificationHandler.List)
		r.Auth().GET("/notifications/unread/count", c.NotificationHandler.UnreadCount)
		r.Auth().PUT("/notifications/:id/read", c.NotificationHandler.MarkAsRead)
		r.Auth().PUT("/notifications/mark-all-read", c.NotificationHandler.MarkAllRead)
		r.Auth().DELETE("/notifications/:id", c.NotificationHandler.Delete)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := func(ctx echo.Context) error {
			return ctx.JSON(http.StatusNotImplemented, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "NOT_IMPLEMENTED",
					"message": "Notification service not available",
				},
			})
		}
		r.Auth().GET("/notifications", placeholder)
		r.Auth().GET("/notifications/unread/count", placeholder)
		r.Auth().PUT("/notifications/:id/read", placeholder)
		r.Auth().PUT("/notifications/mark-all-read", placeholder)
		r.Auth().DELETE("/notifications/:id", placeholder)
	}
}

// registerUserRoutes registers user-related routes.
func registerUserRoutes(r *httpserver.Router, c *Container) {
	if c.UserHandler != nil {
		r.Auth().GET("/users/me", c.UserHandler.GetMe)
		r.Auth().PUT("/users/me", c.UserHandler.UpdateMe)
		r.Auth().GET("/users/:id", c.UserHandler.Get)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := func(ctx echo.Context) error {
			return ctx.JSON(http.StatusNotImplemented, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "NOT_IMPLEMENTED",
					"message": "User service not available",
				},
			})
		}
		r.Auth().GET("/users/me", placeholder)
		r.Auth().PUT("/users/me", placeholder)
		r.Auth().GET("/users/:id", placeholder)
	}
}

// registerWebSocketRoutes registers WebSocket routes.
func registerWebSocketRoutes(r *httpserver.Router, c *Container) {
	// WebSocket endpoint requires authentication
	r.Auth().GET("/ws", c.WSHandler.HandleWebSocket)
}

// SetupHealthEndpoints adds health check endpoints directly to an Echo instance.
// This is useful when you need to add health checks before full routing is set up.
func SetupHealthEndpoints(e *echo.Echo, c *Container) {
	// Liveness probe - checks if the application is running
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{
			"status": statusHealthy,
		})
	})

	// Readiness probe - checks if the application is ready to serve traffic
	e.GET("/ready", func(ctx echo.Context) error {
		if c.IsReady(ctx.Request().Context()) {
			return ctx.JSON(http.StatusOK, map[string]any{
				"status":     statusReady,
				"components": c.GetHealthStatus(ctx.Request().Context()),
			})
		}
		return ctx.JSON(http.StatusServiceUnavailable, map[string]any{
			"status":     statusNotReady,
			"components": c.GetHealthStatus(ctx.Request().Context()),
		})
	})

	// Detailed health status endpoint
	e.GET("/health/details", func(ctx echo.Context) error {
		statuses := c.GetHealthStatus(ctx.Request().Context())

		allHealthy := true
		for _, s := range statuses {
			if s.Status != statusHealthy {
				allHealthy = false
				break
			}
		}

		statusCode := http.StatusOK
		overallStatus := statusHealthy
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
			overallStatus = statusUnhealthy
		}

		return ctx.JSON(statusCode, map[string]any{
			"status":     overallStatus,
			"components": statuses,
		})
	})
}
