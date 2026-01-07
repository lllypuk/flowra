package httphandler

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Template handler constants.
const (
	defaultPageLimit = 100
)

// TemplateRenderer implements echo.Renderer for HTML template rendering.
type TemplateRenderer struct {
	templates *template.Template
	mu        sync.RWMutex
	logger    *slog.Logger
	devMode   bool
	fs        embed.FS
}

// TemplateRendererConfig holds configuration for the template renderer.
type TemplateRendererConfig struct {
	// FS is the embedded filesystem containing templates.
	FS embed.FS
	// Logger is the structured logger.
	Logger *slog.Logger
	// DevMode enables template reloading on each request.
	DevMode bool
}

// NewTemplateRenderer creates a new template renderer.
func NewTemplateRenderer(cfg TemplateRendererConfig) (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		logger:  cfg.Logger,
		devMode: cfg.DevMode,
		fs:      cfg.FS,
	}

	if r.logger == nil {
		r.logger = slog.Default()
	}

	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// loadTemplates parses all templates from the embedded filesystem.
func (r *TemplateRenderer) loadTemplates() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl := template.New("").Funcs(TemplateFuncs())

	// Walk through all template files
	err := fs.WalkDir(r.fs, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}

		// Read template content
		content, readErr := r.fs.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		// Parse template with relative name
		name := path[len("templates/"):]
		_, parseErr := tmpl.New(name).Parse(string(content))
		if parseErr != nil {
			r.logger.Error("failed to parse template",
				slog.String("path", path),
				slog.String("error", parseErr.Error()))
			return parseErr
		}

		r.logger.Debug("loaded template", slog.String("name", name))
		return nil
	})

	if err != nil {
		return err
	}

	r.templates = tmpl
	return nil
}

// Render implements echo.Renderer.
func (r *TemplateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	// In dev mode, reload templates on each request
	if r.devMode {
		if err := r.loadTemplates(); err != nil {
			r.logger.Error("failed to reload templates", slog.String("error", err.Error()))
			return fmt.Errorf("failed to reload templates: %w", err)
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.templates.ExecuteTemplate(w, name, data)
}

// Flash represents flash message data for templates.
type Flash struct {
	Success []string
	Error   []string
	Info    []string
	Warning []string
}

// PageData represents common data passed to all page templates.
type PageData struct {
	Title string
	User  *UserView
	Flash *Flash
	Data  any
	Meta  map[string]string
}

// UserView represents user data for templates.
type UserView struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
	AvatarURL   string
}

// OAuthClient defines the interface for OAuth operations.
type OAuthClient interface {
	// AuthorizationURL generates the OAuth authorization URL.
	AuthorizationURL(redirectURI, state string) string
	// ExchangeCode exchanges an authorization code for tokens.
	ExchangeCode(ctx context.Context, code, redirectURI string) (*OAuthTokenResponse, error)
}

// OAuthTokenResponse represents OAuth token response.
type OAuthTokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// TemplateHandler provides handlers for rendering HTML pages.
type TemplateHandler struct {
	renderer         *TemplateRenderer
	logger           *slog.Logger
	workspaceService WorkspaceService
	memberService    MemberService
	oauthClient      OAuthClient
}

// NewTemplateHandler creates a new template handler.
func NewTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	workspaceService WorkspaceService,
	memberService MemberService,
) *TemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TemplateHandler{
		renderer:         renderer,
		logger:           logger,
		workspaceService: workspaceService,
		memberService:    memberService,
	}
}

// SetServices sets the workspace and member services.
// This is used to inject services after the handler is created.
func (h *TemplateHandler) SetServices(workspaceService WorkspaceService, memberService MemberService) {
	h.workspaceService = workspaceService
	h.memberService = memberService
}

// SetOAuthClient sets the OAuth client for authentication.
func (h *TemplateHandler) SetOAuthClient(client OAuthClient) {
	h.oauthClient = client
}

// render is a helper to render a template with common page data.
func (h *TemplateHandler) render(c echo.Context, templateName string, title string, data any) error {
	pageData := PageData{
		Title: title,
		User:  h.getUserView(c),
		Flash: h.getFlash(c),
		Data:  data,
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, pageData, c)
}

