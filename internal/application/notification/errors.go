package notification

import "errors"

var (
	// ErrNotificationNotFound возвращается, когда notification не найден
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrNotificationAccessDenied возвращается, когда пользователь пытается получить доступ к чужому notification
	ErrNotificationAccessDenied = errors.New("notification access denied")

	// ErrInvalidNotificationType возвращается при неверном типе notification
	ErrInvalidNotificationType = errors.New("invalid notification type")

	// ErrNotificationAlreadyRead возвращается при попытке пометить прочитанное notification как прочитанное
	ErrNotificationAlreadyRead = errors.New("notification already marked as read")
)
