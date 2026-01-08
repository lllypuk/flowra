package notification

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// Result - result operatsii s notification
type Result struct {
	appcore.Result[*notification.Notification]
}

// ListResult - result operatsii with spiskom notifications
type ListResult struct {
	Notifications []*notification.Notification
	TotalCount    int
	Offset        int
	Limit         int
}

// CountResult - result podscheta
type CountResult struct {
	Count int
}
