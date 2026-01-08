package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestGetNotificationUseCase_Execute_Success(t *testing.T) {
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

	useCase := notification.NewGetNotificationUseCase(repo)

	query := notification.GetNotificationQuery{
		NotificationID: notif.ID(),
		UserID:         userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected notification to be returned")
	}

	if result.Value.ID() != notif.ID() {
		t.Errorf("expected notification ID %s, got %s", notif.ID(), result.Value.ID())
	}

	if result.Value.Title() != "Task Assigned" {
		t.Errorf("expected title 'Task Assigned', got %s", result.Value.Title())
	}
}

func TestGetNotificationUseCase_Execute_NotificationNotFound(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewGetNotificationUseCase(repo)

	query := notification.GetNotificationQuery{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected error for notification not found")
	}

	if !errors.Is(err, notification.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got: %v", err)
	}
}

func TestGetNotificationUseCase_Execute_AccessDenied(t *testing.T) {
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

	useCase := notification.NewGetNotificationUseCase(repo)

	query := notification.GetNotificationQuery{
		NotificationID: notif.ID(),
		UserID:         otherUserID, // другой userель
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected access denied error")
	}

	if !errors.Is(err, notification.ErrNotificationAccessDenied) {
		t.Errorf("expected ErrNotificationAccessDenied, got: %v", err)
	}
}

func TestGetNotificationUseCase_Validate_MissingNotificationID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewGetNotificationUseCase(repo)

	query := notification.GetNotificationQuery{
		NotificationID: uuid.UUID(""),
		UserID:         uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing notificationID")
	}
}

func TestGetNotificationUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewGetNotificationUseCase(repo)

	query := notification.GetNotificationQuery{
		NotificationID: uuid.NewUUID(),
		UserID:         uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
