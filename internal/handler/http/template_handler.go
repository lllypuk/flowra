package httphandler

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
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

// TemplateHandler provides handlers for rendering HTML pages.
type TemplateHandler struct {
	renderer *TemplateRenderer
	logger   *slog.Logger
}

// NewTemplateHandler creates a new template handler.
func NewTemplateHandler(renderer *TemplateRenderer, logger *slog.Logger) *TemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TemplateHandler{
		renderer: renderer,
		logger:   logger,
	}
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

	// Build auth URL (for now using a placeholder)
	// When real Keycloak is integrated, this will be the actual OAuth URL
	authURL := "/auth/callback?code=mock-code&state=" + state

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

	// For mock mode, create a simple session
	// In production, this would exchange the code for tokens with Keycloak
	if code == "mock-code" {
		// Create a mock session (in real implementation, call auth service)
		const mockExpiresIn = 3600 // 1 hour
		setSessionCookie(c, "mock-session-token", mockExpiresIn)

		// Get redirect URL or default to workspaces
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

	// These will be implemented in subsequent tasks
	// e.GET("/workspaces", h.WorkspaceList)
	// e.GET("/workspaces/:id", h.WorkspaceView)
	// etc.
}
