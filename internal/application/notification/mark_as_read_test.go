package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestMarkAsReadUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Создаем notification
	notif, _ := domainnotification.NewNotification(
		userID,
		domainnotification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to a task",
		uuid.NewUUID().String(),
	)
	repo.Save(context.Background(), notif)

	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: notif.ID(),
		UserID:         userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected notification to be returned")
	}

	if !result.Value.IsRead() {
		t.Error("expected notification to be marked as read")
	}

	if result.Value.ReadAt() == nil {
		t.Error("expected readAt to be set")
	}
}

func TestMarkAsReadUseCase_Execute_NotificationNotFound(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for notification not found")
	}

	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got: %v", err)
	}
}

func TestMarkAsReadUseCase_Execute_AccessDenied(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()

	// Создаем notification для другого пользователя
	notif, _ := domainnotification.NewNotification(
		userID,
		domainnotification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to a task",
		uuid.NewUUID().String(),
	)
	repo.Save(context.Background(), notif)

	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: notif.ID(),
		UserID:         otherUserID, // другой пользователь
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected access denied error")
	}

	if !errors.Is(err, notification.ErrNotificationAccessDenied) {
		t.Errorf("expected ErrNotificationAccessDenied, got: %v", err)
	}
}

func TestMarkAsReadUseCase_Execute_AlreadyRead(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Создаем и сразу помечаем как прочитанное
	notif, _ := domainnotification.NewNotification(
		userID,
		domainnotification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to a task",
		uuid.NewUUID().String(),
	)
	notif.MarkAsRead()
	repo.Save(context.Background(), notif)

	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: notif.ID(),
		UserID:         userID,
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for already read notification")
	}

	if !errors.Is(err, notification.ErrNotificationAlreadyRead) {
		t.Errorf("expected ErrNotificationAlreadyRead, got: %v", err)
	}
}

func TestMarkAsReadUseCase_Validate_MissingNotificationID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: uuid.UUID(""),
		UserID:         uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing notificationID")
	}
}

func TestMarkAsReadUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewMarkAsReadUseCase(repo)

	cmd := notification.MarkAsReadCommand{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
