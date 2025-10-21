package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/domain/errs"
)

// NotificationData представляет данные уведомления для тестов
type NotificationData struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Message   string
	IsRead    bool
	CreatedAt time.Time
}

// MockNotificationRepository реализует репозиторий уведомлений для тестирования
type MockNotificationRepository struct {
	mu            sync.RWMutex
	notifications map[uuid.UUID]*NotificationData
	calls         map[string]int
}

// NewMockNotificationRepository создает новый mock репозиторий
func NewMockNotificationRepository() *MockNotificationRepository {
	return &MockNotificationRepository{
		notifications: make(map[uuid.UUID]*NotificationData),
		calls:         make(map[string]int),
	}
}

// Load загружает уведомление по ID
func (r *MockNotificationRepository) Load(ctx context.Context, id uuid.UUID) (*NotificationData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Load"]++

	notif, ok := r.notifications[id]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return notif, nil
}

// Save сохраняет уведомление
func (r *MockNotificationRepository) Save(ctx context.Context, notif *NotificationData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++
	r.notifications[notif.ID] = notif

	return nil
}

// FindByUserID находит все уведомления для пользователя
func (r *MockNotificationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*NotificationData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["FindByUserID"]++

	var notifs []*NotificationData
	for _, notif := range r.notifications {
		if notif.UserID == userID {
			notifs = append(notifs, notif)
		}
	}
	return notifs, nil
}

// FindUnreadByUserID находит непрочитанные уведомления для пользователя
func (r *MockNotificationRepository) FindUnreadByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*NotificationData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["FindUnreadByUserID"]++

	var notifs []*NotificationData
	for _, notif := range r.notifications {
		if notif.UserID == userID && !notif.IsRead {
			notifs = append(notifs, notif)
		}
	}
	return notifs, nil
}

// GetAll возвращает все уведомления
func (r *MockNotificationRepository) GetAll() []*NotificationData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var notifs []*NotificationData
	for _, notif := range r.notifications {
		notifs = append(notifs, notif)
	}
	return notifs
}

// GetCallCount возвращает количество вызовов метода
func (r *MockNotificationRepository) GetCallCount(method string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls[method]
}

// Reset очищает все данные
func (r *MockNotificationRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.notifications = make(map[uuid.UUID]*NotificationData)
	r.calls = make(map[string]int)
}
