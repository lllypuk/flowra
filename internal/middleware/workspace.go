package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Workspace context keys.
const (
	// ContextKeyWorkspaceID is the context key for workspace ID.
	ContextKeyWorkspaceID contextKey = "workspace_id"

	// ContextKeyWorkspaceRole is the context key for user's role in the workspace.
	ContextKeyWorkspaceRole contextKey = "workspace_role"

	// ContextKeyWorkspaceName is the context key for workspace name.
	ContextKeyWorkspaceName contextKey = "workspace_name"
)

// Workspace role constants.
const (
	WorkspaceRoleOwner  = "owner"
	WorkspaceRoleAdmin  = "admin"
	WorkspaceRoleMember = "member"
)

// Workspace errors.
var (
	ErrWorkspaceNotFound   = errors.New("workspace not found")
	ErrNotWorkspaceMember  = errors.New("user is not a member of this workspace")
	ErrInvalidWorkspaceID  = errors.New("invalid workspace ID")
	ErrWorkspaceIDRequired = errors.New("workspace ID is required")
)

// WorkspaceMembership represents a user's membership in a workspace.
type WorkspaceMembership struct {
	// WorkspaceID is the ID of the workspace.
	WorkspaceID uuid.UUID

	// WorkspaceName is the name of the workspace.
	WorkspaceName string

	// UserID is the ID of the user.
	UserID uuid.UUID

	// Role is the user's role in the workspace.
	Role string
}

// WorkspaceAccessChecker defines the interface for checking workspace access.
type WorkspaceAccessChecker interface {
	// GetMembership returns the user's membership in a workspace.
	// Returns nil if the user is not a member.
	GetMembership(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMembership, error)

	// WorkspaceExists checks if a workspace exists.
	WorkspaceExists(ctx context.Context, workspaceID uuid.UUID) (bool, error)
}

// WorkspaceConfig holds configuration for the workspace middleware.
type WorkspaceConfig struct {
	// Logger is the structured logger for workspace events.
	Logger *slog.Logger

	// AccessChecker checks workspace access.
	AccessChecker WorkspaceAccessChecker

	// WorkspaceIDParam is the name of the path parameter containing the workspace ID.
	// Default is "workspace_id".
	WorkspaceIDParam string

	// RequiredRoles specifies which roles are allowed to access the resource.
	// If empty, any member can access.
	RequiredRoles []string

	// AllowSystemAdmin allows system administrators to bypass workspace membership checks.
	AllowSystemAdmin bool
}

// DefaultWorkspaceConfig returns a WorkspaceConfig with sensible defaults.
func DefaultWorkspaceConfig() WorkspaceConfig {
	return WorkspaceConfig{
		Logger:           slog.Default(),
		WorkspaceIDParam: "workspace_id",
		RequiredRoles:    nil,
		AllowSystemAdmin: true,
	}
}

