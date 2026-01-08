package notification

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Query базовый interface запросов
type Query interface {
	QueryName() string
}

// GetNotificationQuery - retrieval notification по ID
type GetNotificationQuery struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID // check, that notification принадлежит user
}

func (q GetNotificationQuery) QueryName() string { return "GetNotification" }

// ListNotificationsQuery - list notifications user
type ListNotificationsQuery struct {
	UserID     uuid.UUID
	UnreadOnly bool // filter only unread
	Limit      int
	Offset     int
}

func (q ListNotificationsQuery) QueryName() string { return "ListNotifications" }

// CountUnreadQuery - count unread
type CountUnreadQuery struct {
	UserID uuid.UUID
}

func (q CountUnreadQuery) QueryName() string { return "CountUnread" }
