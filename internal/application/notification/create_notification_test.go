package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/notification"
	domainnotification "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// mockNotificationRepository - мок репозитория for testing
type mockNotificationRepository struct {
	notifications map[uuid.UUID]*domainnotification.Notification
	saveError     error
	findError     error
}

func newMockNotificationRepository() *mockNotificationRepository {
	return &mockNotificationRepository{
		notifications: make(map[uuid.UUID]*domainnotification.Notification),
	}
}

func (m *mockNotificationRepository) FindByID(
	_ context.Context,
	id uuid.UUID,
) (*domainnotification.Notification, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	if notif, ok := m.notifications[id]; ok {
		return notif, nil
	}
	return nil, errors.New("not found")
}

func (m *mockNotificationRepository) FindByUserID(
	_ context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*domainnotification.Notification, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	var result []*domainnotification.Notification
	for _, notif := range m.notifications {
		if notif.UserID() == userID {
			result = append(result, notif)
		}
	}

	if offset >= len(result) {
		return []*domainnotification.Notification{}, nil
	}

	end := min(offset+limit, len(result))

	return result[offset:end], nil
}

func (m *mockNotificationRepository) FindUnreadByUserID(
	_ context.Context,
	userID uuid.UUID,
	limit int,
) ([]*domainnotification.Notification, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	var result []*domainnotification.Notification
	for _, notif := range m.notifications {
		if notif.UserID() == userID && !notif.IsRead() {
			result = append(result, notif)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockNotificationRepository) CountUnreadByUserID(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, notif := range m.notifications {
		if notif.UserID() == userID && !notif.IsRead() {
			count++
		}
	}
	return count, nil
}

func (m *mockNotificationRepository) Save(_ context.Context, notif *domainnotification.Notification) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.notifications[notif.ID()] = notif
	return nil
}

func (m *mockNotificationRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.notifications, id)
	return nil
}

func (m *mockNotificationRepository) MarkAsRead(_ context.Context, id uuid.UUID) error {
	if notif, ok := m.notifications[id]; ok {
		notif.MarkAsRead()
	}
	return nil
}

func (m *mockNotificationRepository) MarkAllAsRead(_ context.Context, userID uuid.UUID) error {
	for _, notif := range m.notifications {
		if notif.UserID() == userID {
			notif.MarkAsRead()
		}
	}
	return nil
}

func (m *mockNotificationRepository) MarkManyAsRead(_ context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if notif, ok := m.notifications[id]; ok {
			notif.MarkAsRead()
		}
	}
	return nil
}

func (m *mockNotificationRepository) SaveBatch(
	_ context.Context,
	notifications []*domainnotification.Notification,
) error {
	if m.saveError != nil {
		return m.saveError
	}
	for _, notif := range notifications {
		m.notifications[notif.ID()] = notif
	}
	return nil
}

func (m *mockNotificationRepository) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	for id, notif := range m.notifications {
		if notif.UserID() == userID {
			delete(m.notifications, id)
		}
	}
	return nil
}

func (m *mockNotificationRepository) DeleteOlderThan(_ context.Context, before time.Time) (int, error) {
	count := 0
	for id, notif := range m.notifications {
		if notif.CreatedAt().Before(before) {
			delete(m.notifications, id)
			count++
		}
	}
	return count, nil
}

func (m *mockNotificationRepository) DeleteReadOlderThan(_ context.Context, before time.Time) (int, error) {
	count := 0
	for id, notif := range m.notifications {
		if notif.IsRead() && notif.CreatedAt().Before(before) {
			delete(m.notifications, id)
			count++
		}
	}
	return count, nil
}