// WorkspaceAccess returns a middleware that verifies workspace access.
//
//nolint:gocognit,funlen // Workspace middleware requires complex access control logic with many edge cases.
func WorkspaceAccess(config WorkspaceConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.WorkspaceIDParam == "" {
		config.WorkspaceIDParam = "workspace_id"
	}

	requiredRoles := make(map[string]struct{}, len(config.RequiredRoles))
	for _, role := range config.RequiredRoles {
		requiredRoles[role] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract workspace ID from path parameter
			workspaceIDStr := c.Param(config.WorkspaceIDParam)
			if workspaceIDStr == "" {
				return respondWorkspaceError(c, ErrWorkspaceIDRequired)
			}

			workspaceID, err := uuid.ParseUUID(workspaceIDStr)
			if err != nil {
				config.Logger.Warn("invalid workspace ID",
					slog.String("workspace_id", workspaceIDStr),
					slog.String("error", err.Error()),
				)
				return respondWorkspaceError(c, ErrInvalidWorkspaceID)
			}

			// Check if system admin bypass is allowed
			//nolint:nestif // System admin bypass requires nested checks for workspace existence
			if config.AllowSystemAdmin && IsSystemAdmin(c) {
				// System admin can access any workspace
				// Still need to verify workspace exists
				if config.AccessChecker != nil {
					exists, existsErr := config.AccessChecker.WorkspaceExists(c.Request().Context(), workspaceID)
					if existsErr != nil {
						config.Logger.Error("failed to check workspace existence",
							slog.String("workspace_id", workspaceID.String()),
							slog.String("error", existsErr.Error()),
						)
						return respondWorkspaceError(c, ErrWorkspaceNotFound)
					}
					if !exists {
						return respondWorkspaceError(c, ErrWorkspaceNotFound)
					}
				}

				c.Set(string(ContextKeyWorkspaceID), workspaceID)
				c.Set(string(ContextKeyWorkspaceRole), WorkspaceRoleAdmin)

				config.Logger.Debug("system admin accessing workspace",
					slog.String("workspace_id", workspaceID.String()),
					slog.String("user_id", GetUserID(c).String()),
				)

				return next(c)
			}

			// Get user ID from context (set by Auth middleware)
			userID := GetUserID(c)
			if userID.IsZero() {
				config.Logger.Warn("user ID not found in context")
				return respondAuthError(c, ErrInsufficientPermissions)
			}

			// Check workspace membership
			if config.AccessChecker == nil {
				config.Logger.Error("access checker not configured")
				return respondWorkspaceError(c, ErrWorkspaceNotFound)
			}

			membership, err := config.AccessChecker.GetMembership(c.Request().Context(), workspaceID, userID)
			if err != nil {
				if errors.Is(err, ErrWorkspaceNotFound) {
					return respondWorkspaceError(c, ErrWorkspaceNotFound)
				}
				config.Logger.Error("failed to check workspace membership",
					slog.String("workspace_id", workspaceID.String()),
					slog.String("user_id", userID.String()),
					slog.String("error", err.Error()),
				)
				return respondWorkspaceError(c, ErrNotWorkspaceMember)
			}

			if membership == nil {
				config.Logger.Debug("user not a member of workspace",
					slog.String("workspace_id", workspaceID.String()),
					slog.String("user_id", userID.String()),
				)
				return respondWorkspaceError(c, ErrNotWorkspaceMember)
			}

			// Check required roles
			if len(requiredRoles) > 0 {
				if _, ok := requiredRoles[membership.Role]; !ok {
					config.Logger.Debug("user lacks required role",
						slog.String("workspace_id", workspaceID.String()),
						slog.String("user_id", userID.String()),
						slog.String("user_role", membership.Role),
					)
					return respondAuthError(c, ErrInsufficientPermissions)
				}
			}

			// Enrich context with workspace information
			c.Set(string(ContextKeyWorkspaceID), membership.WorkspaceID)
			c.Set(string(ContextKeyWorkspaceName), membership.WorkspaceName)
			c.Set(string(ContextKeyWorkspaceRole), membership.Role)

			config.Logger.Debug("workspace access granted",
				slog.String("workspace_id", workspaceID.String()),
				slog.String("user_id", userID.String()),
				slog.String("role", membership.Role),
			)

			return next(c)
		}
	}
}

