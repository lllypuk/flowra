package httphandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants.
const (
	maxWorkspaceNameLength        = 100
	maxWorkspaceDescriptionLength = 500
)

// Workspace handler errors.
var (
	ErrWorkspaceNotFound     = errors.New("workspace not found")
	ErrMemberAlreadyExists   = errors.New("member already exists in workspace")
	ErrMemberNotFound        = errors.New("member not found in workspace")
	ErrCannotRemoveOwner     = errors.New("cannot remove workspace owner")
	ErrInvalidRole           = errors.New("invalid role")
	ErrInsufficientPrivilege = errors.New("insufficient privileges for this operation")
)

// CreateWorkspaceRequest represents the request to create a workspace.
type CreateWorkspaceRequest struct {
	Name        string `json:"name"        form:"name"`
	Description string `json:"description" form:"description"`
}

// UpdateWorkspaceRequest represents the request to update a workspace.
type UpdateWorkspaceRequest struct {
	Name        string `json:"name"        form:"name"`
	Description string `json:"description" form:"description"`
}

// AddMemberRequest represents the request to add a member to a workspace.
type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

// UpdateMemberRoleRequest represents the request to update a member's role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

// WorkspaceResponse represents a workspace in API responses.
type WorkspaceResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	MemberCount int       `json:"member_count"`
}

// WorkspaceListResponse represents a list of workspaces in API responses.
type WorkspaceListResponse struct {
	Workspaces []WorkspaceResponse `json:"workspaces"`
	Total      int                 `json:"total"`
	Offset     int                 `json:"offset"`
	Limit      int                 `json:"limit"`
}

// MemberResponse represents a workspace member in API responses.
type MemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt string    `json:"joined_at"`
	Username string    `json:"username,omitempty"`
	Email    string    `json:"email,omitempty"`
}

// WorkspaceService defines the interface for workspace operations.
// Declared on the consumer side per project guidelines.
type WorkspaceService interface {
	// CreateWorkspace creates a new workspace.
	CreateWorkspace(ctx context.Context, ownerID uuid.UUID, name, description string) (*workspace.Workspace, error)

	// GetWorkspace gets a workspace by ID.
	GetWorkspace(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// ListUserWorkspaces lists workspaces for a user.
	ListUserWorkspaces(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, int, error)

	// UpdateWorkspace updates a workspace.
	UpdateWorkspace(ctx context.Context, id uuid.UUID, name, description string) (*workspace.Workspace, error)

	// DeleteWorkspace deletes a workspace (soft delete).
	DeleteWorkspace(ctx context.Context, id uuid.UUID) error

	// GetMemberCount returns the number of members in a workspace.
	GetMemberCount(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// MemberService defines the interface for workspace member operations.
// Declared on the consumer side per project guidelines.
type MemberService interface {
	// AddMember adds a member to a workspace.
	AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role workspace.Role) (*workspace.Member, error)

	// RemoveMember removes a member from a workspace.
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMemberRole updates a member's role in a workspace.
	UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role workspace.Role) (*workspace.Member, error)

	// GetMember gets a member from a workspace.
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// ListMembers lists all members of a workspace.
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, int, error)

	// IsOwner checks if a user is the owner of a workspace.
	IsOwner(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
}

// WorkspaceHandler handles workspace-related HTTP requests.
type WorkspaceHandler struct {
	workspaceService WorkspaceService
	memberService    MemberService
}

// NewWorkspaceHandler creates a new WorkspaceHandler.
func NewWorkspaceHandler(workspaceService WorkspaceService, memberService MemberService) *WorkspaceHandler {
	return &WorkspaceHandler{
		workspaceService: workspaceService,
		memberService:    memberService,
	}
}

// RegisterRoutes registers workspace routes with the router.
func (h *WorkspaceHandler) RegisterRoutes(r *httpserver.Router) {
	// Workspace CRUD (authenticated routes)
	r.Auth().POST("/workspaces", h.Create)
	r.Auth().GET("/workspaces", h.List)
	r.Auth().GET("/workspaces/:id", h.Get)
	r.Auth().PUT("/workspaces/:id", h.Update)
	r.Auth().DELETE("/workspaces/:id", h.Delete)

	// Member management (workspace-scoped routes)
	r.Auth().POST("/workspaces/:id/members", h.AddMember)
	r.Auth().DELETE("/workspaces/:id/members/:user_id", h.RemoveMember)
	r.Auth().PUT("/workspaces/:id/members/:user_id/role", h.UpdateMemberRole)
}

// Create handles POST /api/v1/workspaces.
// Creates a new workspace with the authenticated user as owner.
func (h *WorkspaceHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	var req CreateWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	// Validate required fields
	if req.Name == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace name is required",
		)
	}

	if len(req.Name) > maxWorkspaceNameLength {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace name must be at most 100 characters",
		)
	}

	if len(req.Description) > maxWorkspaceDescriptionLength {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace description must be at most 500 characters",
		)
	}

	ws, err := h.workspaceService.CreateWorkspace(c.Request().Context(), userID, req.Name, req.Description)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"CREATE_FAILED",
			"Failed to create workspace",
		)
	}

	return httpserver.RespondCreated(c, ToWorkspaceResponse(ws, 1)) // Owner is the first member
}