// RenderPartial renders a template without the base layout.
// This is used for HTMX partial updates.
func (h *TemplateHandler) RenderPartial(c echo.Context, templateName string, data any) error {
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, data, c)
}

// getUserView extracts user information from the context for templates.
func (h *TemplateHandler) getUserView(c echo.Context) *UserView {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return nil
	}

	// Get user from context if available
	user := c.Get("user")
	if user == nil {
		// Return minimal user view with just ID
		return &UserView{
			ID: userID.String(),
		}
	}

	// Try to extract user details from context
	// This assumes the auth middleware populates user details
	if userMap, ok := user.(map[string]any); ok {
		return &UserView{
			ID:          getString(userMap, "id"),
			Email:       getString(userMap, "email"),
			Username:    getString(userMap, "username"),
			DisplayName: getString(userMap, "display_name"),
			AvatarURL:   getString(userMap, "avatar_url"),
		}
	}

	return &UserView{
		ID: userID.String(),
	}
}

// getFlash retrieves flash messages from the session.
func (h *TemplateHandler) getFlash(_ echo.Context) *Flash {
	// Flash messages can be stored in cookies or session
	// For now, return nil - will be implemented with session management
	return nil
}

// getString safely extracts a string from a map.
func getString(m map[string]any, key string) string {
	if v, found := m[key]; found {
		if s, isStr := v.(string); isStr {
			return s
		}
	}
	return ""
}

// Home renders the home page.
func (h *TemplateHandler) Home(c echo.Context) error {
	return h.render(c, "home.html", "Home", nil)
}

// LoginPage renders the login page with OAuth auth URL.
func (h *TemplateHandler) LoginPage(c echo.Context) error {
	// Redirect if already logged in
	token := getSessionCookie(c)
	if token != "" {
		return c.Redirect(http.StatusFound, "/workspaces")
	}

	// Generate state for CSRF protection
	state := generateState()
	setStateCookie(c, state)

	// Build auth URL
	var authURL string
	redirectURI := GetRedirectURI(c)

	if h.oauthClient != nil {
		// Use real Keycloak OAuth URL
		authURL = h.oauthClient.AuthorizationURL(redirectURI, state)
	} else {
		// Fallback to mock for development without Keycloak
		h.logger.Warn("OAuth client not configured, using mock auth flow")
		authURL = "/auth/callback?code=mock-code&state=" + state
	}

	// Template expects AuthURL and Error at top level
	data := map[string]any{
		"Title":   "Login",
		"AuthURL": authURL,
		"Error":   c.QueryParam("error"),
		"User":    nil, // Not logged in
	}

	return h.renderer.Render(c.Response().Writer, "auth/login.html", data, c)
}

// AuthCallback handles OAuth callback from the authentication provider.
func (h *TemplateHandler) AuthCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	errorParam := c.QueryParam("error")

	// Check for OAuth error
	if errorParam != "" {
		return h.renderCallback(c, "", "Authentication failed: "+errorParam)
	}

	// Validate state parameter (CSRF protection)
	expectedState := getStateCookie(c)
	clearStateCookie(c)

	if state != expectedState || expectedState == "" {
		return h.renderCallback(c, "", "Invalid state parameter. Please try again.")
	}

	// Exchange code for tokens
	if h.oauthClient != nil {
		// Real OAuth flow with Keycloak
		redirectURI := GetRedirectURI(c)
		tokens, err := h.oauthClient.ExchangeCode(c.Request().Context(), code, redirectURI)
		if err != nil {
			h.logger.Error("failed to exchange code for tokens",
				slog.String("error", err.Error()),
			)
			return h.renderCallback(c, "", "Authentication failed. Please try again.")
		}

		// Store access token in session cookie
		setSessionCookie(c, tokens.AccessToken, tokens.ExpiresIn)

		// Get redirect URL or default to workspaces
		redirectURL := getRedirectCookie(c)
		if redirectURL == "" {
			redirectURL = "/workspaces"
		}
		clearRedirectCookie(c)

		return h.renderCallback(c, redirectURL, "")
	}

	// Fallback mock mode (only when OAuth client is not configured)
	if code == "mock-code" {
		h.logger.Warn("using mock authentication flow")
		const mockExpiresIn = 3600 // 1 hour
		setSessionCookie(c, "mock-session-token", mockExpiresIn)

		redirectURL := getRedirectCookie(c)
		if redirectURL == "" {
			redirectURL = "/workspaces"
		}
		clearRedirectCookie(c)

		return h.renderCallback(c, redirectURL, "")
	}

	return h.renderCallback(c, "", "Invalid authentication code")
}

