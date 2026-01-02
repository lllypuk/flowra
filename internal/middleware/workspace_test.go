package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWorkspaceConfig(t *testing.T) {
	config := middleware.DefaultWorkspaceConfig()

	assert.NotNil(t, config.Logger)
	assert.Equal(t, "workspace_id", config.WorkspaceIDParam)
	assert.Nil(t, config.RequiredRoles)
	assert.True(t, config.AllowSystemAdmin)
}

func TestWorkspaceAccess_MissingWorkspaceID(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	// Request without workspace_id parameter
	req := httptest.NewRequest(http.MethodGet, "/workspaces/", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Echo will return 404 for missing param, but if somehow hit, should return error
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestWorkspaceAccess_InvalidWorkspaceID(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	// Set up auth context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/invalid-uuid", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "INVALID_WORKSPACE_ID")
}

func TestWorkspaceAccess_WorkspaceNotFound(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	// Set up auth context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "WORKSPACE_NOT_FOUND")
}

func TestWorkspaceAccess_NotMember(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	// Add workspace but not membership
	accessChecker.AddWorkspace(workspaceID)

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "NOT_WORKSPACE_MEMBER")
}

func TestWorkspaceAccess_ValidMember(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	accessChecker.AddMembership(&middleware.WorkspaceMembership{
		WorkspaceID:   workspaceID,
		WorkspaceName: "Test Workspace",
		UserID:        userID,
		Role:          middleware.WorkspaceRoleMember,
	})

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	var capturedWorkspaceID uuid.UUID
	var capturedRole string

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		capturedWorkspaceID = middleware.GetWorkspaceID(c)
		capturedRole = middleware.GetWorkspaceRole(c)
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, workspaceID, capturedWorkspaceID)
	assert.Equal(t, middleware.WorkspaceRoleMember, capturedRole)
}

func TestWorkspaceAccess_RequiredRoles(t *testing.T) {
	tests := []struct {
		name          string
		userRole      string
		requiredRoles []string
		expectedCode  int
	}{
		{
			name:          "admin required, user is admin",
			userRole:      middleware.WorkspaceRoleAdmin,
			requiredRoles: []string{middleware.WorkspaceRoleAdmin, middleware.WorkspaceRoleOwner},
			expectedCode:  http.StatusOK,
		},
		{
			name:          "admin required, user is owner",
			userRole:      middleware.WorkspaceRoleOwner,
			requiredRoles: []string{middleware.WorkspaceRoleAdmin, middleware.WorkspaceRoleOwner},
			expectedCode:  http.StatusOK,
		},
		{
			name:          "admin required, user is member",
			userRole:      middleware.WorkspaceRoleMember,
			requiredRoles: []string{middleware.WorkspaceRoleAdmin, middleware.WorkspaceRoleOwner},
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "owner required, user is admin",
			userRole:      middleware.WorkspaceRoleAdmin,
			requiredRoles: []string{middleware.WorkspaceRoleOwner},
			expectedCode:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			accessChecker := middleware.NewMockWorkspaceAccessChecker()
			userID := uuid.NewUUID()
			workspaceID := uuid.NewUUID()

			accessChecker.AddMembership(&middleware.WorkspaceMembership{
				WorkspaceID:   workspaceID,
				WorkspaceName: "Test Workspace",
				UserID:        userID,
				Role:          tt.userRole,
			})

			config := middleware.WorkspaceConfig{
				AccessChecker:    accessChecker,
				WorkspaceIDParam: "workspace_id",
				RequiredRoles:    tt.requiredRoles,
			}

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set(string(middleware.ContextKeyUserID), userID)
					return next(c)
				}
			})

			e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}, middleware.WorkspaceAccess(config))

			req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestWorkspaceAccess_SystemAdminBypass(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	// Add workspace but NOT membership for the admin
	accessChecker.AddWorkspace(workspaceID)

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
		AllowSystemAdmin: true,
	}

	var capturedWorkspaceID uuid.UUID
	var capturedRole string

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			c.Set(string(middleware.ContextKeyIsSystemAdmin), true)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		capturedWorkspaceID = middleware.GetWorkspaceID(c)
		capturedRole = middleware.GetWorkspaceRole(c)
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, workspaceID, capturedWorkspaceID)
	assert.Equal(t, middleware.WorkspaceRoleAdmin, capturedRole)
}

