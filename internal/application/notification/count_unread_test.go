package notification_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestCountUnreadUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Создаем 7 непрочитанных notifications
	for range 7 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	// Создаем 3 прочитанных notifications
	for range 3 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		notif.MarkAsRead()
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewCountUnreadUseCase(repo)

	query := notification.CountUnreadQuery{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 7 {
		t.Errorf("expected 7 unread notifications, got %d", result.Count)
	}
}

func TestCountUnreadUseCase_Execute_NoUnread(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Создаем только прочитанные notifications
	for range 5 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		notif.MarkAsRead()
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewCountUnreadUseCase(repo)

	query := notification.CountUnreadQuery{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("expected 0 unread notifications, got %d", result.Count)
	}
}

func TestCountUnreadUseCase_Execute_NoNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	useCase := notification.NewCountUnreadUseCase(repo)

	query := notification.CountUnreadQuery{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("expected 0 unread notifications, got %d", result.Count)
	}
}

func TestCountUnreadUseCase_Execute_OnlyUserNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()

	// Создаем 4 непрочитанных для user1
	for range 4 {
		notif, _ := domainnotification.NewNotification(
			user1ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	// Создаем 6 непрочитанных для user2
	for range 6 {
		notif, _ := domainnotification.NewNotification(
			user2ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewCountUnreadUseCase(repo)

	query := notification.CountUnreadQuery{
		UserID: user1ID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 4 {
		t.Errorf("expected 4 unread notifications for user1, got %d", result.Count)
	}
}

func TestCountUnreadUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCountUnreadUseCase(repo)

	query := notification.CountUnreadQuery{
		UserID: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
