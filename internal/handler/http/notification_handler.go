package httphandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/application/appcore"
	notifapp "github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants for notification handler.
const (
	defaultNotificationListLimit = 20
	maxNotificationListLimit     = 100
)

// Notification handler errors.
var (
	ErrNotificationNotFound     = errors.New("notification not found")
	ErrNotificationAccessDenied = errors.New("notification access denied")
	ErrNotificationAlreadyRead  = errors.New("notification already marked as read")
)

// NotificationResponse represents a notification in API responses.
type NotificationResponse struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	IsRead     bool    `json:"is_read"`
	ResourceID string  `json:"resource_id,omitempty"`
	Link       string  `json:"link,omitempty"`
	CreatedAt  string  `json:"created_at"`
	ReadAt     *string `json:"read_at,omitempty"`
}

// NotificationListResponse represents a list of notifications in API responses.
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int                    `json:"total"`
	HasMore       bool                   `json:"has_more"`
}

// UnreadCountResponse represents the count of unread notifications.
type UnreadCountResponse struct {
	Count int `json:"count"`
}

// MarkAllReadResponse represents the response after marking all notifications as read.
type MarkAllReadResponse struct {
	MarkedCount int `json:"marked_count"`
}

// NotificationService defines the interface for notification operations.
// Declared on the consumer side per project guidelines.
type NotificationService interface {
	// ListNotifications lists notifications for a user.
	ListNotifications(ctx context.Context, query notifapp.ListNotificationsQuery) (notifapp.ListResult, error)

	// CountUnread counts unread notifications for a user.
	CountUnread(ctx context.Context, query notifapp.CountUnreadQuery) (notifapp.CountResult, error)

	// MarkAsRead marks a notification as read.
	MarkAsRead(ctx context.Context, cmd notifapp.MarkAsReadCommand) (notifapp.Result, error)

	// MarkAllAsRead marks all notifications as read for a user.
	MarkAllAsRead(ctx context.Context, cmd notifapp.MarkAllAsReadCommand) (notifapp.CountResult, error)

	// DeleteNotification deletes a notification.
	DeleteNotification(ctx context.Context, cmd notifapp.DeleteNotificationCommand) error

	// GetNotification gets a notification by ID.
	GetNotification(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) (*notification.Notification, error)
}

// NotificationHandler handles notification-related HTTP requests.
type NotificationHandler struct {
	notificationService NotificationService
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notificationService NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// RegisterRoutes registers notification routes with the router.
func (h *NotificationHandler) RegisterRoutes(r *httpserver.Router) {
	// All notification routes require authentication
	r.Auth().GET("/notifications", h.List)
	r.Auth().GET("/notifications/unread/count", h.UnreadCount)
	r.Auth().PUT("/notifications/:id/read", h.MarkAsRead)
	r.Auth().PUT("/notifications/mark-all-read", h.MarkAllRead)
	r.Auth().DELETE("/notifications/:id", h.Delete)
}

// List handles GET /api/v1/notifications.
// Lists notifications for the current user.
func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	// Parse query parameters
	limit, offset := parseNotificationPagination(c)
	unreadOnly := c.QueryParam("unread_only") == "true"

	query := notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		Limit:      limit,
		Offset:     offset,
	}

	result, err := h.notificationService.ListNotifications(c.Request().Context(), query)
	if err != nil {
		return handleNotificationError(c, err)
	}

	// Build response
	notifications := make([]NotificationResponse, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notifications = append(notifications, ToNotificationResponse(n))
	}

	hasMore := offset+len(notifications) < result.TotalCount

	resp := NotificationListResponse{
		Notifications: notifications,
		Total:         result.TotalCount,
		HasMore:       hasMore,
	}

	return httpserver.RespondOK(c, resp)
}

// UnreadCount handles GET /api/v1/notifications/unread/count.
// Returns the count of unread notifications.
func (h *NotificationHandler) UnreadCount(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	query := notifapp.CountUnreadQuery{
		UserID: userID,
	}

	result, err := h.notificationService.CountUnread(c.Request().Context(), query)
	if err != nil {
		return handleNotificationError(c, err)
	}

	resp := UnreadCountResponse{
		Count: result.Count,
	}

	return httpserver.RespondOK(c, resp)
}