// renderCallback renders the OAuth callback page.
func (h *TemplateHandler) renderCallback(c echo.Context, redirectURL, errorMsg string) error {
	data := map[string]any{
		"Title":       "Signing In",
		"RedirectURL": redirectURL,
		"Error":       errorMsg,
		"User":        nil, // Not logged in yet during callback
	}
	return h.renderer.Render(c.Response().Writer, "auth/callback.html", data, c)
}

// LogoutPage renders the logout confirmation page.
func (h *TemplateHandler) LogoutPage(c echo.Context) error {
	data := map[string]any{
		"Title": "Sign Out",
		"User":  h.getUserView(c),
	}
	return h.renderer.Render(c.Response().Writer, "auth/logout.html", data, c)
}

// LogoutHandler handles the logout action.
func (h *TemplateHandler) LogoutHandler(c echo.Context) error {
	// Clear session cookie
	clearSessionCookie(c)
	clearRedirectCookie(c)

	// For HTMX requests, return success with redirect header
	//nolint:canonicalheader // HTMX uses non-canonical header names
	if c.Request().Header.Get("HX-Request") == "true" {
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Redirect", "/")
		return c.NoContent(http.StatusOK)
	}

	// For regular requests, redirect to home
	return c.Redirect(http.StatusFound, "/")
}

// NotFound renders the 404 page.
func (h *TemplateHandler) NotFound(c echo.Context) error {
	return c.Render(http.StatusNotFound, "layout/base.html", PageData{
		Title: "Page Not Found",
		User:  h.getUserView(c),
		Data: map[string]string{
			"message": "The page you're looking for doesn't exist.",
		},
	})
}

// ServerError renders the 500 page.
func (h *TemplateHandler) ServerError(c echo.Context, err error) error {
	h.logger.Error("server error", slog.String("error", err.Error()))
	return c.Render(http.StatusInternalServerError, "layout/base.html", PageData{
		Title: "Server Error",
		User:  h.getUserView(c),
		Data: map[string]string{
			"message": "Something went wrong. Please try again later.",
		},
	})
}

// SetupStaticRoutes registers routes for serving static files.
func SetupStaticRoutes(e *echo.Echo, staticFS embed.FS) error {
	// Extract static subdirectory
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}

	// Serve static files
	e.StaticFS("/static", staticSub)

	return nil
}

// SetupPageRoutes registers HTML page routes.
func (h *TemplateHandler) SetupPageRoutes(e *echo.Echo) {
	// Public pages
	e.GET("/", h.Home)
	e.GET("/login", h.LoginPage)

	// Auth pages
	e.GET("/auth/callback", h.AuthCallback)
	e.GET("/logout", h.LogoutPage)
	e.POST("/auth/logout", h.LogoutHandler)

	// Workspace pages (protected)
	workspaces := e.Group("/workspaces", RequireAuth)
	workspaces.GET("", h.WorkspaceList)
	workspaces.GET("/:id", h.WorkspaceView)
	workspaces.GET("/:id/members", h.WorkspaceMembers)
	workspaces.GET("/:id/settings", h.WorkspaceSettings)

	// Workspace partials (protected)
	partials := e.Group("/partials", RequireAuth)
	partials.GET("/workspaces", h.WorkspaceListPartial)
	partials.GET("/workspace/create-form", h.WorkspaceCreateForm)
	partials.POST("/workspace/create", h.WorkspaceCreate)
	partials.GET("/workspace/:id/members", h.WorkspaceMembersPartial)
	partials.GET("/workspace/:id/invite-form", h.WorkspaceInviteForm)
}

// WorkspaceList renders the workspace list page.
func (h *TemplateHandler) WorkspaceList(c echo.Context) error {
	return h.render(c, "workspace/list.html", "Workspaces", nil)
}

