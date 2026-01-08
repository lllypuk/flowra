package notification_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestMarkAllAsReadUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 5 unread notifications
	for range 5 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	// Creating 2 уже прочитанных notifications
	for range 2 {
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

	useCase := notification.NewMarkAllAsReadUseCase(repo)

	cmd := notification.MarkAllAsReadCommand{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 5 {
		t.Errorf("expected 5 notifications to be marked, got %d", result.Count)
	}

	// Checking, that all notifications теперь прочитаны
	unreadCount, _ := repo.CountUnreadByUserID(context.Background(), userID)
	if unreadCount != 0 {
		t.Errorf("expected 0 unread notifications, got %d", unreadCount)
	}
}

func TestMarkAllAsReadUseCase_Execute_NoUnreadNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating only прочитанные notifications
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

	useCase := notification.NewMarkAllAsReadUseCase(repo)

	cmd := notification.MarkAllAsReadCommand{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("expected 0 notifications to be marked, got %d", result.Count)
	}
}

func TestMarkAllAsReadUseCase_Execute_EmptyNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	useCase := notification.NewMarkAllAsReadUseCase(repo)

	cmd := notification.MarkAllAsReadCommand{
		UserID: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("expected 0 notifications to be marked, got %d", result.Count)
	}
}

func TestMarkAllAsReadUseCase_Execute_OnlyUserNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()

	// Creating 3 notifications for user1
	for range 3 {
		notif, _ := domainnotification.NewNotification(
			user1ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	// Creating 2 notifications for user2
	for range 2 {
		notif, _ := domainnotification.NewNotification(
			user2ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewMarkAllAsReadUseCase(repo)

	cmd := notification.MarkAllAsReadCommand{
		UserID: user1ID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("expected 3 notifications to be marked for user1, got %d", result.Count)
	}

	// Checking, that у user2 остались непрочитанные
	unreadCountUser2, _ := repo.CountUnreadByUserID(context.Background(), user2ID)
	if unreadCountUser2 != 2 {
		t.Errorf("expected 2 unread notifications for user2, got %d", unreadCountUser2)
	}
}

func TestMarkAllAsReadUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewMarkAllAsReadUseCase(repo)

	cmd := notification.MarkAllAsReadCommand{
		UserID: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
