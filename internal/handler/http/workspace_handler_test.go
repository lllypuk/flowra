package httphandler_test

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httphandler "github.com/lllypuk/flowra/internal/handler/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test workspace.
func createTestWorkspace(t *testing.T, ownerID uuid.UUID, name string) *workspace.Workspace {
	t.Helper()
	ws, err := workspace.NewWorkspace(name, "keycloak-group-"+uuid.NewUUID().String(), ownerID)
	require.NoError(t, err)
	return ws
}

// Helper function to set up workspace auth context.
func setupWorkspaceAuthContext(c echo.Context, userID uuid.UUID, isSystemAdmin bool) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	c.Set(string(middleware.ContextKeyIsSystemAdmin), isSystemAdmin)
}

// Helper functions for building URLs.
func workspaceMembersURL(workspaceID uuid.UUID) string {
	return "/api/v1/workspaces/" + workspaceID.String() + "/members"
}

func workspaceMemberURL(workspaceID, userID uuid.UUID) string {
	return "/api/v1/workspaces/" + workspaceID.String() + "/members/" + userID.String()
}

func workspaceMemberRoleURL(workspaceID, userID uuid.UUID) string {
	return workspaceMemberURL(workspaceID, userID) + "/role"
}

func TestWorkspaceHandler_Create(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"name": "Test Workspace", "description": "A test workspace"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("missing name", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"description": "A test workspace"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	})

	t.Run("name too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		longName := strings.Repeat("a", 101)
		reqBody := `{"name": "` + longName + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("description too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		longDesc := strings.Repeat("a", 501)
		reqBody := `{"name": "Test", "description": "` + longDesc + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized - no user in context", func(t *testing.T) {
		e := echo.New()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"name": "Test Workspace"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `not valid json`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestWorkspaceHandler_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws1 := createTestWorkspace(t, userID, "Workspace 1")
		ws2 := createTestWorkspace(t, userID, "Workspace 2")
		mockWSService.AddWorkspace(ws1, 3)
		mockWSService.AddWorkspace(ws2, 5)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		for range 5 {
			ws := createTestWorkspace(t, userID, "Workspace")
			mockWSService.AddWorkspace(ws, 1)
		}

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces?offset=2&limit=2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("unauthorized - no user in context", func(t *testing.T) {
		e := echo.New()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestWorkspaceHandler_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, userID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		// Add user as member
		member := workspace.NewMember(userID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("workspace not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		nonExistentID := uuid.NewUUID()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/"+nonExistentID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentID.String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("forbidden - not a member", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("system admin can access any workspace", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, true)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid workspace ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestWorkspaceHandler_Update(t *testing.T) {
	t.Run("successful update by admin", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, userID, "Original Name")
		mockWSService.AddWorkspace(ws, 1)

		// Add user as admin
		member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"name": "Updated Name", "description": "Updated description"}`
		req := httptest.NewRequest(
			stdhttp.MethodPut,
			"/api/v1/workspaces/"+ws.ID().String(),
			strings.NewReader(reqBody),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("forbidden - not admin", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Original Name")
		mockWSService.AddWorkspace(ws, 1)

		// Add user as regular member
		member := workspace.NewMember(userID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"name": "Updated Name"}`
		req := httptest.NewRequest(
			stdhttp.MethodPut,
			"/api/v1/workspaces/"+ws.ID().String(),
			strings.NewReader(reqBody),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("missing name", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, userID, "Original Name")
		mockWSService.AddWorkspace(ws, 1)

		member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"description": "Updated description"}`
		req := httptest.NewRequest(
			stdhttp.MethodPut,
			"/api/v1/workspaces/"+ws.ID().String(),
			strings.NewReader(reqBody),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("workspace not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		nonExistentID := uuid.NewUUID()
		reqBody := `{"name": "Updated Name"}`
		reqURL := "/api/v1/workspaces/" + nonExistentID.String()
		req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentID.String())

		setupWorkspaceAuthContext(c, userID, true) // System admin

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})
}

func TestWorkspaceHandler_Delete(t *testing.T) {
	t.Run("successful delete by owner", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, userID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)
		mockMemberService.SetOwner(ws.ID(), userID)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("forbidden - not owner", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		// Add user as admin (not owner)
		member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, userID, false)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("system admin can delete", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/workspaces/"+ws.ID().String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, true)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})
}

func TestWorkspaceHandler_AddMember(t *testing.T) {
	t.Run("successful add member", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		newUserID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		// Add admin as owner
		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&adminMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"user_id": "` + newUserID.String() + `", "role": "member"}`
		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members"
		req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.AddMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("member already exists", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		existingUserID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&adminMember)

		existingMember := workspace.NewMember(existingUserID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&existingMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"user_id": "` + existingUserID.String() + `", "role": "member"}`
		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members"
		req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.AddMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusConflict, rec.Code)
	})

	t.Run("cannot add owner role", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		newUserID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&adminMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"user_id": "` + newUserID.String() + `", "role": "owner"}`
		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members"
		req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.AddMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid role", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		newUserID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&adminMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"user_id": "` + newUserID.String() + `", "role": "invalid"}`
		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members"
		req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.AddMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("forbidden - not admin", func(t *testing.T) {
		e := echo.New()
		memberID := uuid.NewUUID()
		newUserID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)

		// Add user as regular member
		member := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"user_id": "` + newUserID.String() + `", "role": "member"}`
		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members"
		req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(ws.ID().String())

		setupWorkspaceAuthContext(c, memberID, false)

		err := handler.AddMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})
}

func TestWorkspaceHandler_RemoveMember(t *testing.T) {
	t.Run("admin removes member", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		memberID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&adminMember)

		targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&targetMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members/" + memberID.String()
		req := httptest.NewRequest(stdhttp.MethodDelete, reqURL, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), memberID.String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.RemoveMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("member removes self", func(t *testing.T) {
		e := echo.New()
		memberID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)

		member := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&member)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, workspaceMemberURL(ws.ID(), memberID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), memberID.String())

		setupWorkspaceAuthContext(c, memberID, false)

		err := handler.RemoveMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&adminMember)

		ownerMember := workspace.NewMember(ownerID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&ownerMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, workspaceMemberURL(ws.ID(), ownerID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), ownerID.String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.RemoveMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("member not found", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		nonExistentID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, adminID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 1)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&adminMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		req := httptest.NewRequest(stdhttp.MethodDelete, workspaceMemberURL(ws.ID(), nonExistentID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), nonExistentID.String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.RemoveMember(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})
}

func TestWorkspaceHandler_UpdateMemberRole(t *testing.T) {
	t.Run("owner updates member role", func(t *testing.T) {
		e := echo.New()
		ownerID := uuid.NewUUID()
		memberID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		ownerMember := workspace.NewMember(ownerID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&ownerMember)

		targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&targetMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"role": "admin"}`
		reqURL := workspaceMemberRoleURL(ws.ID(), memberID)
		req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), memberID.String())

		setupWorkspaceAuthContext(c, ownerID, false)

		err := handler.UpdateMemberRole(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("cannot change owner role", func(t *testing.T) {
		e := echo.New()
		ownerID := uuid.NewUUID()
		adminID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		ownerMember := workspace.NewMember(ownerID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&ownerMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"role": "member"}`
		reqURL := workspaceMemberRoleURL(ws.ID(), ownerID)
		req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), ownerID.String())

		setupWorkspaceAuthContext(c, adminID, true) // Even system admin cannot change owner role

		err := handler.UpdateMemberRole(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("cannot assign owner role", func(t *testing.T) {
		e := echo.New()
		ownerID := uuid.NewUUID()
		memberID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 2)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		ownerMember := workspace.NewMember(ownerID, ws.ID(), workspace.RoleOwner)
		mockMemberService.AddMemberToMock(&ownerMember)

		targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&targetMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"role": "owner"}`
		reqURL := workspaceMemberRoleURL(ws.ID(), memberID)
		req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), memberID.String())

		setupWorkspaceAuthContext(c, ownerID, false)

		err := handler.UpdateMemberRole(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("admin cannot change roles", func(t *testing.T) {
		e := echo.New()
		adminID := uuid.NewUUID()
		memberID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		mockWSService := httphandler.NewMockWorkspaceService()
		mockMemberService := httphandler.NewMockMemberService()

		ws := createTestWorkspace(t, ownerID, "Test Workspace")
		mockWSService.AddWorkspace(ws, 3)
		mockMemberService.SetOwner(ws.ID(), ownerID)

		adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleAdmin)
		mockMemberService.AddMemberToMock(&adminMember)

		targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
		mockMemberService.AddMemberToMock(&targetMember)

		handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

		reqBody := `{"role": "admin"}`
		reqURL := workspaceMemberRoleURL(ws.ID(), memberID)
		req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(ws.ID().String(), memberID.String())

		setupWorkspaceAuthContext(c, adminID, false)

		err := handler.UpdateMemberRole(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		input    string
		expected workspace.Role
		hasError bool
	}{
		{"owner", workspace.RoleOwner, false},
		{"admin", workspace.RoleAdmin, false},
		{"member", workspace.RoleMember, false},
		{"invalid", "", true},
		{"", "", true},
		{"ADMIN", "", true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			role, err := httphandler.ParseRole(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, role)
			}
		})
	}
}

func TestParsePagination(t *testing.T) {
	e := echo.New()

	t.Run("default values", func(t *testing.T) {
		req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		offset, limit := httphandler.ParsePagination(c)
		assert.Equal(t, 0, offset)
		assert.Equal(t, 20, limit)
	})

	t.Run("custom values", func(t *testing.T) {
		req := httptest.NewRequest(stdhttp.MethodGet, "/?offset=10&limit=50", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		offset, limit := httphandler.ParsePagination(c)
		assert.Equal(t, 10, offset)
		assert.Equal(t, 50, limit)
	})

	t.Run("max limit enforced", func(t *testing.T) {
		req := httptest.NewRequest(stdhttp.MethodGet, "/?limit=500", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_, limit := httphandler.ParsePagination(c)
		assert.Equal(t, 20, limit) // Falls back to default when over max
	})

	t.Run("negative offset ignored", func(t *testing.T) {
		req := httptest.NewRequest(stdhttp.MethodGet, "/?offset=-5", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		offset, _ := httphandler.ParsePagination(c)
		assert.Equal(t, 0, offset)
	})

	t.Run("invalid values use defaults", func(t *testing.T) {
		req := httptest.NewRequest(stdhttp.MethodGet, "/?offset=abc&limit=xyz", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		offset, limit := httphandler.ParsePagination(c)
		assert.Equal(t, 0, offset)
		assert.Equal(t, 20, limit)
	})
}

func TestNewWorkspaceHandler(t *testing.T) {
	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	assert.NotNil(t, handler)
}

func TestToWorkspaceResponse(t *testing.T) {
	ownerID := uuid.NewUUID()
	ws, err := workspace.NewWorkspace("Test Workspace", "keycloak-group", ownerID)
	require.NoError(t, err)

	resp := httphandler.ToWorkspaceResponse(ws, 5)

	assert.Equal(t, ws.ID(), resp.ID)
	assert.Equal(t, ws.Name(), resp.Name)
	assert.Equal(t, ws.CreatedBy(), resp.OwnerID)
	assert.Equal(t, 5, resp.MemberCount)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

func TestToMemberResponse(t *testing.T) {
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()
	member := workspace.NewMember(userID, workspaceID, workspace.RoleAdmin)

	resp := httphandler.ToMemberResponse(&member)

	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "admin", resp.Role)
	assert.NotEmpty(t, resp.JoinedAt)
}

func TestWorkspaceHandler_RegisterRoutes(t *testing.T) {
	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	e := echo.New()
	router := httpserver.NewRouter(e, httpserver.DefaultRouterConfig())

	// Should not panic
	handler.RegisterRoutes(router)

	// Verify routes are registered
	routes := e.Routes()
	assert.NotEmpty(t, routes)
}

func TestWorkspaceHandler_Create_EmptyName(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"name": ""}`
	req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_AddMember_MissingUserID(t *testing.T) {
	e := echo.New()
	adminID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, adminID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)

	adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
	mockMemberService.AddMemberToMock(&adminMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"role": "member"}`
	req := httptest.NewRequest(stdhttp.MethodPost, workspaceMembersURL(ws.ID()), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ws.ID().String())

	setupWorkspaceAuthContext(c, adminID, false)

	err := handler.AddMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	var resp httpserver.Response
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestWorkspaceHandler_AddMember_InvalidJSON(t *testing.T) {
	e := echo.New()
	adminID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, adminID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)

	adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleOwner)
	mockMemberService.AddMemberToMock(&adminMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `not valid json`
	req := httptest.NewRequest(stdhttp.MethodPost, workspaceMembersURL(ws.ID()), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ws.ID().String())

	setupWorkspaceAuthContext(c, adminID, false)

	err := handler.AddMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_InvalidUserID(t *testing.T) {
	e := echo.New()
	adminID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, adminID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)

	adminMember := workspace.NewMember(adminID, ws.ID(), workspace.RoleAdmin)
	mockMemberService.AddMemberToMock(&adminMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members/invalid-id"
	req := httptest.NewRequest(stdhttp.MethodDelete, reqURL, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues(ws.ID().String(), "invalid-id")

	setupWorkspaceAuthContext(c, adminID, false)

	err := handler.RemoveMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_UpdateMemberRole_InvalidWorkspaceID(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	targetUserID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"role": "admin"}`
	reqURL := "/api/v1/workspaces/invalid-id/members/" + targetUserID.String() + "/role"
	req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("invalid-id", targetUserID.String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.UpdateMemberRole(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_UpdateMemberRole_InvalidUserID(t *testing.T) {
	e := echo.New()
	ownerID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, ownerID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)
	mockMemberService.SetOwner(ws.ID(), ownerID)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"role": "admin"}`
	reqURL := "/api/v1/workspaces/" + ws.ID().String() + "/members/invalid-id/role"
	req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues(ws.ID().String(), "invalid-id")

	setupWorkspaceAuthContext(c, ownerID, false)

	err := handler.UpdateMemberRole(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_UpdateMemberRole_InvalidJSON(t *testing.T) {
	e := echo.New()
	ownerID := uuid.NewUUID()
	memberID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, ownerID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)
	mockMemberService.SetOwner(ws.ID(), ownerID)

	targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
	mockMemberService.AddMemberToMock(&targetMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `not valid json`
	reqURL := workspaceMemberRoleURL(ws.ID(), memberID)
	req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues(ws.ID().String(), memberID.String())

	setupWorkspaceAuthContext(c, ownerID, false)

	err := handler.UpdateMemberRole(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_UpdateMemberRole_InvalidRole(t *testing.T) {
	e := echo.New()
	ownerID := uuid.NewUUID()
	memberID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, ownerID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 1)
	mockMemberService.SetOwner(ws.ID(), ownerID)

	targetMember := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
	mockMemberService.AddMemberToMock(&targetMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"role": "invalid_role"}`
	reqURL := workspaceMemberRoleURL(ws.ID(), memberID)
	req := httptest.NewRequest(stdhttp.MethodPut, reqURL, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues(ws.ID().String(), memberID.String())

	setupWorkspaceAuthContext(c, ownerID, false)

	err := handler.UpdateMemberRole(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Update_InvalidJSON(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, userID, "Original Name")
	mockWSService.AddWorkspace(ws, 1)

	member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
	mockMemberService.AddMemberToMock(&member)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `not valid json`
	req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/workspaces/"+ws.ID().String(), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ws.ID().String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Update(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Update_NameTooLong(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, userID, "Original Name")
	mockWSService.AddWorkspace(ws, 1)

	member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
	mockMemberService.AddMemberToMock(&member)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	longName := strings.Repeat("a", 101)
	reqBody := `{"name": "` + longName + `"}`
	req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/workspaces/"+ws.ID().String(), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ws.ID().String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Update(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Update_DescriptionTooLong(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, userID, "Original Name")
	mockWSService.AddWorkspace(ws, 1)

	member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
	mockMemberService.AddMemberToMock(&member)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	longDesc := strings.Repeat("a", 501)
	reqBody := `{"name": "Updated", "description": "` + longDesc + `"}`
	req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/workspaces/"+ws.ID().String(), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ws.ID().String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Update(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Delete_InvalidWorkspaceID(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/workspaces/invalid-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Delete_WorkspaceNotFound(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	nonExistentID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()
	mockMemberService.SetOwner(nonExistentID, userID) // User is owner but ws doesn't exist

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/workspaces/"+nonExistentID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(nonExistentID.String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
}

func TestWorkspaceHandler_AddMember_InvalidWorkspaceID(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	newUserID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqBody := `{"user_id": "` + newUserID.String() + `", "role": "member"}`
	reqURL := "/api/v1/workspaces/invalid-id/members"
	req := httptest.NewRequest(stdhttp.MethodPost, reqURL, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.AddMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_InvalidWorkspaceID(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	targetUserID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	reqURL := "/api/v1/workspaces/invalid-id/members/" + targetUserID.String()
	req := httptest.NewRequest(stdhttp.MethodDelete, reqURL, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("invalid-id", targetUserID.String())

	setupWorkspaceAuthContext(c, userID, false)

	err := handler.RemoveMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_Forbidden(t *testing.T) {
	e := echo.New()
	memberID := uuid.NewUUID()
	targetUserID := uuid.NewUUID()
	ownerID := uuid.NewUUID()

	mockWSService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()

	ws := createTestWorkspace(t, ownerID, "Test Workspace")
	mockWSService.AddWorkspace(ws, 3)

	// memberID is a regular member, not admin
	member := workspace.NewMember(memberID, ws.ID(), workspace.RoleMember)
	mockMemberService.AddMemberToMock(&member)

	// targetUserID is also a member
	targetMember := workspace.NewMember(targetUserID, ws.ID(), workspace.RoleMember)
	mockMemberService.AddMemberToMock(&targetMember)

	handler := httphandler.NewWorkspaceHandler(mockWSService, mockMemberService)

	req := httptest.NewRequest(stdhttp.MethodDelete, workspaceMemberURL(ws.ID(), targetUserID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "user_id")
	c.SetParamValues(ws.ID().String(), targetUserID.String())

	setupWorkspaceAuthContext(c, memberID, false)

	err := handler.RemoveMember(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
}

func TestWorkspaceErrors(t *testing.T) {
	t.Run("error variables are defined", func(t *testing.T) {
		require.Error(t, httphandler.ErrWorkspaceNotFound)
		require.Error(t, httphandler.ErrMemberAlreadyExists)
		require.Error(t, httphandler.ErrMemberNotFound)
		require.Error(t, httphandler.ErrCannotRemoveOwner)
		require.Error(t, httphandler.ErrInvalidRole)
		require.Error(t, httphandler.ErrInsufficientPrivilege)
	})
}

func TestMockWorkspaceService_EdgeCases(t *testing.T) {
	mockService := httphandler.NewMockWorkspaceService()

	t.Run("list with offset beyond total", func(t *testing.T) {
		userID := uuid.NewUUID()
		ctx := t.Context()
		ws, _ := mockService.CreateWorkspace(ctx, userID, "Test", "")

		result, total, err := mockService.ListUserWorkspaces(ctx, userID, 100, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.GreaterOrEqual(t, total, 1) // At least the one we created

		// Clean up
		_ = mockService.DeleteWorkspace(ctx, ws.ID())
	})
}

func TestMockMemberService_EdgeCases(t *testing.T) {
	mockService := httphandler.NewMockMemberService()
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	t.Run("list with offset beyond total", func(t *testing.T) {
		ctx := t.Context()
		member, err := mockService.AddMember(ctx, workspaceID, userID, workspace.RoleMember)
		require.NoError(t, err)
		require.NotNil(t, member)

		result, total, err := mockService.ListMembers(ctx, workspaceID, 100, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 1, total)
	})
}