func TestWorkspaceAccess_SystemAdminBypassDisabled(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	// Add workspace but NOT membership
	accessChecker.AddWorkspace(workspaceID)

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
		AllowSystemAdmin: false, // Disable system admin bypass
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			c.Set(string(middleware.ContextKeyIsSystemAdmin), true)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should fail because admin bypass is disabled and user is not a member
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWorkspaceAccess_SystemAdminNonExistentWorkspace(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	// Do NOT add workspace

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
		AllowSystemAdmin: true,
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			c.Set(string(middleware.ContextKeyIsSystemAdmin), true)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should fail because workspace doesn't exist
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkspaceAccess_NoUserIDInContext(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	workspaceID := uuid.NewUUID()

	accessChecker.AddWorkspace(workspaceID)

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
		AllowSystemAdmin: false,
	}

	// Note: no auth middleware setting user ID

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWorkspaceAccess_NoAccessChecker(t *testing.T) {
	e := echo.New()

	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	config := middleware.WorkspaceConfig{
		AccessChecker:    nil, // No access checker
		WorkspaceIDParam: "workspace_id",
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetWorkspaceID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	workspaceID := middleware.GetWorkspaceID(c)
	assert.True(t, workspaceID.IsZero())

	// Test with workspace ID set
	expectedID := uuid.NewUUID()
	c.Set(string(middleware.ContextKeyWorkspaceID), expectedID)
	workspaceID = middleware.GetWorkspaceID(c)
	assert.Equal(t, expectedID, workspaceID)
}

func TestGetWorkspaceName(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	name := middleware.GetWorkspaceName(c)
	assert.Empty(t, name)

	// Test with name set
	c.Set(string(middleware.ContextKeyWorkspaceName), "My Workspace")
	name = middleware.GetWorkspaceName(c)
	assert.Equal(t, "My Workspace", name)
}

func TestGetWorkspaceRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	role := middleware.GetWorkspaceRole(c)
	assert.Empty(t, role)

	// Test with role set
	c.Set(string(middleware.ContextKeyWorkspaceRole), middleware.WorkspaceRoleAdmin)
	role = middleware.GetWorkspaceRole(c)
	assert.Equal(t, middleware.WorkspaceRoleAdmin, role)
}

func TestIsWorkspaceOwner(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test empty context
	assert.False(t, middleware.IsWorkspaceOwner(c))

	// Test with member role
	c.Set(string(middleware.ContextKeyWorkspaceRole), middleware.WorkspaceRoleMember)
	assert.False(t, middleware.IsWorkspaceOwner(c))

	// Test with admin role
	c.Set(string(middleware.ContextKeyWorkspaceRole), middleware.WorkspaceRoleAdmin)
	assert.False(t, middleware.IsWorkspaceOwner(c))

	// Test with owner role
	c.Set(string(middleware.ContextKeyWorkspaceRole), middleware.WorkspaceRoleOwner)
	assert.True(t, middleware.IsWorkspaceOwner(c))
}

func TestIsWorkspaceAdmin(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{role: "", expected: false},
		{role: middleware.WorkspaceRoleMember, expected: false},
		{role: middleware.WorkspaceRoleAdmin, expected: true},
		{role: middleware.WorkspaceRoleOwner, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.role != "" {
				c.Set(string(middleware.ContextKeyWorkspaceRole), tt.role)
			}

			assert.Equal(t, tt.expected, middleware.IsWorkspaceAdmin(c))
		})
	}
}