// List handles GET /api/v1/workspaces.
// Lists all workspaces the authenticated user is a member of.
func (h *WorkspaceHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	// Parse pagination parameters
	offset, limit := ParsePagination(c)

	workspaces, total, err := h.workspaceService.ListUserWorkspaces(c.Request().Context(), userID, offset, limit)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"LIST_FAILED",
			"Failed to list workspaces",
		)
	}

	responses := make([]WorkspaceResponse, 0, len(workspaces))
	for _, ws := range workspaces {
		memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), ws.ID())
		responses = append(responses, ToWorkspaceResponse(ws, memberCount))
	}

	return httpserver.RespondOK(c, WorkspaceListResponse{
		Workspaces: responses,
		Total:      total,
		Offset:     offset,
		Limit:      limit,
	})
}

// Get handles GET /api/v1/workspaces/:id.
// Gets a workspace by ID.
func (h *WorkspaceHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	ws, err := h.workspaceService.GetWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace not found",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"GET_FAILED",
			"Failed to get workspace",
		)
	}

	// Check if user is a member of the workspace
	member, _ := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if member == nil && !middleware.IsSystemAdmin(c) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"You are not a member of this workspace",
		)
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), ws.ID())
	return httpserver.RespondOK(c, ToWorkspaceResponse(ws, memberCount))
}

// Update handles PUT /api/v1/workspaces/:id.
// Updates a workspace.
func (h *WorkspaceHandler) Update(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, parseErr := uuid.ParseUUID(c.Param("id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	// Check if user has admin privileges
	if !h.hasAdminPrivileges(c, workspaceID, userID) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Insufficient privileges to update workspace",
		)
	}

	var req UpdateWorkspaceRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	// Validate fields
	if req.Name == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace name is required",
		)
	}

	if len(req.Name) > maxWorkspaceNameLength {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace name must be at most 100 characters",
		)
	}

	if len(req.Description) > maxWorkspaceDescriptionLength {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Workspace description must be at most 500 characters",
		)
	}

	ws, updateErr := h.workspaceService.UpdateWorkspace(c.Request().Context(), workspaceID, req.Name, req.Description)
	if updateErr != nil {
		if errors.Is(updateErr, ErrWorkspaceNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace not found",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"UPDATE_FAILED",
			"Failed to update workspace",
		)
	}

	memberCount, _ := h.workspaceService.GetMemberCount(c.Request().Context(), ws.ID())
	return httpserver.RespondOK(c, ToWorkspaceResponse(ws, memberCount))
}

// Delete handles DELETE /api/v1/workspaces/:id.
// Deletes a workspace (soft delete).
func (h *WorkspaceHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, parseErr := uuid.ParseUUID(c.Param("id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	// Only owner can delete workspace
	isOwner, _ := h.memberService.IsOwner(c.Request().Context(), workspaceID, userID)
	if !isOwner && !middleware.IsSystemAdmin(c) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Only the workspace owner can delete the workspace",
		)
	}

	deleteErr := h.workspaceService.DeleteWorkspace(c.Request().Context(), workspaceID)
	if deleteErr != nil {
		if errors.Is(deleteErr, ErrWorkspaceNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace not found",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"DELETE_FAILED",
			"Failed to delete workspace",
		)
	}

	return httpserver.RespondNoContent(c)
}

