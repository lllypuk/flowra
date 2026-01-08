package notification

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// Result - result операции с notification
type Result struct {
	appcore.Result[*notification.Notification]
}

// ListResult - result операции with списком notifications
type ListResult struct {
	Notifications []*notification.Notification
	TotalCount    int
	Offset        int
	Limit         int
}

// CountResult - result подсчета
type CountResult struct {
	Count int
}
