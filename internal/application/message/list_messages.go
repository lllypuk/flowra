package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// ListMessagesUseCase обрабатывает получение списка сообщений в чате
type ListMessagesUseCase struct {
	messageRepo message.Repository
}

// NewListMessagesUseCase создает новый ListMessagesUseCase
func NewListMessagesUseCase(messageRepo message.Repository) *ListMessagesUseCase {
	return &ListMessagesUseCase{
		messageRepo: messageRepo,
	}
}

// Execute выполняет получение списка сообщений
func (uc *ListMessagesUseCase) Execute(
	ctx context.Context,
	query ListMessagesQuery,
) (ListResult, error) {
	// Валидация
	if err := uc.validate(&query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Подготовка пагинации
	pagination := message.Pagination{
		Limit:  query.Limit,
		Offset: query.Offset,
	}

	// Загрузка сообщений
	messages, err := uc.messageRepo.FindByChatID(ctx, query.ChatID, pagination)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to find messages: %w", err)
	}

	return ListResult{
		Value: messages,
	}, nil
}

func (uc *ListMessagesUseCase) validate(query *ListMessagesQuery) error {
	if err := shared.ValidateUUID("chatID", query.ChatID); err != nil {
		return err
	}

	// Установка дефолтных значений
	if query.Limit == 0 {
		query.Limit = DefaultLimit
	}
	if query.Limit > MaxLimit {
		query.Limit = MaxLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	return nil
}
