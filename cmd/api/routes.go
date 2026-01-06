// Package main provides the API server entry point.
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/web"
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
				"/health/details",
				"/api/v1/auth/login",
				"/api/v1/auth/register",
			},
			AllowExpiredForPaths: []string{
				"/api/v1/auth/refresh",
			},
			// Session cookie support for HTMX frontend
			SessionCookieName: "flowra_session",
			MockSessionToken:  "mock-session-token",
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

	// Setup template renderer for HTML pages
	e.Renderer = c.TemplateRenderer

	// Setup static file serving
	if err := httphandler.SetupStaticRoutes(e, web.StaticFS); err != nil {
		c.Logger.Error("failed to setup static routes", "error", err)
	}

	// Register health check endpoints using the HealthChecker interface.
	// Container implements httpserver.HealthChecker, so we pass it directly.
	// This ensures proper context handling from the request.
	router.RegisterHealthEndpointsWithChecker(c)

	// Register HTML page routes
	registerPageRoutes(e, c)

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
		placeholder := createPlaceholderHandler("Message")
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
		placeholder := createPlaceholderHandler("Task")
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
		placeholder := createPlaceholderHandler("Notification")
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
		placeholder := createPlaceholderHandler("User")
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

// createPlaceholderHandler creates a handler that returns 501 Not Implemented.
// This is used for endpoints where the handler is not yet available.
func createPlaceholderHandler(serviceName string) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return ctx.JSON(http.StatusNotImplemented, map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "NOT_IMPLEMENTED",
				"message": serviceName + " service not available",
			},
		})
	}
}

// registerPageRoutes registers HTML page routes for the HTMX frontend.
func registerPageRoutes(e *echo.Echo, c *Container) {
	// Public pages
	e.GET("/", c.TemplateHandler.Home)

	// Auth pages (public)
	e.GET("/login", c.TemplateHandler.LoginPage)
	e.GET("/auth/callback", c.TemplateHandler.AuthCallback)

	// Auth actions
	e.GET("/logout", httphandler.RequireAuth(c.TemplateHandler.LogoutPage))
	e.POST("/auth/logout", c.TemplateHandler.LogoutHandler)

	// Protected pages (require authentication)
	// Workspace pages
	workspaces := e.Group("/workspaces", httphandler.RequireAuth)
	workspaces.GET("", c.TemplateHandler.WorkspaceList)
	workspaces.GET("/:id", c.TemplateHandler.WorkspaceView)
	workspaces.GET("/:id/members", c.TemplateHandler.WorkspaceMembers)
	workspaces.GET("/:id/settings", c.TemplateHandler.WorkspaceSettings)

	// Workspace partials (for HTMX)
	partials := e.Group("/partials", httphandler.RequireAuth)
	partials.GET("/workspaces", c.TemplateHandler.WorkspaceListPartial)
	partials.GET("/workspace/create-form", c.TemplateHandler.WorkspaceCreateForm)
	partials.GET("/workspace/:id/members", c.TemplateHandler.WorkspaceMembersPartial)
	partials.GET("/workspace/:id/invite-form", c.TemplateHandler.WorkspaceInviteForm)

	// TODO: Add more protected pages as frontend features are implemented:
	// - /workspaces/:id/chats/:chat_id (chat view)
	// - /workspaces/:id/board (kanban board)
	// - /settings (user settings)
}