func TestRequireWorkspaceRole(t *testing.T) {
	tests := []struct {
		name         string
		userRole     string
		allowedRoles []string
		expectedCode int
	}{
		{
			name:         "allowed role",
			userRole:     middleware.WorkspaceRoleAdmin,
			allowedRoles: []string{middleware.WorkspaceRoleAdmin, middleware.WorkspaceRoleOwner},
			expectedCode: http.StatusOK,
		},
		{
			name:         "not allowed role",
			userRole:     middleware.WorkspaceRoleMember,
			allowedRoles: []string{middleware.WorkspaceRoleAdmin, middleware.WorkspaceRoleOwner},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set(string(middleware.ContextKeyWorkspaceRole), tt.userRole)
					return next(c)
				}
			})

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}, middleware.RequireWorkspaceRole(tt.allowedRoles...))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestRequireWorkspaceAdmin(t *testing.T) {
	tests := []struct {
		name         string
		role         string
		expectedCode int
	}{
		{
			name:         "owner allowed",
			role:         middleware.WorkspaceRoleOwner,
			expectedCode: http.StatusOK,
		},
		{
			name:         "admin allowed",
			role:         middleware.WorkspaceRoleAdmin,
			expectedCode: http.StatusOK,
		},
		{
			name:         "member not allowed",
			role:         middleware.WorkspaceRoleMember,
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set(string(middleware.ContextKeyWorkspaceRole), tt.role)
					return next(c)
				}
			})

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}, middleware.RequireWorkspaceAdmin())

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestRequireWorkspaceOwner(t *testing.T) {
	tests := []struct {
		name         string
		role         string
		expectedCode int
	}{
		{
			name:         "owner allowed",
			role:         middleware.WorkspaceRoleOwner,
			expectedCode: http.StatusOK,
		},
		{
			name:         "admin not allowed",
			role:         middleware.WorkspaceRoleAdmin,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "member not allowed",
			role:         middleware.WorkspaceRoleMember,
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set(string(middleware.ContextKeyWorkspaceRole), tt.role)
					return next(c)
				}
			})

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}, middleware.RequireWorkspaceOwner())

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestMockWorkspaceAccessChecker(t *testing.T) {
	checker := middleware.NewMockWorkspaceAccessChecker()
	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	t.Run("workspace not found", func(t *testing.T) {
		_, err := checker.GetMembership(ctx, workspaceID, userID)
		require.Error(t, err)
		require.ErrorIs(t, err, middleware.ErrWorkspaceNotFound)
	})

	t.Run("workspace exists but no membership", func(t *testing.T) {
		checker.AddWorkspace(workspaceID)

		membership, err := checker.GetMembership(ctx, workspaceID, userID)
		require.NoError(t, err)
		assert.Nil(t, membership)
	})

	t.Run("workspace exists with membership", func(t *testing.T) {
		checker.AddMembership(&middleware.WorkspaceMembership{
			WorkspaceID:   workspaceID,
			WorkspaceName: "Test",
			UserID:        userID,
			Role:          middleware.WorkspaceRoleMember,
		})

		membership, err := checker.GetMembership(ctx, workspaceID, userID)
		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, workspaceID, membership.WorkspaceID)
		assert.Equal(t, userID, membership.UserID)
		assert.Equal(t, middleware.WorkspaceRoleMember, membership.Role)
	})

	t.Run("workspace exists check", func(t *testing.T) {
		exists, err := checker.WorkspaceExists(ctx, workspaceID)
		require.NoError(t, err)
		assert.True(t, exists)

		nonExistentID := uuid.NewUUID()
		exists, err = checker.WorkspaceExists(ctx, nonExistentID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestWorkspaceAccess_ContextEnrichment(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()
	workspaceName := "Engineering Team"

	accessChecker.AddMembership(&middleware.WorkspaceMembership{
		WorkspaceID:   workspaceID,
		WorkspaceName: workspaceName,
		UserID:        userID,
		Role:          middleware.WorkspaceRoleAdmin,
	})

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "workspace_id",
	}

	var extractedWorkspaceID uuid.UUID
	var extractedWorkspaceName string
	var extractedRole string

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/workspaces/:workspace_id", func(c echo.Context) error {
		extractedWorkspaceID = middleware.GetWorkspaceID(c)
		extractedWorkspaceName = middleware.GetWorkspaceName(c)
		extractedRole = middleware.GetWorkspaceRole(c)
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, workspaceID, extractedWorkspaceID)
	assert.Equal(t, workspaceName, extractedWorkspaceName)
	assert.Equal(t, middleware.WorkspaceRoleAdmin, extractedRole)
}

func TestWorkspaceAccess_CustomWorkspaceIDParam(t *testing.T) {
	e := echo.New()

	accessChecker := middleware.NewMockWorkspaceAccessChecker()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	accessChecker.AddMembership(&middleware.WorkspaceMembership{
		WorkspaceID:   workspaceID,
		WorkspaceName: "Test Workspace",
		UserID:        userID,
		Role:          middleware.WorkspaceRoleMember,
	})

	config := middleware.WorkspaceConfig{
		AccessChecker:    accessChecker,
		WorkspaceIDParam: "ws_id", // Custom param name
	}

	var capturedWorkspaceID uuid.UUID

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyUserID), userID)
			return next(c)
		}
	})

	e.GET("/ws/:ws_id/chats", func(c echo.Context) error {
		capturedWorkspaceID = middleware.GetWorkspaceID(c)
		return c.String(http.StatusOK, "ok")
	}, middleware.WorkspaceAccess(config))

	req := httptest.NewRequest(http.MethodGet, "/ws/"+workspaceID.String()+"/chats", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, workspaceID, capturedWorkspaceID)
}
