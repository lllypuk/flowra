package httphandler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	notifapp "github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Notification template handler constants.
const (
	defaultNotificationTemplateListLimit = 20
	maxNotificationTemplateListLimit     = 100
	dropdownNotificationLimit            = 10
)

// NotificationTemplateService defines the interface for notification operations needed by templates.
// Declared on the consumer side per project guidelines.
type NotificationTemplateService interface {
	// ListNotifications lists notifications for a user.
	ListNotifications(ctx context.Context, query notifapp.ListNotificationsQuery) (notifapp.ListResult, error)

	// CountUnread counts unread notifications for a user.
	CountUnread(ctx context.Context, query notifapp.CountUnreadQuery) (notifapp.CountResult, error)

	// MarkAsRead marks a notification as read.
	MarkAsRead(ctx context.Context, cmd notifapp.MarkAsReadCommand) (notifapp.Result, error)

	// GetNotification gets a notification by ID.
	GetNotification(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) (*notification.Notification, error)
}

// NotificationViewData represents notification data for templates.
type NotificationViewData struct {
	ID         string
	Type       string
	Title      string
	Message    string
	IsRead     bool
	ResourceID string
	Link       string
	CreatedAt  time.Time
	ReadAt     *time.Time
}

// NotificationListData represents data for the notification list template.
type NotificationListData struct {
	Notifications []NotificationViewData
	TotalCount    int
	UnreadCount   int
	HasMore       bool
	NextOffset    int
	Filter        string
}

// NotificationTemplateHandler provides handlers for rendering notification HTML pages.
type NotificationTemplateHandler struct {
	renderer            *TemplateRenderer
	logger              *slog.Logger
	notificationService NotificationTemplateService
}

// NewNotificationTemplateHandler creates a new notification template handler.
func NewNotificationTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	notificationService NotificationTemplateService,
) *NotificationTemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotificationTemplateHandler{
		renderer:            renderer,
		logger:              logger,
		notificationService: notificationService,
	}
}

// SetupNotificationRoutes registers notification-related page and partial routes.
func (h *NotificationTemplateHandler) SetupNotificationRoutes(e *echo.Echo) {
	// Notification pages (protected)
	e.GET("/notifications", h.NotificationsPage, RequireAuth)
	e.GET("/notifications/:id/redirect", h.NotificationRedirect, RequireAuth)

	// Notification partials (protected)
	partials := e.Group("/partials", RequireAuth)
	partials.GET("/notifications", h.NotificationsDropdownPartial)
	partials.GET("/notifications/count", h.NotificationCountPartial)
	partials.GET("/notifications/list", h.NotificationsListPartial)
}

// NotificationsPage renders the full notifications page.
func (h *NotificationTemplateHandler) NotificationsPage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return c.Redirect(http.StatusFound, "/login")
	}

	filter := c.QueryParam("filter")

	// Get unread count for header
	countQuery := notifapp.CountUnreadQuery{UserID: userID}
	countResult, err := h.notificationService.CountUnread(c.Request().Context(), countQuery)
	if err != nil {
		h.logger.Error("failed to count unread notifications", slog.String("error", err.Error()))
		countResult = notifapp.CountResult{Count: 0}
	}

	data := map[string]any{
		"UnreadCount": countResult.Count,
		"Filter":      filter,
	}

	return h.render(c, "notification/list.html", "Notifications", data)
}

// NotificationsDropdownPartial returns notification dropdown content as HTML partial.
func (h *NotificationTemplateHandler) NotificationsDropdownPartial(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	limit := dropdownNotificationLimit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxNotificationTemplateListLimit {
			limit = l
		}
	}

	// Get recent notifications
	query := notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      limit,
		Offset:     0,
	}

	result, err := h.notificationService.ListNotifications(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to list notifications", slog.String("error", err.Error()))
		return h.renderPartial(c, "notification/dropdown-content", NotificationListData{
			Notifications: []NotificationViewData{},
			UnreadCount:   0,
		})
	}

	// Get unread count
	countQuery := notifapp.CountUnreadQuery{UserID: userID}
	countResult, err := h.notificationService.CountUnread(c.Request().Context(), countQuery)
	if err != nil {
		h.logger.Error("failed to count unread notifications", slog.String("error", err.Error()))
		countResult = notifapp.CountResult{Count: 0}
	}

	notifications := make([]NotificationViewData, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notifications = append(notifications, h.toNotificationViewData(n))
	}

	data := NotificationListData{
		Notifications: notifications,
		TotalCount:    result.TotalCount,
		UnreadCount:   countResult.Count,
		HasMore:       len(result.Notifications) < result.TotalCount,
	}

	return h.renderPartial(c, "notification/dropdown-content", data)
}

// NotificationCountPartial returns the notification badge count as HTML partial.
func (h *NotificationTemplateHandler) NotificationCountPartial(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return h.renderPartial(c, "notification/badge-count", map[string]int{"Count": 0})
	}

	query := notifapp.CountUnreadQuery{UserID: userID}
	result, err := h.notificationService.CountUnread(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to count unread notifications", slog.String("error", err.Error()))
		result = notifapp.CountResult{Count: 0}
	}

	return h.renderPartial(c, "notification/badge-count", map[string]int{"Count": result.Count})
}