// AddMember handles POST /api/v1/workspaces/:id/members.
// Adds a member to a workspace.
func (h *WorkspaceHandler) AddMember(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, parseErr := uuid.ParseUUID(c.Param("id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	// Check if user has admin privileges
	if !h.hasAdminPrivileges(c, workspaceID, userID) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Insufficient privileges to add members",
		)
	}

	var req AddMemberRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	// Validate required fields
	if req.UserID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"User ID is required",
		)
	}

	role, err := ParseRole(req.Role)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Role must be one of: admin, member",
		)
	}

	// Cannot add owner role through this endpoint
	if role == workspace.RoleOwner {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Cannot assign owner role through this endpoint",
		)
	}

	member, err := h.memberService.AddMember(c.Request().Context(), workspaceID, req.UserID, role)
	if err != nil {
		if errors.Is(err, ErrMemberAlreadyExists) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusConflict,
				"MEMBER_ALREADY_EXISTS",
				"User is already a member of this workspace",
			)
		}
		if errors.Is(err, ErrWorkspaceNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace not found",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"ADD_MEMBER_FAILED",
			"Failed to add member",
		)
	}

	return httpserver.RespondCreated(c, ToMemberResponse(member))
}

// RemoveMember handles DELETE /api/v1/workspaces/:id/members/:user_id.
// Removes a member from a workspace.
func (h *WorkspaceHandler) RemoveMember(c echo.Context) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, parseErr := uuid.ParseUUID(c.Param("id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	targetUserID, parseUserErr := uuid.ParseUUID(c.Param("user_id"))
	if parseUserErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_USER_ID",
			"Invalid user ID format",
		)
	}

	// Users can remove themselves, admins can remove others
	if currentUserID != targetUserID && !h.hasAdminPrivileges(c, workspaceID, currentUserID) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Insufficient privileges to remove members",
		)
	}

	// Cannot remove the owner
	isOwner, _ := h.memberService.IsOwner(c.Request().Context(), workspaceID, targetUserID)
	if isOwner {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"CANNOT_REMOVE_OWNER",
			"Cannot remove the workspace owner",
		)
	}

	removeErr := h.memberService.RemoveMember(c.Request().Context(), workspaceID, targetUserID)
	if removeErr != nil {
		if errors.Is(removeErr, ErrMemberNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"MEMBER_NOT_FOUND",
				"Member not found in workspace",
			)
		}
		if errors.Is(removeErr, ErrWorkspaceNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace not found",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"REMOVE_MEMBER_FAILED",
			"Failed to remove member",
		)
	}

	return httpserver.RespondNoContent(c)
}

// UpdateMemberRole handles PUT /api/v1/workspaces/:id/members/:user_id/role.
// Updates a member's role in a workspace.
func (h *WorkspaceHandler) UpdateMemberRole(c echo.Context) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
		)
	}

	workspaceID, parseErr := uuid.ParseUUID(c.Param("id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_WORKSPACE_ID",
			"Invalid workspace ID format",
		)
	}

	targetUserID, parseUserErr := uuid.ParseUUID(c.Param("user_id"))
	if parseUserErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_USER_ID",
			"Invalid user ID format",
		)
	}

	// Only owner can change roles
	isOwner, _ := h.memberService.IsOwner(c.Request().Context(), workspaceID, currentUserID)
	if !isOwner && !middleware.IsSystemAdmin(c) {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Only the workspace owner can change member roles",
		)
	}

	// Cannot change owner's role
	isTargetOwner, _ := h.memberService.IsOwner(c.Request().Context(), workspaceID, targetUserID)
	if isTargetOwner {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"CANNOT_CHANGE_OWNER_ROLE",
			"Cannot change the role of the workspace owner",
		)
	}

	var req UpdateMemberRoleRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request body",
		)
	}

	role, err := ParseRole(req.Role)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Role must be one of: admin, member",
		)
	}

	// Cannot assign owner role
	if role == workspace.RoleOwner {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Cannot assign owner role",
		)
	}

	member, err := h.memberService.UpdateMemberRole(c.Request().Context(), workspaceID, targetUserID, role)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusNotFound,
				"MEMBER_NOT_FOUND",
				"Member not found in workspace",
			)
		}
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusInternalServerError,
			"UPDATE_ROLE_FAILED",
			"Failed to update member role",
		)
	}

	return httpserver.RespondOK(c, ToMemberResponse(member))
}

