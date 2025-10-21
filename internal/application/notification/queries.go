package notification

import "github.com/flowra/flowra/internal/domain/uuid"

// Query базовый интерфейс запросов
type Query interface {
	QueryName() string
}

// GetNotificationQuery - получение notification по ID
type GetNotificationQuery struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID // проверка, что notification принадлежит пользователю
}

func (q GetNotificationQuery) QueryName() string { return "GetNotification" }

// ListNotificationsQuery - список notifications пользователя
type ListNotificationsQuery struct {
	UserID     uuid.UUID
	UnreadOnly bool // фильтр только непрочитанных
	Limit      int
	Offset     int
}

func (q ListNotificationsQuery) QueryName() string { return "ListNotifications" }

// CountUnreadQuery - количество непрочитанных
type CountUnreadQuery struct {
	UserID uuid.UUID
}

func (q CountUnreadQuery) QueryName() string { return "CountUnread" }