// NotificationsListPartial returns the notification list as HTML partial for HTMX.
func (h *NotificationTemplateHandler) NotificationsListPartial(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// Parse query parameters
	limit, offset := h.parseNotificationPagination(c)
	filter := c.QueryParam("filter")
	unreadOnly := filter == "unread"

	query := notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		Limit:      limit,
		Offset:     offset,
	}

	result, err := h.notificationService.ListNotifications(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to list notifications", slog.String("error", err.Error()))
		return h.renderPartial(c, "notification/list-partial", NotificationListData{
			Notifications: []NotificationViewData{},
		})
	}

	notifications := make([]NotificationViewData, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notifications = append(notifications, h.toNotificationViewData(n))
	}

	data := NotificationListData{
		Notifications: notifications,
		TotalCount:    result.TotalCount,
		HasMore:       offset+len(notifications) < result.TotalCount,
		NextOffset:    offset + len(notifications),
		Filter:        filter,
	}

	return h.renderPartial(c, "notification/list-partial", data)
}

// NotificationRedirect marks a notification as read and redirects to the related resource.
func (h *NotificationTemplateHandler) NotificationRedirect(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return c.Redirect(http.StatusFound, "/login")
	}

	notificationID, err := uuid.ParseUUID(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/notifications")
	}

	// Get the notification
	notif, err := h.notificationService.GetNotification(c.Request().Context(), notificationID, userID)
	if err != nil {
		h.logger.Error("failed to get notification",
			slog.String("notification_id", notificationID.String()),
			slog.String("error", err.Error()))
		return c.Redirect(http.StatusFound, "/notifications")
	}

	// Mark as read if not already read
	if !notif.IsRead() {
		cmd := notifapp.MarkAsReadCommand{
			NotificationID: notificationID,
			UserID:         userID,
		}
		_, markErr := h.notificationService.MarkAsRead(c.Request().Context(), cmd)
		if markErr != nil {
			h.logger.Warn("failed to mark notification as read",
				slog.String("notification_id", notificationID.String()),
				slog.String("error", markErr.Error()))
		}
	}

	// Redirect to the resource
	link := h.generateNotificationLink(notif.Type(), notif.ResourceID())
	if link == "" {
		link = "/notifications"
	}

	// For HTMX requests, use HX-Redirect header
	if c.Request().Header.Get("Hx-Request") == "true" {
		c.Response().Header().Set("Hx-Redirect", link)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, link)
}

// Helper methods

// render renders a full page template with common page data.
func (h *NotificationTemplateHandler) render(c echo.Context, templateName, title string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	pageData := PageData{
		Title: title,
		User:  h.getUserView(c),
		Flash: nil,
		Data:  data,
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, pageData, c)
}

// renderPartial renders a template partial for HTMX requests.
func (h *NotificationTemplateHandler) renderPartial(c echo.Context, templateName string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, data, c)
}

// getUserView extracts user information from the context for templates.
func (h *NotificationTemplateHandler) getUserView(c echo.Context) *UserView {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return nil
	}

	user := c.Get("user")
	if user == nil {
		return &UserView{ID: userID.String()}
	}

	if userMap, ok := user.(map[string]any); ok {
		return &UserView{
			ID:          getString(userMap, "id"),
			Email:       getString(userMap, "email"),
			Username:    getString(userMap, "username"),
			DisplayName: getString(userMap, "display_name"),
			AvatarURL:   getString(userMap, "avatar_url"),
		}
	}

	return &UserView{ID: userID.String()}
}

// toNotificationViewData converts a domain notification to template view data.
func (h *NotificationTemplateHandler) toNotificationViewData(n *notification.Notification) NotificationViewData {
	return NotificationViewData{
		ID:         n.ID().String(),
		Type:       string(n.Type()),
		Title:      n.Title(),
		Message:    n.Message(),
		IsRead:     n.IsRead(),
		ResourceID: n.ResourceID(),
		Link:       h.generateNotificationLink(n.Type(), n.ResourceID()),
		CreatedAt:  n.CreatedAt(),
		ReadAt:     n.ReadAt(),
	}
}

// generateNotificationLink generates a link based on notification type.
func (h *NotificationTemplateHandler) generateNotificationLink(notifType notification.Type, resourceID string) string {
	if resourceID == "" {
		return ""
	}

	switch notifType {
	case notification.TypeTaskStatusChanged, notification.TypeTaskAssigned, notification.TypeTaskCreated:
		return "/tasks/" + resourceID
	case notification.TypeChatMention, notification.TypeChatMessage:
		return "/chats/" + resourceID
	case notification.TypeWorkspaceInvite:
		return "/workspaces/" + resourceID
	case notification.TypeSystem:
		return "/notifications"
	default:
		return ""
	}
}

// parseNotificationPagination parses pagination parameters from the request.
func (h *NotificationTemplateHandler) parseNotificationPagination(c echo.Context) (int, int) {
	limit := defaultNotificationTemplateListLimit
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxNotificationTemplateListLimit {
				limit = maxNotificationTemplateListLimit
			}
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Support page-based pagination
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			offset = (page - 1) * limit
		}
	}

	return limit, offset
}
