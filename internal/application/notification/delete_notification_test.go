package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestDeleteNotificationUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating notification
	notif, _ := domainnotification.NewNotification(
		userID,
		domainnotification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to a task",
		uuid.NewUUID().String(),
	)
	repo.Save(context.Background(), notif)

	useCase := notification.NewDeleteNotificationUseCase(repo)

	cmd := notification.DeleteNotificationCommand{
		NotificationID: notif.ID(),
		UserID:         userID,
	}

	// Act
	err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// check, that notification удален
	if len(repo.notifications) != 0 {
		t.Errorf("expected 0 notifications in repository, got %d", len(repo.notifications))
	}
}

func TestDeleteNotificationUseCase_Execute_NotificationNotFound(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewDeleteNotificationUseCase(repo)

	cmd := notification.DeleteNotificationCommand{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.NewUUID(),
	}

	// Act
	err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for notification not found")
	}

	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got: %v", err)
	}
}

func TestDeleteNotificationUseCase_Execute_AccessDenied(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()

	// Creating notification for other user
	notif, _ := domainnotification.NewNotification(
		userID,
		domainnotification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to a task",
		uuid.NewUUID().String(),
	)
	repo.Save(context.Background(), notif)

	useCase := notification.NewDeleteNotificationUseCase(repo)

	cmd := notification.DeleteNotificationCommand{
		NotificationID: notif.ID(),
		UserID:         otherUserID, // другой userель
	}

	// Act
	err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected access denied error")
	}

	if !errors.Is(err, notification.ErrNotificationAccessDenied) {
		t.Errorf("expected ErrNotificationAccessDenied, got: %v", err)
	}

	// check, that notification not удален
	if len(repo.notifications) != 1 {
		t.Errorf("expected 1 notification in repository, got %d", len(repo.notifications))
	}
}

func TestDeleteNotificationUseCase_Validate_MissingNotificationID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewDeleteNotificationUseCase(repo)

	cmd := notification.DeleteNotificationCommand{
		NotificationID: uuid.UUID(""),
		UserID:         uuid.NewUUID(),
	}

	// Act
	err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing notificationID")
	}
}

func TestDeleteNotificationUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewDeleteNotificationUseCase(repo)

	cmd := notification.DeleteNotificationCommand{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.UUID(""),
	}

	// Act
	err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
