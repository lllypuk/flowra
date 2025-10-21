package notification

import (
	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/notification"
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
