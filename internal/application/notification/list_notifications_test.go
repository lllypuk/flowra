package notification_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestListNotificationsUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 10 notifications
	for range 10 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      5,
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Notifications) != 5 {
		t.Errorf("expected 5 notifications, got %d", len(result.Notifications))
	}

	if result.Limit != 5 {
		t.Errorf("expected limit 5, got %d", result.Limit)
	}

	if result.Offset != 0 {
		t.Errorf("expected offset 0, got %d", result.Offset)
	}
}

func TestListNotificationsUseCase_Execute_UnreadOnly(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 5 unread
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

	// Creating 3 прочитанных
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

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: true,
		Limit:      10,
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Notifications) != 5 {
		t.Errorf("expected 5 unread notifications, got %d", len(result.Notifications))
	}

	// check, that all непрочитанные
	for _, notif := range result.Notifications {
		if notif.IsRead() {
			t.Error("expected all notifications to be unread")
		}
	}

	if result.TotalCount != 5 {
		t.Errorf("expected total count 5, got %d", result.TotalCount)
	}
}

func TestListNotificationsUseCase_Execute_Pagination(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 20 notifications
	for range 20 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewListNotificationsUseCase(repo)

	// Первая страница
	query1 := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     0,
	}

	result1, err := useCase.Execute(context.Background(), query1)
	if err != nil {
		t.Fatalf("expected no error for page 1, got: %v", err)
	}

	if len(result1.Notifications) != 10 {
		t.Errorf("expected 10 notifications on page 1, got %d", len(result1.Notifications))
	}

	// Вторая страница
	query2 := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     10,
	}

	result2, err := useCase.Execute(context.Background(), query2)
	if err != nil {
		t.Fatalf("expected no error for page 2, got: %v", err)
	}

	if len(result2.Notifications) != 10 {
		t.Errorf("expected 10 notifications on page 2, got %d", len(result2.Notifications))
	}
}

func TestListNotificationsUseCase_Execute_DefaultLimit(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 60 notifications
	for range 60 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      0, // должен исuserься дефолтный лимит 50
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Limit != 50 {
		t.Errorf("expected default limit 50, got %d", result.Limit)
	}

	if len(result.Notifications) != 50 {
		t.Errorf("expected 50 notifications, got %d", len(result.Notifications))
	}
}

func TestListNotificationsUseCase_Execute_MaxLimit(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	// Creating 200 notifications
	for range 200 {
		notif, _ := domainnotification.NewNotification(
			userID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      150, // greater максимума (100)
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Limit != 50 {
		t.Errorf("expected max limit to be capped at 50 (default), got %d", result.Limit)
	}
}

func TestListNotificationsUseCase_Execute_EmptyResult(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	userID := uuid.NewUUID()

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Notifications) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(result.Notifications))
	}
}

func TestListNotificationsUseCase_Execute_OnlyUserNotifications(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()

	// Creating 5 notifications for user1
	for range 5 {
		notif, _ := domainnotification.NewNotification(
			user1ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	// Creating 3 notifications for user2
	for range 3 {
		notif, _ := domainnotification.NewNotification(
			user2ID,
			domainnotification.TypeTaskAssigned,
			"Task Assigned",
			"You have been assigned to a task",
			uuid.NewUUID().String(),
		)
		repo.Save(context.Background(), notif)
	}

	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     user1ID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     0,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Notifications) != 5 {
		t.Errorf("expected 5 notifications for user1, got %d", len(result.Notifications))
	}

	// check, that all notifications принадлежат user1
	for _, notif := range result.Notifications {
		if notif.UserID() != user1ID {
			t.Errorf("expected notification to belong to user1, got userID %s", notif.UserID())
		}
	}
}

func TestListNotificationsUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewListNotificationsUseCase(repo)

	query := notification.ListNotificationsQuery{
		UserID:     uuid.UUID(""),
		UnreadOnly: false,
		Limit:      10,
		Offset:     0,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}