// WorkspaceListPartial returns the workspace list as HTML partial for HTMX.
func (h *TemplateHandler) WorkspaceListPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// Check if services are available
	if h.workspaceService == nil {
		return h.RenderPartial(c, "empty-workspaces", nil)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.RenderPartial(c, "empty-workspaces", nil)
	}

	workspaces, _, err := h.workspaceService.ListUserWorkspaces(c.Request().Context(), userID, 0, defaultPageLimit)
	if err != nil {
		h.logger.Error("failed to list workspaces", slog.String("error", err.Error()))
		return h.RenderPartial(c, "empty-workspaces", nil)
	}

	// Convert to view models
	workspaceViews := make([]WorkspaceViewData, 0, len(workspaces))
	for _, ws := range workspaces {
		memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), ws.ID())
		workspaceViews = append(workspaceViews, WorkspaceViewData{
			ID:          ws.ID().String(),
			Name:        ws.Name(),
			Description: "", // Description not in domain model yet
			MemberCount: memberCount,
			CreatedAt:   ws.CreatedAt(),
			UnreadCount: 0, // TODO: implement unread count
		})
	}

	data := map[string]any{
		"Workspaces": workspaceViews,
	}
	return h.RenderPartial(c, "workspace/list-partial", data)
}

// WorkspaceView renders a single workspace page.
func (h *TemplateHandler) WorkspaceView(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Check if services are available
	if h.workspaceService == nil || h.memberService == nil {
		return h.NotFound(c)
	}

	workspaceID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return h.NotFound(c)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.NotFound(c)
	}

	ws, err := h.workspaceService.GetWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		return h.NotFound(c)
	}

	member, err := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if err != nil {
		return h.NotFound(c)
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), workspaceID)

	data := map[string]any{
		"Workspace": WorkspaceViewData{
			ID:          ws.ID().String(),
			Name:        ws.Name(),
			Description: "",
			MemberCount: memberCount,
			CreatedAt:   ws.CreatedAt(),
		},
		"UserRole":    member.Role().String(),
		"ActiveTab":   c.QueryParam("tab"),
		"UnreadChats": 0,
	}

	return h.render(c, "workspace/view.html", ws.Name(), data)
}

// WorkspaceCreateForm returns the create workspace form partial.
func (h *TemplateHandler) WorkspaceCreateForm(c echo.Context) error {
	return h.RenderPartial(c, "workspace/create-form", nil)
}

// WorkspaceCreate handles POST /partials/workspace/create and returns HTML partial.
func (h *TemplateHandler) WorkspaceCreate(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Retarget", "#modal-container")
		return c.String(http.StatusUnauthorized, `<div class="error">Unauthorized</div>`)
	}

	if h.workspaceService == nil {
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Retarget", "#modal-container")
		return c.String(http.StatusServiceUnavailable, `<div class="error">Service unavailable</div>`)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Retarget", "#modal-container")
		return c.String(http.StatusBadRequest, `<div class="error">Invalid user ID</div>`)
	}

	name := c.FormValue("name")
	description := c.FormValue("description")

	if name == "" {
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Retarget", "#modal-container")
		return c.String(http.StatusBadRequest, `<div class="error">Workspace name is required</div>`)
	}

	ws, err := h.workspaceService.CreateWorkspace(c.Request().Context(), userID, name, description)
	if err != nil {
		h.logger.Error("failed to create workspace", slog.String("error", err.Error()))
		//nolint:canonicalheader // HTMX uses non-canonical header names
		c.Response().Header().Set("HX-Retarget", "#modal-container")
		return c.String(http.StatusInternalServerError, `<div class="error">Failed to create workspace</div>`)
	}

	// Return the workspace card HTML partial
	data := WorkspaceViewData{
		ID:          ws.ID().String(),
		Name:        ws.Name(),
		Description: description,
		MemberCount: 1, // Owner is the first member
		CreatedAt:   ws.CreatedAt(),
		UnreadCount: 0,
	}

	return h.RenderPartial(c, "workspace_card", data)
}