// MarkAsRead handles PUT /api/v1/notifications/:id/read.
// Marks a notification as read.
func (h *NotificationHandler) MarkAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	notificationIDStr := c.Param("id")
	notificationID, parseErr := uuid.ParseUUID(notificationIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_NOTIFICATION_ID", "invalid notification ID format")
	}

	cmd := notifapp.MarkAsReadCommand{
		NotificationID: notificationID,
		UserID:         userID,
	}

	result, err := h.notificationService.MarkAsRead(c.Request().Context(), cmd)
	if err != nil {
		return handleNotificationError(c, err)
	}

	resp := ToNotificationResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// MarkAllRead handles PUT /api/v1/notifications/mark-all-read.
// Marks all notifications as read for the current user.
func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	cmd := notifapp.MarkAllAsReadCommand{
		UserID: userID,
	}

	result, err := h.notificationService.MarkAllAsRead(c.Request().Context(), cmd)
	if err != nil {
		return handleNotificationError(c, err)
	}

	resp := MarkAllReadResponse{
		MarkedCount: result.Count,
	}

	return httpserver.RespondOK(c, resp)
}

// Delete handles DELETE /api/v1/notifications/:id.
// Deletes a notification.
func (h *NotificationHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	notificationIDStr := c.Param("id")
	notificationID, parseErr := uuid.ParseUUID(notificationIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_NOTIFICATION_ID", "invalid notification ID format")
	}

	cmd := notifapp.DeleteNotificationCommand{
		NotificationID: notificationID,
		UserID:         userID,
	}

	err := h.notificationService.DeleteNotification(c.Request().Context(), cmd)
	if err != nil {
		return handleNotificationError(c, err)
	}

	return httpserver.RespondNoContent(c)
}

// Helper functions

func parseNotificationPagination(c echo.Context) (int, int) {
	limit := defaultNotificationListLimit
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxNotificationListLimit {
				limit = maxNotificationListLimit
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

func handleNotificationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, notifapp.ErrNotificationNotFound):
		return httpserver.RespondErrorWithCode(
			c, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "notification not found")
	case errors.Is(err, notifapp.ErrNotificationAccessDenied):
		return httpserver.RespondErrorWithCode(
			c, http.StatusForbidden, "ACCESS_DENIED", "you don't have access to this notification")
	case errors.Is(err, notifapp.ErrNotificationAlreadyRead):
		return httpserver.RespondErrorWithCode(
			c, http.StatusConflict, "ALREADY_READ", "notification is already marked as read")
	default:
		return httpserver.RespondError(c, err)
	}
}

// ToNotificationResponse converts a domain Notification to NotificationResponse.
func ToNotificationResponse(n *notification.Notification) NotificationResponse {
	resp := NotificationResponse{
		ID:         n.ID().String(),
		Type:       string(n.Type()),
		Title:      n.Title(),
		Body:       n.Message(),
		IsRead:     n.IsRead(),
		ResourceID: n.ResourceID(),
		CreatedAt:  n.CreatedAt().Format(time.RFC3339),
	}

	// Generate link based on notification type and resource ID
	if n.ResourceID() != "" {
		resp.Link = generateNotificationLink(n.Type(), n.ResourceID())
	}

	if n.ReadAt() != nil {
		readAtStr := n.ReadAt().Format(time.RFC3339)
		resp.ReadAt = &readAtStr
	}

	return resp
}

// generateNotificationLink generates a link based on notification type.
func generateNotificationLink(notifType notification.Type, resourceID string) string {
	switch notifType {
	case notification.TypeTaskStatusChanged, notification.TypeTaskAssigned, notification.TypeTaskCreated:
		return "/tasks/" + resourceID
	case notification.TypeChatMention, notification.TypeChatMessage:
		return "/chats/" + resourceID
	case notification.TypeWorkspaceInvite:
		return "/workspaces/" + resourceID
	case notification.TypeSystem:
		return "/notifications/" + resourceID
	default:
		return ""
	}
}

// MockNotificationService is a mock implementation of NotificationService for testing.
type MockNotificationService struct {
	notifications map[uuid.UUID]*notification.Notification
	userNotifs    map[uuid.UUID][]*notification.Notification
}

