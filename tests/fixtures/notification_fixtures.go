package fixtures

import (
	notifapp "github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/notification"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateNotificationCommandBuilder creates builder for CreateNotificationCommand
type CreateNotificationCommandBuilder struct {
	cmd notifapp.CreateNotificationCommand
}

// NewCreateNotificationCommandBuilder creates New builder (accepts domain UUID)
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

// Build returns prepared command
func (b *CreateNotificationCommandBuilder) Build() notifapp.CreateNotificationCommand {
	return b.cmd
}

// MarkAsReadCommandBuilder creates builder for MarkAsReadCommand
type MarkAsReadCommandBuilder struct {
	cmd notifapp.MarkAsReadCommand
}

// NewMarkAsReadCommandBuilder creates New builder (accepts domain UUID)
func NewMarkAsReadCommandBuilder(notificationID domainUUID.UUID, userID domainUUID.UUID) *MarkAsReadCommandBuilder {
	return &MarkAsReadCommandBuilder{
		cmd: notifapp.MarkAsReadCommand{
			NotificationID: notificationID,
			UserID:         userID,
		},
	}
}

// Build returns prepared command
func (b *MarkAsReadCommandBuilder) Build() notifapp.MarkAsReadCommand {
	return b.cmd
}

// DeleteNotificationCommandBuilder creates builder for DeleteNotificationCommand
type DeleteNotificationCommandBuilder struct {
	cmd notifapp.DeleteNotificationCommand
}

// NewDeleteNotificationCommandBuilder creates New builder (accepts domain UUID)
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

// Build returns prepared command
func (b *DeleteNotificationCommandBuilder) Build() notifapp.DeleteNotificationCommand {
	return b.cmd
}
