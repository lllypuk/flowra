package notification

import "errors"

var (
	// ErrNotificationNotFound is returned when notification is not found
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrNotificationAccessDenied is returned when user tries to access another user's notification
	ErrNotificationAccessDenied = errors.New("notification access denied")

	// ErrInvalidNotificationType is returned when notification type is invalid
	ErrInvalidNotificationType = errors.New("invalid notification type")

	// ErrNotificationAlreadyRead is returned when trying to mark an already read notification as read
	ErrNotificationAlreadyRead = errors.New("notification already marked as read")
)