func (m *mockNotificationRepository) FindByType(
	_ context.Context,
	userID uuid.UUID,
	notificationType domainnotification.Type,
	offset, limit int,
) ([]*domainnotification.Notification, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	var result []*domainnotification.Notification
	for _, notif := range m.notifications {
		if notif.UserID() == userID && notif.Type() == notificationType {
			result = append(result, notif)
		}
	}
	if offset >= len(result) {
		return []*domainnotification.Notification{}, nil
	}
	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

func (m *mockNotificationRepository) FindByResourceID(
	_ context.Context,
	resourceID string,
) ([]*domainnotification.Notification, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	var result []*domainnotification.Notification
	for _, notif := range m.notifications {
		if notif.ResourceID() == resourceID {
			result = append(result, notif)
		}
	}
	return result, nil
}

func (m *mockNotificationRepository) CountByType(
	_ context.Context,
	userID uuid.UUID,
) (map[domainnotification.Type]int, error) {
	result := make(map[domainnotification.Type]int)
	for _, notif := range m.notifications {
		if notif.UserID() == userID {
			result[notif.Type()]++
		}
	}
	return result, nil
}

func TestCreateNotificationUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCreateNotificationUseCase(repo)
	userID := uuid.NewUUID()

	cmd := notification.CreateNotificationCommand{
		UserID:     userID,
		Type:       domainnotification.TypeTaskAssigned,
		Title:      "Task Assigned",
		Message:    "You have been assigned to a task",
		ResourceID: uuid.NewUUID().String(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected notification to be created")
	}

	if result.Value.Type() != cmd.Type {
		t.Errorf("expected type %s, got %s", cmd.Type, result.Value.Type())
	}

	if result.Value.Title() != cmd.Title {
		t.Errorf("expected title %s, got %s", cmd.Title, result.Value.Title())
	}

	if result.Value.Message() != cmd.Message {
		t.Errorf("expected message %s, got %s", cmd.Message, result.Value.Message())
	}

	if result.Value.UserID() != userID {
		t.Errorf("expected userID %s, got %s", userID, result.Value.UserID())
	}

	if result.Value.IsRead() {
		t.Error("expected notification to be unread")
	}

	// check, that notification savен
	if len(repo.notifications) != 1 {
		t.Errorf("expected 1 notification in repository, got %d", len(repo.notifications))
	}
}

func TestCreateNotificationUseCase_Validate_MissingUserID(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCreateNotificationUseCase(repo)

	cmd := notification.CreateNotificationCommand{
		UserID:  uuid.UUID(""),
		Type:    domainnotification.TypeTaskAssigned,
		Title:   "Task Assigned",
		Message: "You have been assigned to a task",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing userID")
	}
}

func TestCreateNotificationUseCase_Validate_InvalidType(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCreateNotificationUseCase(repo)

	cmd := notification.CreateNotificationCommand{
		UserID:  uuid.NewUUID(),
		Type:    domainnotification.Type("invalid.type"),
		Title:   "Task Assigned",
		Message: "You have been assigned to a task",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
}

func TestCreateNotificationUseCase_Validate_MissingTitle(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCreateNotificationUseCase(repo)

	cmd := notification.CreateNotificationCommand{
		UserID:  uuid.NewUUID(),
		Type:    domainnotification.TypeTaskAssigned,
		Title:   "",
		Message: "You have been assigned to a task",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing title")
	}
}

func TestCreateNotificationUseCase_Validate_MissingMessage(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	useCase := notification.NewCreateNotificationUseCase(repo)

	cmd := notification.CreateNotificationCommand{
		UserID:  uuid.NewUUID(),
		Type:    domainnotification.TypeTaskAssigned,
		Title:   "Task Assigned",
		Message: "",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing message")
	}
}

func TestCreateNotificationUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockNotificationRepository()
	repo.saveError = errors.New("database error")
	useCase := notification.NewCreateNotificationUseCase(repo)

	cmd := notification.CreateNotificationCommand{
		UserID:  uuid.NewUUID(),
		Type:    domainnotification.TypeTaskAssigned,
		Title:   "Task Assigned",
		Message: "You have been assigned to a task",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}
}
