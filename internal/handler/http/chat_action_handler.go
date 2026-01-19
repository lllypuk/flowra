package httphandler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// ActionService defines the interface for chat actions
// This is a consumer-side interface to avoid import cycles
type ActionService interface {
	ChangeStatus(
		ctx context.Context,
		chatID uuid.UUID,
		newStatus string,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)
	SetPriority(
		ctx context.Context,
		chatID uuid.UUID,
		priority string,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)
	AssignUser(
		ctx context.Context,
		chatID uuid.UUID,
		assigneeID *uuid.UUID,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)
	SetDueDate(
		ctx context.Context,
		chatID uuid.UUID,
		dueDate *time.Time,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)
	Close(ctx context.Context, chatID uuid.UUID, actorID uuid.UUID) (*appcore.ActionResult, error)
	Reopen(ctx context.Context, chatID uuid.UUID, actorID uuid.UUID) (*appcore.ActionResult, error)
	Rename(ctx context.Context, chatID uuid.UUID, newTitle string, actorID uuid.UUID) (*appcore.ActionResult, error)
}

// ChatActionHandler handles chat action endpoints (status, priority, etc.)
// These endpoints create system messages with tags instead of direct modifications
type ChatActionHandler struct {
	actionService ActionService
}

// NewChatActionHandler creates a new ChatActionHandler
func NewChatActionHandler(actionService ActionService) *ChatActionHandler {
	return &ChatActionHandler{
		actionService: actionService,
	}
}

// ChangeStatus handles POST /api/v1/chats/:id/actions/status
//
//nolint:dupl // Similar HTTP handler pattern with different validation logic
func (h *ChatActionHandler) ChangeStatus(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req struct {
		Status string `json:"status"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	if req.Status == "" {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_STATUS", "status is required")
	}

	_, err := h.actionService.ChangeStatus(ctx, chatID, req.Status, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// For HTMX, return empty response with HX-Trigger header
	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// ChangePriority handles POST /api/v1/chats/:id/actions/priority
//
//nolint:dupl // Similar HTTP handler pattern with different validation logic
func (h *ChatActionHandler) ChangePriority(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req struct {
		Priority string `json:"priority"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	if req.Priority == "" {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_PRIORITY", "priority is required")
	}

	_, err := h.actionService.SetPriority(ctx, chatID, req.Priority, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// ChangeAssignee handles POST /api/v1/chats/:id/actions/assignee
func (h *ChatActionHandler) ChangeAssignee(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req struct {
		AssigneeID string `json:"assignee_id"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var assigneeID *uuid.UUID
	if req.AssigneeID != "" {
		parsed, err := uuid.ParseUUID(req.AssigneeID)
		if err != nil {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusBadRequest,
				"INVALID_ASSIGNEE_ID",
				"invalid assignee ID format",
			)
		}
		assigneeID = &parsed
	}

	_, err := h.actionService.AssignUser(ctx, chatID, assigneeID, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// SetDueDate handles POST /api/v1/chats/:id/actions/due-date
func (h *ChatActionHandler) SetDueDate(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req struct {
		DueDate string `json:"due_date"` // ISO 8601 date string or empty to clear
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return httpserver.RespondErrorWithCode(
				c,
				http.StatusBadRequest,
				"INVALID_DATE",
				"invalid date format, use YYYY-MM-DD",
			)
		}
		dueDate = &parsed
	}

	_, err := h.actionService.SetDueDate(ctx, chatID, dueDate, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// Close handles POST /api/v1/chats/:id/actions/close
func (h *ChatActionHandler) Close(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	_, err := h.actionService.Close(ctx, chatID, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// Reopen handles POST /api/v1/chats/:id/actions/reopen
func (h *ChatActionHandler) Reopen(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	_, err := h.actionService.Reopen(ctx, chatID, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}

// Rename handles POST /api/v1/chats/:id/actions/rename
//
//nolint:dupl // Similar HTTP handler pattern with different validation logic
func (h *ChatActionHandler) Rename(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req struct {
		Title string `json:"title"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	if req.Title == "" {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_TITLE", "title is required")
	}

	_, err := h.actionService.Rename(ctx, chatID, req.Title, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	c.Response().Header().Set("Hx-Trigger", "chatUpdated")
	return c.NoContent(http.StatusOK)
}