// NewMockNotificationService creates a new mock notification service.
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		notifications: make(map[uuid.UUID]*notification.Notification),
		userNotifs:    make(map[uuid.UUID][]*notification.Notification),
	}
}

// AddNotification adds a notification to the mock service.
func (m *MockNotificationService) AddNotification(n *notification.Notification) {
	m.notifications[n.ID()] = n
	m.userNotifs[n.UserID()] = append(m.userNotifs[n.UserID()], n)
}

// ListNotifications lists notifications in the mock service.
func (m *MockNotificationService) ListNotifications(
	_ context.Context,
	query notifapp.ListNotificationsQuery,
) (notifapp.ListResult, error) {
	notifs := m.userNotifs[query.UserID]
	if notifs == nil {
		notifs = []*notification.Notification{}
	}

	// Filter by unread if requested
	var filtered []*notification.Notification
	for _, n := range notifs {
		if query.UnreadOnly && n.IsRead() {
			continue
		}
		filtered = append(filtered, n)
	}

	total := len(filtered)

	// Apply pagination
	start := query.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + query.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return notifapp.ListResult{
		Notifications: filtered[start:end],
		TotalCount:    total,
		Offset:        query.Offset,
		Limit:         query.Limit,
	}, nil
}

// CountUnread counts unread notifications in the mock service.
func (m *MockNotificationService) CountUnread(
	_ context.Context,
	query notifapp.CountUnreadQuery,
) (notifapp.CountResult, error) {
	notifs := m.userNotifs[query.UserID]
	count := 0
	for _, n := range notifs {
		if !n.IsRead() {
			count++
		}
	}
	return notifapp.CountResult{Count: count}, nil
}

// MarkAsRead marks a notification as read in the mock service.
func (m *MockNotificationService) MarkAsRead(
	_ context.Context,
	cmd notifapp.MarkAsReadCommand,
) (notifapp.Result, error) {
	n, ok := m.notifications[cmd.NotificationID]
	if !ok {
		return notifapp.Result{}, notifapp.ErrNotificationNotFound
	}

	if n.UserID() != cmd.UserID {
		return notifapp.Result{}, notifapp.ErrNotificationAccessDenied
	}

	if n.IsRead() {
		return notifapp.Result{}, notifapp.ErrNotificationAlreadyRead
	}

	_ = n.MarkAsRead()

	return notifapp.Result{
		Result: appcore.Result[*notification.Notification]{Value: n},
	}, nil
}

// MarkAllAsRead marks all notifications as read in the mock service.
func (m *MockNotificationService) MarkAllAsRead(
	_ context.Context,
	cmd notifapp.MarkAllAsReadCommand,
) (notifapp.CountResult, error) {
	notifs := m.userNotifs[cmd.UserID]
	count := 0
	for _, n := range notifs {
		if !n.IsRead() {
			_ = n.MarkAsRead()
			count++
		}
	}
	return notifapp.CountResult{Count: count}, nil
}

// DeleteNotification deletes a notification from the mock service.
func (m *MockNotificationService) DeleteNotification(
	_ context.Context,
	cmd notifapp.DeleteNotificationCommand,
) error {
	n, ok := m.notifications[cmd.NotificationID]
	if !ok {
		return notifapp.ErrNotificationNotFound
	}

	if n.UserID() != cmd.UserID {
		return notifapp.ErrNotificationAccessDenied
	}

	delete(m.notifications, cmd.NotificationID)

	// Remove from user notifications
	userNotifs := m.userNotifs[cmd.UserID]
	for i, notif := range userNotifs {
		if notif.ID() == cmd.NotificationID {
			m.userNotifs[cmd.UserID] = append(userNotifs[:i], userNotifs[i+1:]...)
			break
		}
	}

	return nil
}

// GetNotification gets a notification from the mock service.
func (m *MockNotificationService) GetNotification(
	_ context.Context,
	notificationID uuid.UUID,
	userID uuid.UUID,
) (*notification.Notification, error) {
	n, ok := m.notifications[notificationID]
	if !ok {
		return nil, notifapp.ErrNotificationNotFound
	}

	if n.UserID() != userID {
		return nil, notifapp.ErrNotificationAccessDenied
	}

	return n, nil
}