// hasAdminPrivileges checks if a user has admin privileges in a workspace.
func (h *WorkspaceHandler) hasAdminPrivileges(c echo.Context, workspaceID, userID uuid.UUID) bool {
	if middleware.IsSystemAdmin(c) {
		return true
	}

	member, err := h.memberService.GetMember(c.Request().Context(), workspaceID, userID)
	if err != nil {
		return false
	}

	return member.IsAdmin()
}

// ToWorkspaceResponse converts a domain Workspace to a WorkspaceResponse.
func ToWorkspaceResponse(ws *workspace.Workspace, memberCount int) WorkspaceResponse {
	return WorkspaceResponse{
		ID:          ws.ID(),
		Name:        ws.Name(),
		Description: "", // Description not in domain model yet
		OwnerID:     ws.CreatedBy(),
		CreatedAt:   ws.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   ws.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		MemberCount: memberCount,
	}
}

// ToMemberResponse converts a domain Member to a MemberResponse.
func ToMemberResponse(m *workspace.Member) MemberResponse {
	return MemberResponse{
		UserID:   m.UserID(),
		Role:     m.Role().String(),
		JoinedAt: m.JoinedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ParseRole parses a role string into a workspace.Role.
func ParseRole(roleStr string) (workspace.Role, error) {
	switch roleStr {
	case "owner":
		return workspace.RoleOwner, nil
	case "admin":
		return workspace.RoleAdmin, nil
	case "member":
		return workspace.RoleMember, nil
	default:
		return "", ErrInvalidRole
	}
}

// ParsePagination extracts pagination parameters from the request.
func ParsePagination(c echo.Context) (int, int) {
	const defaultLimit = 20
	const maxLimit = 100

	offset := 0
	limit := defaultLimit

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= maxLimit {
			limit = parsed
		}
	}

	return offset, limit
}

// MockWorkspaceService is a mock implementation of WorkspaceService for testing.
type MockWorkspaceService struct {
	workspaces   map[uuid.UUID]*workspace.Workspace
	memberCounts map[uuid.UUID]int
}

// NewMockWorkspaceService creates a new mock workspace service.
func NewMockWorkspaceService() *MockWorkspaceService {
	return &MockWorkspaceService{
		workspaces:   make(map[uuid.UUID]*workspace.Workspace),
		memberCounts: make(map[uuid.UUID]int),
	}
}

// AddWorkspace adds a workspace to the mock service.
func (m *MockWorkspaceService) AddWorkspace(ws *workspace.Workspace, memberCount int) {
	m.workspaces[ws.ID()] = ws
	m.memberCounts[ws.ID()] = memberCount
}

// CreateWorkspace implements WorkspaceService.
func (m *MockWorkspaceService) CreateWorkspace(
	_ context.Context,
	ownerID uuid.UUID,
	name, _ string,
) (*workspace.Workspace, error) {
	ws, err := workspace.NewWorkspace(name, "keycloak-group-"+uuid.NewUUID().String(), ownerID)
	if err != nil {
		return nil, err
	}
	m.workspaces[ws.ID()] = ws
	m.memberCounts[ws.ID()] = 1
	return ws, nil
}

// GetWorkspace implements WorkspaceService.
func (m *MockWorkspaceService) GetWorkspace(_ context.Context, id uuid.UUID) (*workspace.Workspace, error) {
	ws, ok := m.workspaces[id]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	return ws, nil
}

// ListUserWorkspaces implements WorkspaceService.
func (m *MockWorkspaceService) ListUserWorkspaces(
	_ context.Context,
	_ uuid.UUID,
	offset, limit int,
) ([]*workspace.Workspace, int, error) {
	all := make([]*workspace.Workspace, 0, len(m.workspaces))
	for _, ws := range m.workspaces {
		all = append(all, ws)
	}

	total := len(all)
	if offset >= total {
		return []*workspace.Workspace{}, total, nil
	}

	end := min(offset+limit, total)

	return all[offset:end], total, nil
}

// UpdateWorkspace implements WorkspaceService.
func (m *MockWorkspaceService) UpdateWorkspace(
	_ context.Context,
	id uuid.UUID,
	name, _ string,
) (*workspace.Workspace, error) {
	ws, ok := m.workspaces[id]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	if err := ws.UpdateName(name); err != nil {
		return nil, err
	}
	return ws, nil
}

// DeleteWorkspace implements WorkspaceService.
func (m *MockWorkspaceService) DeleteWorkspace(_ context.Context, id uuid.UUID) error {
	if _, ok := m.workspaces[id]; !ok {
		return ErrWorkspaceNotFound
	}
	delete(m.workspaces, id)
	delete(m.memberCounts, id)
	return nil
}

// GetMemberCount implements WorkspaceService.
func (m *MockWorkspaceService) GetMemberCount(_ context.Context, workspaceID uuid.UUID) (int, error) {
	count, ok := m.memberCounts[workspaceID]
	if !ok {
		return 0, ErrWorkspaceNotFound
	}
	return count, nil
}

// MockMemberService is a mock implementation of MemberService for testing.
type MockMemberService struct {
	members map[string]*workspace.Member // key: "workspaceID:userID"
	owners  map[uuid.UUID]uuid.UUID      // workspaceID -> ownerID
}

// NewMockMemberService creates a new mock member service.
func NewMockMemberService() *MockMemberService {
	return &MockMemberService{
		members: make(map[string]*workspace.Member),
		owners:  make(map[uuid.UUID]uuid.UUID),
	}
}

// AddMemberToMock adds a member to the mock service.
func (m *MockMemberService) AddMemberToMock(member *workspace.Member) {
	key := member.WorkspaceID().String() + ":" + member.UserID().String()
	m.members[key] = member
}

// SetOwner sets the owner of a workspace in the mock.
func (m *MockMemberService) SetOwner(workspaceID, ownerID uuid.UUID) {
	m.owners[workspaceID] = ownerID
}

// AddMember implements MemberService.
func (m *MockMemberService) AddMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	key := workspaceID.String() + ":" + userID.String()
	if _, exists := m.members[key]; exists {
		return nil, ErrMemberAlreadyExists
	}

	member := workspace.NewMember(userID, workspaceID, role)
	m.members[key] = &member
	return &member, nil
}