// WorkspaceMembers renders the workspace members page.
func (h *TemplateHandler) WorkspaceMembers(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Check if services are available
	if h.workspaceService == nil || h.memberService == nil {
		return h.NotFound(c)
	}

	workspaceID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return h.NotFound(c)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.NotFound(c)
	}

	ws, err := h.workspaceService.GetWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		return h.NotFound(c)
	}

	member, err := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if err != nil {
		return h.NotFound(c)
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), workspaceID)

	data := map[string]any{
		"Workspace": WorkspaceViewData{
			ID:          ws.ID().String(),
			Name:        ws.Name(),
			Description: "",
			MemberCount: memberCount,
			CreatedAt:   ws.CreatedAt(),
		},
		"UserRole":      member.Role().String(),
		"CurrentUserID": user.ID,
	}

	return h.render(c, "workspace/members.html", "Members - "+ws.Name(), data)
}

// WorkspaceMembersPartial returns the workspace members list as HTML partial.
func (h *TemplateHandler) WorkspaceMembersPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// Check if services are available
	if h.workspaceService == nil || h.memberService == nil {
		return c.String(http.StatusServiceUnavailable, "Service unavailable")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid workspace ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	ws, err := h.workspaceService.GetWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		return c.String(http.StatusNotFound, "Workspace not found")
	}

	currentMember, err := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if err != nil {
		return c.String(http.StatusForbidden, "Not a member of this workspace")
	}

	members, _, err := h.memberService.ListMembers(c.Request().Context(), workspaceID, 0, defaultPageLimit)
	if err != nil {
		h.logger.Error("failed to list members", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to load members")
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), workspaceID)

	// Convert to view models
	memberViews := make([]MemberViewData, 0, len(members))
	for _, m := range members {
		memberViews = append(memberViews, MemberViewData{
			UserID:      m.UserID().String(),
			Username:    "user" + m.UserID().String()[:8], // TODO: get actual username
			DisplayName: "User " + m.UserID().String()[:8],
			AvatarURL:   "",
			Role:        m.Role().String(),
			JoinedAt:    m.JoinedAt(),
		})
	}

	data := map[string]any{
		"Members": memberViews,
		"Workspace": WorkspaceViewData{
			ID:          ws.ID().String(),
			Name:        ws.Name(),
			MemberCount: memberCount,
		},
		"UserRole":      currentMember.Role().String(),
		"CurrentUserID": user.ID,
	}

	return h.RenderPartial(c, "workspace/members-partial", data)
}

// WorkspaceSettings renders the workspace settings page.
func (h *TemplateHandler) WorkspaceSettings(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Check if services are available
	if h.workspaceService == nil || h.memberService == nil {
		return h.NotFound(c)
	}

	workspaceID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return h.NotFound(c)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.NotFound(c)
	}

	ws, err := h.workspaceService.GetWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		return h.NotFound(c)
	}

	member, err := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if err != nil {
		return h.NotFound(c)
	}

	// Only owner can access settings
	if member.Role().String() != "owner" {
		return c.Redirect(http.StatusFound, "/workspaces/"+workspaceID.String())
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), workspaceID)

	data := map[string]any{
		"Workspace": WorkspaceViewData{
			ID:          ws.ID().String(),
			Name:        ws.Name(),
			Description: "",
			MemberCount: memberCount,
			CreatedAt:   ws.CreatedAt(),
		},
		"UserRole": member.Role().String(),
	}

	return h.render(c, "workspace/settings.html", "Settings - "+ws.Name(), data)
}

// WorkspaceInviteForm returns the invite member form partial.
func (h *TemplateHandler) WorkspaceInviteForm(c echo.Context) error {
	workspaceID := c.Param("id")
	data := map[string]any{
		"WorkspaceID": workspaceID,
	}
	return h.RenderPartial(c, "workspace/invite-form", data)
}

// WorkspaceViewData represents workspace data for templates.
type WorkspaceViewData struct {
	ID          string
	Name        string
	Description string
	MemberCount int
	CreatedAt   time.Time
	UnreadCount int
}

// MemberViewData represents member data for templates.
type MemberViewData struct {
	UserID      string
	Username    string
	DisplayName string
	AvatarURL   string
	Role        string
	JoinedAt    time.Time
}
