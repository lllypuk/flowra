package notification

import (
	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// Result - результат операции с notification
type Result struct {
	shared.Result[*notification.Notification]
}

// ListResult - результат операции со списком notifications
type ListResult struct {
	Notifications []*notification.Notification
	TotalCount    int
	Offset        int
	Limit         int
}

// CountResult - результат подсчета
type CountResult struct {
	Count int
}