// RemoveMember implements MemberService.
func (m *MockMemberService) RemoveMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	key := workspaceID.String() + ":" + userID.String()
	if _, exists := m.members[key]; !exists {
		return ErrMemberNotFound
	}
	delete(m.members, key)
	return nil
}

// UpdateMemberRole implements MemberService.
func (m *MockMemberService) UpdateMemberRole(
	_ context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	key := workspaceID.String() + ":" + userID.String()
	if _, exists := m.members[key]; !exists {
		return nil, ErrMemberNotFound
	}

	member := workspace.NewMember(userID, workspaceID, role)
	m.members[key] = &member
	return &member, nil
}

// GetMember implements MemberService.
func (m *MockMemberService) GetMember(_ context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error) {
	key := workspaceID.String() + ":" + userID.String()
	member, exists := m.members[key]
	if !exists {
		return nil, ErrMemberNotFound
	}
	return member, nil
}

// ListMembers implements MemberService.
func (m *MockMemberService) ListMembers(
	_ context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*workspace.Member, int, error) {
	var members []*workspace.Member
	prefix := workspaceID.String() + ":"
	for key, member := range m.members {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			members = append(members, member)
		}
	}

	total := len(members)
	if offset >= total {
		return []*workspace.Member{}, total, nil
	}

	end := min(offset+limit, total)

	return members[offset:end], total, nil
}

// IsOwner implements MemberService.
func (m *MockMemberService) IsOwner(_ context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	ownerID, exists := m.owners[workspaceID]
	if !exists {
		return false, nil
	}
	return ownerID == userID, nil
}
