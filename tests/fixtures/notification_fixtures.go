package fixtures

import (
	notifapp "github.com/lllypuk/teams-up/internal/application/notification"
	"github.com/lllypuk/teams-up/internal/domain/notification"
	domainUUID "github.com/lllypuk/teams-up/internal/domain/uuid"
)

// CreateNotificationCommandBuilder создает builder для CreateNotificationCommand
type CreateNotificationCommandBuilder struct {
	cmd notifapp.CreateNotificationCommand
}

// NewCreateNotificationCommandBuilder создает новый builder (accepts domain UUID)
func NewCreateNotificationCommandBuilder(userID domainUUID.UUID) *CreateNotificationCommandBuilder {
	return &CreateNotificationCommandBuilder{
		cmd: notifapp.CreateNotificationCommand{
			UserID:     userID,
			Title:      "Test Notification",
			Message:    "This is a test notification",
			Type:       notification.TypeSystem,
			ResourceID: "",
		},
	}
}

// WithTitle устанавливает title
func (b *CreateNotificationCommandBuilder) WithTitle(title string) *CreateNotificationCommandBuilder {
	b.cmd.Title = title
	return b
}

// WithMessage устанавливает message
func (b *CreateNotificationCommandBuilder) WithMessage(message string) *CreateNotificationCommandBuilder {
	b.cmd.Message = message
	return b
}

// WithType устанавливает type
func (b *CreateNotificationCommandBuilder) WithType(notifType notification.Type) *CreateNotificationCommandBuilder {
	b.cmd.Type = notifType
	return b
}

// WithResourceID устанавливает resourceID
func (b *CreateNotificationCommandBuilder) WithResourceID(resourceID string) *CreateNotificationCommandBuilder {
	b.cmd.ResourceID = resourceID
	return b
}

// Build возвращает готовую команду
func (b *CreateNotificationCommandBuilder) Build() notifapp.CreateNotificationCommand {
	return b.cmd
}

// MarkAsReadCommandBuilder создает builder для MarkAsReadCommand
type MarkAsReadCommandBuilder struct {
	cmd notifapp.MarkAsReadCommand
}

// NewMarkAsReadCommandBuilder создает новый builder (accepts domain UUID)
func NewMarkAsReadCommandBuilder(notificationID domainUUID.UUID, userID domainUUID.UUID) *MarkAsReadCommandBuilder {
	return &MarkAsReadCommandBuilder{
		cmd: notifapp.MarkAsReadCommand{
			NotificationID: notificationID,
			UserID:         userID,
		},
	}
}

// Build возвращает готовую команду
func (b *MarkAsReadCommandBuilder) Build() notifapp.MarkAsReadCommand {
	return b.cmd
}

// DeleteNotificationCommandBuilder создает builder для DeleteNotificationCommand
type DeleteNotificationCommandBuilder struct {
	cmd notifapp.DeleteNotificationCommand
}

// NewDeleteNotificationCommandBuilder создает новый builder (accepts domain UUID)
func NewDeleteNotificationCommandBuilder(
	notificationID domainUUID.UUID,
	userID domainUUID.UUID,
) *DeleteNotificationCommandBuilder {
	return &DeleteNotificationCommandBuilder{
		cmd: notifapp.DeleteNotificationCommand{
			NotificationID: notificationID,
			UserID:         userID,
		},
	}
}

// Build возвращает готовую команду
func (b *DeleteNotificationCommandBuilder) Build() notifapp.DeleteNotificationCommand {
	return b.cmd
}