// respondWorkspaceError sends a workspace-related error response.
func respondWorkspaceError(c echo.Context, err error) error {
	code := "WORKSPACE_ERROR"
	message := "Workspace error"
	status := http.StatusForbidden

	switch {
	case errors.Is(err, ErrWorkspaceNotFound):
		code = "WORKSPACE_NOT_FOUND"
		message = "Workspace not found"
		status = http.StatusNotFound
	case errors.Is(err, ErrNotWorkspaceMember):
		code = "NOT_WORKSPACE_MEMBER"
		message = "You are not a member of this workspace"
		status = http.StatusForbidden
	case errors.Is(err, ErrInvalidWorkspaceID):
		code = "INVALID_WORKSPACE_ID"
		message = "Invalid workspace ID format"
		status = http.StatusBadRequest
	case errors.Is(err, ErrWorkspaceIDRequired):
		code = "WORKSPACE_ID_REQUIRED"
		message = "Workspace ID is required"
		status = http.StatusBadRequest
	}

	return c.JSON(status, map[string]any{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// GetWorkspaceID extracts the workspace ID from the echo context.
func GetWorkspaceID(c echo.Context) uuid.UUID {
	if id, ok := c.Get(string(ContextKeyWorkspaceID)).(uuid.UUID); ok {
		return id
	}
	return uuid.UUID("")
}

// GetWorkspaceName extracts the workspace name from the echo context.
func GetWorkspaceName(c echo.Context) string {
	if name, ok := c.Get(string(ContextKeyWorkspaceName)).(string); ok {
		return name
	}
	return ""
}

// GetWorkspaceRole extracts the user's workspace role from the echo context.
func GetWorkspaceRole(c echo.Context) string {
	if role, ok := c.Get(string(ContextKeyWorkspaceRole)).(string); ok {
		return role
	}
	return ""
}

// IsWorkspaceOwner checks if the current user is the owner of the workspace.
func IsWorkspaceOwner(c echo.Context) bool {
	return GetWorkspaceRole(c) == WorkspaceRoleOwner
}

// IsWorkspaceAdmin checks if the current user is an admin of the workspace.
func IsWorkspaceAdmin(c echo.Context) bool {
	role := GetWorkspaceRole(c)
	return role == WorkspaceRoleOwner || role == WorkspaceRoleAdmin
}

// RequireWorkspaceRole returns a middleware that requires a specific workspace role.
func RequireWorkspaceRole(roles ...string) echo.MiddlewareFunc {
	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := GetWorkspaceRole(c)
			if _, ok := roleSet[userRole]; !ok {
				return respondAuthError(c, ErrInsufficientPermissions)
			}
			return next(c)
		}
	}
}

// RequireWorkspaceAdmin returns a middleware that requires workspace admin or owner role.
func RequireWorkspaceAdmin() echo.MiddlewareFunc {
	return RequireWorkspaceRole(WorkspaceRoleOwner, WorkspaceRoleAdmin)
}

// RequireWorkspaceOwner returns a middleware that requires workspace owner role.
func RequireWorkspaceOwner() echo.MiddlewareFunc {
	return RequireWorkspaceRole(WorkspaceRoleOwner)
}

// MockWorkspaceAccessChecker is a mock implementation for testing.
type MockWorkspaceAccessChecker struct {
	memberships map[string]*WorkspaceMembership // key: "workspaceID:userID"
	workspaces  map[uuid.UUID]bool
}

// NewMockWorkspaceAccessChecker creates a new mock access checker.
func NewMockWorkspaceAccessChecker() *MockWorkspaceAccessChecker {
	return &MockWorkspaceAccessChecker{
		memberships: make(map[string]*WorkspaceMembership),
		workspaces:  make(map[uuid.UUID]bool),
	}
}

// AddMembership adds a membership to the mock.
func (m *MockWorkspaceAccessChecker) AddMembership(membership *WorkspaceMembership) {
	key := membership.WorkspaceID.String() + ":" + membership.UserID.String()
	m.memberships[key] = membership
	m.workspaces[membership.WorkspaceID] = true
}

// AddWorkspace adds a workspace to the mock.
func (m *MockWorkspaceAccessChecker) AddWorkspace(workspaceID uuid.UUID) {
	m.workspaces[workspaceID] = true
}

// GetMembership returns the user's membership in a workspace.
// Returns (nil, nil) if the user is not a member but the workspace exists.
//
//nolint:nilnil // nil, nil is a valid return to indicate "not a member" without error
func (m *MockWorkspaceAccessChecker) GetMembership(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (*WorkspaceMembership, error) {
	if _, exists := m.workspaces[workspaceID]; !exists {
		return nil, ErrWorkspaceNotFound
	}

	key := workspaceID.String() + ":" + userID.String()
	membership, ok := m.memberships[key]
	if !ok {
		return nil, nil
	}
	return membership, nil
}

// WorkspaceExists checks if a workspace exists.
func (m *MockWorkspaceAccessChecker) WorkspaceExists(_ context.Context, workspaceID uuid.UUID) (bool, error) {
	exists := m.workspaces[workspaceID]
	return exists, nil
}
