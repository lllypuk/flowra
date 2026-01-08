package message

import (
	"context"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	domainMessage "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MockMessageRepository - мок репозитория сообщений для тестов
type MockMessageRepository struct {
	Messages map[uuid.UUID]*domainMessage.Message
	SaveErr  error
}

// NewMockMessageRepository создает новый мок репозитория
func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		Messages: make(map[uuid.UUID]*domainMessage.Message),
	}
}

// FindByID находит сообщение по ID
func (m *MockMessageRepository) FindByID(_ context.Context, id uuid.UUID) (*domainMessage.Message, error) {
	msg, ok := m.Messages[id]
	if !ok {
		return nil, ErrMessageNotFound
	}
	return msg, nil
}

// FindByChatID находит сообщения по ID чата
func (m *MockMessageRepository) FindByChatID(
	_ context.Context,
	chatID uuid.UUID,
	pagination Pagination,
) ([]*domainMessage.Message, error) {
	var result []*domainMessage.Message
	for _, msg := range m.Messages {
		if msg.ChatID() == chatID {
			result = append(result, msg)
		}
	}

	// Apply pagination
	start := pagination.Offset
	end := pagination.Offset + pagination.Limit

	if start >= len(result) {
		return []*domainMessage.Message{}, nil
	}
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// FindThread находит тред сообщений
func (m *MockMessageRepository) FindThread(
	_ context.Context,
	parentMessageID uuid.UUID,
) ([]*domainMessage.Message, error) {
	result := make([]*domainMessage.Message, 0)
	for _, msg := range m.Messages {
		if msg.ParentMessageID() == parentMessageID {
			result = append(result, msg)
		}
	}
	return result, nil
}

// CountByChatID подсчитывает сообщения в чате
func (m *MockMessageRepository) CountByChatID(_ context.Context, chatID uuid.UUID) (int, error) {
	count := 0
	for _, msg := range m.Messages {
		if msg.ChatID() == chatID {
			count++
		}
	}
	return count, nil
}

// Save сохраняет сообщение
func (m *MockMessageRepository) Save(_ context.Context, msg *domainMessage.Message) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.Messages[msg.ID()] = msg
	return nil
}

// Delete удаляет сообщение
func (m *MockMessageRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.Messages, id)
	return nil
}

// AddReaction добавляет реакцию к сообщению
func (m *MockMessageRepository) AddReaction(
	_ context.Context,
	messageID uuid.UUID,
	emojiCode string,
	userID uuid.UUID,
) error {
	msg, ok := m.Messages[messageID]
	if !ok {
		return ErrMessageNotFound
	}
	_ = msg.AddReaction(userID, emojiCode)
	return nil
}

// RemoveReaction удаляет реакцию с сообщения
func (m *MockMessageRepository) RemoveReaction(
	_ context.Context,
	messageID uuid.UUID,
	emojiCode string,
	userID uuid.UUID,
) error {
	msg, ok := m.Messages[messageID]
	if !ok {
		return ErrMessageNotFound
	}
	return msg.RemoveReaction(userID, emojiCode)
}

// CountThreadReplies подсчитывает ответы в треде
func (m *MockMessageRepository) CountThreadReplies(
	_ context.Context,
	parentMessageID uuid.UUID,
) (int, error) {
	count := 0
	for _, msg := range m.Messages {
		if msg.ParentMessageID() == parentMessageID && !msg.IsDeleted() {
			count++
		}
	}
	return count, nil
}

// GetReactionUsers возвращает пользователей, поставивших реакцию
func (m *MockMessageRepository) GetReactionUsers(
	_ context.Context,
	messageID uuid.UUID,
	emojiCode string,
) ([]uuid.UUID, error) {
	msg, ok := m.Messages[messageID]
	if !ok {
		return nil, ErrMessageNotFound
	}
	var userIDs []uuid.UUID
	for _, r := range msg.Reactions() {
		if r.EmojiCode() == emojiCode {
			userIDs = append(userIDs, r.UserID())
		}
	}
	if userIDs == nil {
		userIDs = make([]uuid.UUID, 0)
	}
	return userIDs, nil
}

// SearchInChat ищет сообщения в чате по тексту
func (m *MockMessageRepository) SearchInChat(
	_ context.Context,
	chatID uuid.UUID,
	query string,
	offset, limit int,
) ([]*domainMessage.Message, error) {
	var result []*domainMessage.Message
	for _, msg := range m.Messages {
		if msg.ChatID() == chatID && !msg.IsDeleted() {
			// Simple contains search
			if contains(msg.Content(), query) {
				result = append(result, msg)
			}
		}
	}

	// Apply pagination
	if offset >= len(result) {
		return []*domainMessage.Message{}, nil
	}
	end := min(offset+limit, len(result))

	return result[offset:end], nil
}

// FindByAuthor находит сообщения автора в чате
func (m *MockMessageRepository) FindByAuthor(
	_ context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	offset, limit int,
) ([]*domainMessage.Message, error) {
	var result []*domainMessage.Message
	for _, msg := range m.Messages {
		if msg.ChatID() == chatID && msg.AuthorID() == authorID && !msg.IsDeleted() {
			result = append(result, msg)
		}
	}

	// Apply pagination
	if offset >= len(result) {
		return []*domainMessage.Message{}, nil
	}
	end := min(offset+limit, len(result))

	return result[offset:end], nil
}

// contains проверяет, содержит ли строка подстроку (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if equalFoldAt(s, substr, i) {
			return true
		}
	}
	return false
}

func equalFoldAt(s, substr string, start int) bool {
	for j := range len(substr) {
		sc := s[start+j]
		tc := substr[j]
		if sc != tc && toLower(sc) != toLower(tc) {
			return false
		}
	}
	return true
}

const uppercaseToLowercaseOffset = 'a' - 'A'

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + uppercaseToLowercaseOffset
	}
	return c
}

// MockChatRepository - мок репозитория чатов для тестов
type MockChatRepository struct {
	Chats map[string]*chatapp.ReadModel
}

// NewMockChatRepository создает новый мок репозитория чатов
func NewMockChatRepository() *MockChatRepository {
	return &MockChatRepository{
		Chats: make(map[string]*chatapp.ReadModel),
	}
}

// FindByID находит чат по ID
func (m *MockChatRepository) FindByID(_ context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error) {
	c, ok := m.Chats[chatID.String()]
	if !ok {
		return nil, ErrChatNotFound
	}
	return c, nil
}

// AddChat добавляет чат с участниками
func (m *MockChatRepository) AddChat(id uuid.UUID, participants []uuid.UUID) {
	var parts []chat.Participant
	for _, pID := range participants {
		parts = append(parts, chat.NewParticipant(pID, chat.RoleMember))
	}
	m.Chats[id.String()] = &chatapp.ReadModel{
		ID:           id,
		Participants: parts,
	}
}

// MockEventBus - мок шины событий для тестов
type MockEventBus struct {
	Published []event.DomainEvent
}

// NewMockEventBus создает новый мок шины событий
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		Published: make([]event.DomainEvent, 0),
	}
}

// Publish публикует событие
func (m *MockEventBus) Publish(_ context.Context, evt event.DomainEvent) error {
	m.Published = append(m.Published, evt)
	return nil
}
